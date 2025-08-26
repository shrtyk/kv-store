package tlog

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shrtyk/kv-store/internal/cfg"
	"github.com/shrtyk/kv-store/internal/core/snapshot"
	sl "github.com/shrtyk/kv-store/pkg/logger"
	pb "github.com/shrtyk/kv-store/proto/log_entries/gen"
	"google.golang.org/protobuf/proto"
)

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

	ReadEvents() (<-chan *pb.LogEntry, <-chan error)
	Err() <-chan error
	Close() error
}

type logFile interface {
	Sync() error
	Stat() (os.FileInfo, error)
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Close() error
	Name() string
}

type logger struct {
	fileMu        sync.Mutex
	isSnaphotting atomic.Bool
	snapshotWg    sync.WaitGroup
	writingsWg    sync.WaitGroup

	cfg         *cfg.WalCfg
	log         *slog.Logger
	file        logFile
	events      chan *pb.LogEntry
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
	l.events = make(chan *pb.LogEntry, 16)
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
				e.Id = atomic.AddUint64(&l.lastSeq, 1)

				data, err := proto.Marshal(e)
				if err != nil {
					l.errs <- err
					l.writingsWg.Done()
					return
				}

				l.fileMu.Lock()
				if err := binary.Write(l.file, binary.LittleEndian, uint32(len(data))); err != nil {
					l.fileMu.Unlock()
					l.errs <- err
					l.writingsWg.Done()
					return
				}
				_, writeErr := l.file.Write(data)
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
		l.restoreFromSnapshot(s, snapshotPath)
	}
	atomic.StoreUint64(&l.lastSeq, lastSeqFromSnapshot)

	// Replay WAL entries created after the snapshot
	evs, errs := l.ReadEvents()
	var e *pb.LogEntry
	ok := true
	var replayErr error
	for ok && replayErr == nil {
		select {
		case replayErr, ok = <-errs:
		case e, ok = <-evs:
			if !ok {
				continue
			}
			if e.Id > lastSeqFromSnapshot {
				switch e.Op {
				case pb.OpType_DELETE:
					replayErr = s.Delete(e.Key)
				case pb.OpType_PUT:
					replayErr = s.Put(e.Key, e.GetValue())
				}
			}
		}
	}
	if replayErr != nil && !errors.Is(replayErr, io.EOF) {
		l.log.Error("unexpected error during WAL replay", sl.ErrorAttr(replayErr))
	}
}

func (l *logger) restoreFromSnapshot(s Store, snapPath string) {
	l.log.Debug("restoring from snapshot", slog.String("path", snapPath))
	state, err := l.snapshotter.Restore(snapPath)
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
}

func (l *logger) ReadEvents() (<-chan *pb.LogEntry, <-chan error) {
	outEvent := make(chan *pb.LogEntry)
	outErr := make(chan error, 1)

	go func() {
		defer close(outEvent)
		defer close(outErr)

		reader := bufio.NewReader(l.file)
		for {
			var length uint32
			if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
				if err != io.EOF {
					outErr <- fmt.Errorf("failed to read transaction log header: %w", err)
				}
				return
			}

			data := make([]byte, length)
			if _, err := io.ReadFull(reader, data); err != nil {
				outErr <- fmt.Errorf("failed to read transaction log data: %w", err)
				return
			}

			e := &pb.LogEntry{}
			if err := proto.Unmarshal(data, e); err != nil {
				outErr <- fmt.Errorf("failed to unmarshal transaction log entry: %w", err)
				return
			}

			atomic.StoreUint64(&l.lastSeq, e.Id)
			outEvent <- e
		}
	}()

	return outEvent, outErr
}

func (l *logger) WritePut(key, val string) {
	l.writingsWg.Add(1)
	l.events <- &pb.LogEntry{Op: pb.OpType_PUT, Key: key, Value: &val}
}

func (l *logger) WriteDelete(key string) {
	l.writingsWg.Add(1)
	l.events <- &pb.LogEntry{Op: pb.OpType_DELETE, Key: key}
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

	reader := bufio.NewReader(sourceFile)
	for {
		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			if err == io.EOF {
				break
			}
			return 0, fmt.Errorf("failed to read message length from wal for snapshotting: %w", err)
		}

		data := make([]byte, length)
		if _, err := io.ReadFull(reader, data); err != nil {
			return 0, fmt.Errorf("failed to read message data from wal for snapshotting: %w", err)
		}

		e := &pb.LogEntry{}
		if err := proto.Unmarshal(data, e); err != nil {
			return 0, fmt.Errorf("failed to unmarshal log entry for snapshotting: %w", err)
		}

		if e.Id > lastSeq {
			lastSeq = e.Id
		}

		switch e.Op {
		case pb.OpType_PUT:
			state[e.Key] = e.GetValue()
		case pb.OpType_DELETE:
			delete(state, e.Key)
		}
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
