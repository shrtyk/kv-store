package snapshot

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shrtyk/kv-store/internal/cfg"
	"github.com/shrtyk/kv-store/pkg/logger"
	pb "github.com/shrtyk/kv-store/proto/grpc/gen"
	"google.golang.org/protobuf/proto"
)

type Snapshotter interface {
	Create(state map[string]string, lastSeq uint64) (string, error)
	FindLatest() (path string, lastSeq uint64, err error)
	Restore(path string) (map[string]string, error)
}

type FileSnapshotter struct {
	dir          string
	maxSnapshots int
	logger       *slog.Logger
}

func NewFileSnapshotter(cfg *cfg.SnapshotsCfg, l *slog.Logger) *FileSnapshotter {
	return &FileSnapshotter{
		dir:          cfg.SnapshotsDir,
		maxSnapshots: cfg.MaxSnapshotsAmount,
		logger:       l,
	}
}

func (s *FileSnapshotter) Create(state map[string]string, lastSeq uint64) (string, error) {
	fileName := fmt.Sprintf("snapshot.%d.%d.dat", time.Now().UnixNano(), lastSeq)
	filePath := filepath.Join(s.dir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create snapshot file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			s.logger.Warn("failed to close snapshot file", logger.ErrorAttr(err))
		}
	}()

	writer := bufio.NewWriter(file)

	for k, v := range state {
		entry := &pb.Entry{Key: k, Value: v}
		data, err := proto.Marshal(entry)
		if err != nil {
			if rerr := os.Remove(fileName); rerr != nil {
				return "", fmt.Errorf("failed to delete partially written snapshot after marshal error: %w: initial error: %w", rerr, err)
			}
			return "", fmt.Errorf("failed to marshal snapshot entry: %w", err)
		}

		if err := binary.Write(writer, binary.LittleEndian, uint32(len(data))); err != nil {
			if rerr := os.Remove(fileName); rerr != nil {
				return "", fmt.Errorf("failed to delete partially written snapshot after length write error: %w: initial error: %w", rerr, err)
			}
			return "", fmt.Errorf("failed to write entry length to snapshot: %w", err)
		}

		if _, err := writer.Write(data); err != nil {
			if rerr := os.Remove(fileName); rerr != nil {
				return "", fmt.Errorf("failed to delete partially written snapshot after data write error: %w: initial error: %w", rerr, err)
			}
			return "", fmt.Errorf("failed to write entry data to snapshot: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return "", fmt.Errorf("failed to flush snapshot writer: %w", err)
	}

	if err = s.tryCleanupSnapshots(); err != nil {
		s.logger.Warn("failed to cleanup snapshots", logger.ErrorAttr(err))
	}

	return filePath, nil
}

func (s *FileSnapshotter) tryCleanupSnapshots() error {
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("failed to read snapshots directory: %w", err)
	}

	type snapFile struct {
		path      string
		timestamp uint64
	}

	var snaps []snapFile
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasPrefix(name, "snapshot.") || !strings.HasSuffix(name, ".dat") {
			continue
		}

		parts := strings.Split(name, ".")
		if len(parts) != 4 {
			continue
		}

		timestamp, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse: '%s' into uint64: %w", parts[1], err)
		}

		snaps = append(snaps, snapFile{
			path:      filepath.Join(s.dir, name),
			timestamp: timestamp,
		})
	}

	if len(snaps) <= s.maxSnapshots {
		return nil
	}

	sort.Slice(snaps, func(i int, j int) bool {
		return snaps[i].timestamp < snaps[j].timestamp
	})

	// loop over all "old" snapshots covers case with delete failures in last cleanup
	for _, snap := range snaps[:len(snaps)-s.maxSnapshots] {
		s.logger.Debug("deleting old snapshot", slog.String("path", snap.path))
		if err := os.Remove(snap.path); err != nil {
			s.logger.Warn("failed to delete old snapshot", slog.String("path", snap.path), logger.ErrorAttr(err))
		}
	}

	return nil
}

func (s *FileSnapshotter) FindLatest() (path string, lastSeq uint64, err error) {
	var latestTime int64
	var latestSeq uint64
	var latestPath string

	err = filepath.WalkDir(s.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if strings.HasPrefix(d.Name(), "snapshot.") && strings.HasSuffix(d.Name(), ".dat") {
			parts := strings.Split(d.Name(), ".")
			if len(parts) != 4 {
				return nil // Not a valid snapshot file name
			}

			timestamp, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil // Invalid timestamp
			}

			seq, err := strconv.ParseUint(parts[2], 10, 64)
			if err != nil {
				return nil // Invalid sequence number
			}

			if timestamp > latestTime {
				latestTime = timestamp
				latestSeq = seq
				latestPath = path
			}
		}
		return nil
	})

	if err != nil {
		return "", 0, fmt.Errorf("error finding latest snapshot: %w", err)
	}

	if latestPath == "" {
		return "", 0, os.ErrNotExist
	}

	return latestPath, latestSeq, nil
}

func (s *FileSnapshotter) Restore(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			s.logger.Warn("failed to close snapshot file", logger.ErrorAttr(err))
		}
	}()

	state := make(map[string]string)
	reader := bufio.NewReader(file)

	for {
		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read snapshot entry length: %w", err)
		}

		data := make([]byte, length)
		if _, err := io.ReadFull(reader, data); err != nil {
			return nil, fmt.Errorf("failed to read snapshot entry data: %w", err)
		}

		entry := &pb.Entry{}
		if err := proto.Unmarshal(data, entry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal snapshot entry: %w", err)
		}
		state[entry.Key] = entry.Value
	}

	return state, nil
}
