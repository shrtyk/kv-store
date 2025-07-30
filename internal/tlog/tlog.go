package tlog

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shrtyk/kv-store/pkg/cfg"
	sl "github.com/shrtyk/kv-store/pkg/logger"
)

type eventType byte

const (
	_                  = iota
	EventPut eventType = iota
	EventDelete
)

type event struct {
	seq   uint64
	event eventType
	key   string
	value string
}

type Store interface {
	Delete(key string) error
	Put(key, val string) error
}

type TransactionsLogger interface {
	Start(ctx context.Context, wg *sync.WaitGroup, s Store)
	Compact()

	WritePut(key, val string)
	WriteDelete(key string)
	ReadEvents() (<-chan event, <-chan error)

	Err() <-chan error
	Close() error
	WaitWritings()
	WaitCompaction()
}

type logger struct {
	fileMu       sync.Mutex
	isCompacting atomic.Bool
	compactWg    sync.WaitGroup
	writingsWg   sync.WaitGroup

	cfg     *cfg.TransLoggerCfg
	log     *slog.Logger
	file    *os.File
	events  chan event
	errs    chan error
	lastSeq uint64
}

func NewFileTransactionalLogger(cfg *cfg.TransLoggerCfg, l *slog.Logger) (*logger, error) {
	file, err := os.OpenFile(cfg.LogFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to open transaction log file: %w", err)
	}

	return &logger{
		cfg:  cfg,
		file: file,
		log:  l,
	}, nil
}

func MustCreateNewFileTransLog(cfg *cfg.TransLoggerCfg, l *slog.Logger) *logger {
	tl, err := NewFileTransactionalLogger(cfg, l)
	if err != nil {
		panic("failed to create new file transaction logger")
	}
	return tl
}

func (l *logger) Start(ctx context.Context, wg *sync.WaitGroup, s Store) {
	l.events = make(chan event, 16)
	l.errs = make(chan error, 1)
	l.restore(s)

	wg.Add(2)
	go l.startFsyncer(ctx, wg)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				l.log.Info("transactional logger shutting down")
				return
			case e := <-l.events:
				l.lastSeq++
				_, err := fmt.Fprintf(l.file, "%d\t%d\t%s\t%s\n", l.lastSeq, e.event, e.key, e.value)
				if err != nil {
					l.errs <- err
					return
				}
				l.writingsWg.Done()
			}
		}
	}()
}

func (l *logger) restore(s Store) {
	evs, errs := l.ReadEvents()

	e := event{}
	var err error
	ok := true
	for ok && err == nil {
		select {
		case err, ok = <-errs:
		case e, ok = <-evs:
			switch e.event {
			case EventDelete:
				err = s.Delete(e.key)
			case EventPut:
				err = s.Put(e.key, e.value)
			}
		}
	}
	if err != nil && err != io.EOF {
		msg := fmt.Sprintf("didn't expect error: %v", err)
		panic(msg)
	}
}

func (l *logger) ReadEvents() (<-chan event, <-chan error) {
	scanner := bufio.NewScanner(l.file)
	outEvent := make(chan event)
	outErr := make(chan error)

	go func() {
		var e event

		defer func() {
			close(outEvent)
			close(outErr)
		}()

		for scanner.Scan() {
			line := scanner.Text()
			_, err := fmt.Sscanf(line, "%d\t%d\t%s\t%s", &e.seq, &e.event, &e.key, &e.value)
			if err != nil {
				outErr <- err
				return
			}

			if l.lastSeq >= e.seq {
				outErr <- errors.New("transaction number out of sequence")
				return
			}

			l.lastSeq = e.seq
			outEvent <- e
		}

		if err := scanner.Err(); err != nil {
			outErr <- fmt.Errorf("failed to read transaction log: %w", err)
			return
		}
	}()

	return outEvent, outErr
}

func (l *logger) WritePut(key, val string) {
	l.writingsWg.Add(1)
	l.events <- event{event: EventPut, key: key, value: val}
}

func (l *logger) WriteDelete(key string) {
	l.writingsWg.Add(1)
	l.events <- event{event: EventDelete, key: key}
}

func (l *logger) Err() <-chan error {
	return l.errs
}

func (l *logger) Close() error {
	l.writingsWg.Wait()
	if l.events != nil {
		close(l.events)
	}
	return l.file.Close()
}

func (l *logger) WaitWritings() {
	l.writingsWg.Wait()
}

func (l *logger) Compact() {
	if !l.isCompacting.CompareAndSwap(false, true) {
		l.log.Info("compcation is already in progress")
		return
	}
	l.compactWg.Add(1)
	go l.runCompactionSupervisor()
}

func (l *logger) runCompactionSupervisor() {
	defer l.compactWg.Done()
	defer l.isCompacting.Store(false)
	l.log.Info("starting compaction supervisor")

	for {
		errs := make(chan error)

		l.compactWg.Add(1)
		go l.runCompaction(errs)

		err := <-errs
		if err != nil {
			l.log.Error("compaction attempt failed, will try again after a delay", sl.ErrorAttr(err))
			time.Sleep(5 * time.Second)
			continue
		}

		l.log.Info("compaction completed, supervisor exiting")
		return
	}
}

func (l *logger) runCompaction(ech chan<- error) {
	defer l.compactWg.Done()
	defer close(ech)

	l.log.Info("transaction log compacter running")
	l.fileMu.Lock()
	curLogName := l.file.Name()
	l.fileMu.Unlock()

	tempLogName := fmt.Sprintf("%s.compacted", curLogName)
	defer os.Remove(tempLogName)

	newSeq, err := l.readAndCompact(curLogName, tempLogName)
	if err != nil {
		ech <- fmt.Errorf("compaction preparation failed: %w", err)
		return
	}

	l.fileMu.Lock()
	defer l.fileMu.Unlock()

	// Wait all writings
	l.WaitWritings()

	// Close old log file
	if err = l.file.Close(); err != nil {
		l.log.Error("failed to close old log file", sl.ErrorAttr(err))
	}

	// Swap new log and old log files
	if err := os.Rename(tempLogName, curLogName); err != nil {
		l.log.Error("failed to rename compacted log file", sl.ErrorAttr(err))
		// Try to recover
		curLogFile, rerr := os.OpenFile(curLogName, os.O_RDWR|os.O_APPEND, 0755)
		if rerr != nil {
			msg := fmt.Sprintf("couldn't rename log and couldn't reopen original log: %v", rerr)
			panic(msg)
		}
		l.file = curLogFile
		ech <- fmt.Errorf("failed to swap logs: %w", err)
		return
	}

	newLogFile, err := os.OpenFile(curLogName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		// Critical failure
		msg := fmt.Sprintf("compaction succeded but couldn't reopen new log file: %v", err)
		panic(msg)
	}

	l.file = newLogFile
	l.lastSeq = newSeq

	l.log.Debug("log file compaction finished successfully")
}

func (l *logger) readAndCompact(sourceName, destName string) (newSeq uint64, err error) {
	sourceFile, err := os.Open(sourceName)
	if err != nil {
		return 0, fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	compactedMap := make(map[string]string)
	s := bufio.NewScanner(sourceFile)
	for s.Scan() {
		var e event
		line := s.Text()
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			l.log.Warn("not enough parts", slog.String("line", line))
			continue
		}

		eType, err := strconv.Atoi(parts[1])
		if err != nil {
			l.log.Warn("invalid event type", slog.String("line", line), sl.ErrorAttr(err))
			continue
		}
		e.event = eventType(eType)
		e.key = parts[2]

		if len(parts) > 3 {
			e.value = parts[3]
		}

		switch e.event {
		case EventPut:
			compactedMap[e.key] = e.value
		case EventDelete:
			delete(compactedMap, e.key)
		}
	}

	if err = s.Err(); err != nil {
		return 0, fmt.Errorf("failed to read source log: %w", err)
	}

	destFile, err := os.OpenFile(destName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to create compacted log: %w", err)
	}
	defer destFile.Close()

	for k, v := range compactedMap {
		newSeq++
		_, err := fmt.Fprintf(destFile, "%d\t%d\t%s\t%s\n", newSeq, EventPut, k, v)
		if err != nil {
			return 0, fmt.Errorf("failed to write to compacted log: %w", err)
		}
	}
	return
}

func (l *logger) WaitCompaction() {
	l.compactWg.Wait()
}

func (l *logger) startFsyncer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	t := time.NewTicker(l.cfg.FsyncIn)
	for {
		select {
		case <-ctx.Done():
			l.log.Info("fsyncer shutting down, starting last fsync")
			l.lastFsyncWithRetries()
			return
		case <-t.C:
			l.fileMu.Lock()
			if err := l.file.Sync(); err != nil {
				l.log.Warn("failed to fsync log file", sl.ErrorAttr(err))
			}
			l.fileMu.Unlock()
		}
	}
}

func (l *logger) lastFsyncWithRetries() {
	for i := range l.cfg.RetriesAmount {
		if err := l.file.Sync(); err != nil {
			tryN := i + 1
			msg := fmt.Sprintf("failed to make last fsync: %d", tryN)
			l.log.Warn(msg, sl.ErrorAttr(err))
			if tryN == l.cfg.RetriesAmount {
				l.log.Error("failed to fsync before full stop. fsyncer stopped")
				return
			}
			time.Sleep(l.cfg.RetryIn)
			continue
		}
		l.log.Info("successfully completed fsync. fsyncer stopped")
		break
	}
}
