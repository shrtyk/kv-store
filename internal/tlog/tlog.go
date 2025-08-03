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

	"github.com/shrtyk/kv-store/internal/snapshot"
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
	Items() map[string]string
}

type TransactionsLogger interface {
	Start(ctx context.Context, wg *sync.WaitGroup, s Store)
	WritePut(key, val string)
	WriteDelete(key string)
	WaitWritings()

	ReadEvents() (<-chan event, <-chan error)
	Err() <-chan error
	Close() error
}

type logger struct {
	fileMu        sync.Mutex
	isSnaphotting atomic.Bool
	snapshotWg    sync.WaitGroup
	writingsWg    sync.WaitGroup

	cfg         *cfg.WalCfg
	log         *slog.Logger
	file        *os.File
	events      chan event
	errs        chan error
	lastSeq     uint64
	snapshotter snapshot.Snapshotter
}

func NewFileTransactionalLogger(cfg *cfg.WalCfg, l *slog.Logger, s snapshot.Snapshotter) (*logger, error) {
	file, err := os.OpenFile(cfg.LogFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to open transaction log file: %w", err)
	}

	return &logger{
		cfg:         cfg,
		file:        file,
		log:         l,
		snapshotter: s,
	}, nil
}

func MustCreateNewFileTransLog(cfg *cfg.WalCfg, l *slog.Logger, s snapshot.Snapshotter) *logger {
	tl, err := NewFileTransactionalLogger(cfg, l, s)
	if err != nil {
		msg := fmt.Sprintf("failed to create new file transaction logger: %v", err)
		panic(msg)
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
			case e, ok := <-l.events:
				if !ok {
					return
				}
				newSeq := atomic.AddUint64(&l.lastSeq, 1)

				l.fileMu.Lock()
				_, writeErr := fmt.Fprintf(l.file, "%d\t%d\t%s\t%s\n", newSeq, e.event, e.key, e.value)
				stat, statErr := l.file.Stat()
				l.fileMu.Unlock()

				l.writingsWg.Done()
				if writeErr != nil {
					l.errs <- writeErr
					return
				}

				// Check the file size on each write to trigger snapshotting.
				//
				// IMPORTANT NOTE: This should be efficient enough as file metadata is typically cached by the OS, avoiding disk I/O.
				if statErr != nil {
					l.log.Warn("failed to get WAL file stats", sl.ErrorAttr(statErr))
				} else if stat.Size() >= l.cfg.MaxSizeBytes {
					l.snapshot(s)
				}
			}
		}
	}()
}

func (l *logger) restore(s Store) {
	// Find and restore from the latest snapshot
	snapshotPath, lastSeqFromSnapshot, err := l.snapshotter.FindLatest()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("failed to find latest snapshot: %v", err)
		panic(msg)
	}

	if snapshotPath != "" {
		l.log.Info("restoring from snapshot", slog.String("path", snapshotPath))
		state, err := l.snapshotter.Restore(snapshotPath)
		if err != nil {
			msg := fmt.Sprintf("failed to restore snapshot: %v", err)
			panic(msg)
		}
		for k, v := range state {
			if err := s.Put(k, v); err != nil {
				msg := fmt.Sprintf("failed to apply snapshot entry to store: %v", err)
				panic(msg)
			}
		}
		atomic.StoreUint64(&l.lastSeq, lastSeqFromSnapshot)
	}

	// Replay WAL entries created after the snapshot
	evs, errs := l.ReadEvents()
	e := event{}
	ok := true
	var replayErr error
	for ok && replayErr == nil {
		select {
		case replayErr, ok = <-errs:
		case e, ok = <-evs:
			if !ok {
				continue
			}
			if e.seq > lastSeqFromSnapshot {
				switch e.event {
				case EventDelete:
					replayErr = s.Delete(e.key)
				case EventPut:
					replayErr = s.Put(e.key, e.value)
				}
			}
		}
	}
	if replayErr != nil && !errors.Is(replayErr, io.EOF) {
		l.log.Error("unexpected error during WAL replay", sl.ErrorAttr(replayErr))
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
			parts := strings.Split(line, "\t")
			if len(parts) < 3 {
				l.log.Warn("not enough parts in log line", slog.String("line", line))
				continue
			}
			seq, err := strconv.ParseUint(parts[0], 10, 64)
			if err != nil {
				outErr <- err
				return
			}
			e.seq = seq

			evt, err := strconv.Atoi(parts[1])
			if err != nil {
				outErr <- err
				return
			}
			e.event = eventType(evt)
			e.key = parts[2]
			if len(parts) > 3 {
				e.value = parts[3]
			}

			atomic.StoreUint64(&l.lastSeq, e.seq)
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

func (l *logger) snapshot(s Store) {
	if !l.isSnaphotting.CompareAndSwap(false, true) {
		l.log.Debug("snapshotting is already in progress")
		return
	}
	l.snapshotWg.Add(1)
	go l.runSnapshotSupervisor(s)
}

func (l *logger) runSnapshotSupervisor(s Store) {
	defer l.snapshotWg.Done()
	defer l.isSnaphotting.Store(false)
	l.log.Info("starting snapshot supervisor")

	for {
		errCh := make(chan error, 1)

		l.snapshotWg.Add(1)
		go l.runSnapshotCreation(s, errCh)

		err := <-errCh
		if err != nil {
			l.log.Error("snapshot creation attempt failed, will try again after a delay", sl.ErrorAttr(err))
			time.Sleep(5 * time.Second)
			continue
		}

		l.log.Info("snapshot creation completed, supervisor exiting")
		return
	}
}

func (l *logger) runSnapshotCreation(s Store, ech chan<- error) {
	defer l.snapshotWg.Done()
	defer close(ech)

	l.log.Info("transaction log compaction and snapshotting running")

	latestSnapshotPath, _, err := l.snapshotter.FindLatest()
	var compactedMap map[string]string
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		ech <- fmt.Errorf("failed to find latest snapshot for compaction: %w", err)
		return
	} else if latestSnapshotPath != "" {
		compactedMap, err = l.snapshotter.Restore(latestSnapshotPath)
		if err != nil {
			ech <- fmt.Errorf("failed to restore latest snapshot for compaction: %w", err)
			return
		}
	} else {
		compactedMap = make(map[string]string)
	}

	l.fileMu.Lock()
	curLogName := l.file.Name()
	if err := l.file.Close(); err != nil {
		l.fileMu.Unlock()
		ech <- fmt.Errorf("failed to close current wal for compaction: %w", err)
		return
	}

	compactingLogName := fmt.Sprintf("%s.compacting", curLogName)
	if err := os.Rename(curLogName, compactingLogName); err != nil {
		l.fileMu.Unlock()
		ech <- fmt.Errorf("failed to rename wal for compaction: %w", err)
		l.file, _ = os.OpenFile(curLogName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
		return
	}

	newLogFile, err := os.OpenFile(curLogName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		l.fileMu.Unlock()
		panic(fmt.Sprintf("failed to create new wal file after renaming old: %v", err))
	}
	l.file = newLogFile
	l.fileMu.Unlock()

	lastSeq, err := l.applyLogToState(compactingLogName, compactedMap)
	if err != nil {
		ech <- fmt.Errorf("snapshotting failed during reading log: %w", err)
		return
	}

	if _, err := l.snapshotter.Create(compactedMap, lastSeq); err != nil {
		ech <- fmt.Errorf("failed to create snapshot: %w", err)
		return
	}

	if err := os.Remove(compactingLogName); err != nil {
		l.log.Error("failed to remove old wal", sl.ErrorAttr(err))
	}

	l.log.Debug("snapshotting finished successfully")
}

func (l *logger) applyLogToState(sourceName string, state map[string]string) (lastSeq uint64, err error) {
	sourceFile, err := os.Open(sourceName)
	if err != nil {
		return 0, fmt.Errorf("failed to open wal file for snapshotting: %w", err)
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			l.log.Warn("failed to close source file", sl.ErrorAttr(err))
		}
	}()

	s := bufio.NewScanner(sourceFile)
	for s.Scan() {
		var e event
		line := s.Text()
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			l.log.Warn("not enough parts", slog.String("line", line))
			continue
		}

		seq, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			l.log.Warn("invalid sequence number", slog.String("line", line), sl.ErrorAttr(err))
			continue
		}
		e.seq = seq
		if e.seq > lastSeq {
			lastSeq = e.seq
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
			state[e.key] = e.value
		case EventDelete:
			delete(state, e.key)
		}
	}

	if err = s.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan source log for snapshotting: %w", err)
	}

	return lastSeq, nil
}

func (l *logger) waitSnapshot() {
	l.snapshotWg.Wait()
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
	for i := range l.cfg.FsyncRetriesAmount {
		if err := l.file.Sync(); err != nil {
			tryN := i + 1
			msg := fmt.Sprintf("failed to make last fsync: %d", tryN)
			l.log.Warn(msg, sl.ErrorAttr(err))
			if tryN == l.cfg.FsyncRetriesAmount {
				l.log.Error("failed to fsync before full stop. fsyncer stopped")
				return
			}
			time.Sleep(l.cfg.FsyncRetryIn)
			continue
		}
		l.log.Info("successfully completed fsync. fsyncer stopped")
		break
	}
}
