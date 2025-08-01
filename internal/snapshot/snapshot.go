package snapshot

import (
	"bufio"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shrtyk/kv-store/pkg/cfg"
	"github.com/shrtyk/kv-store/pkg/logger"
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
	defer file.Close()

	for k, v := range state {
		_, err := fmt.Fprintf(file, "%s\t%s\n", k, v)
		if err != nil {
			if rerr := os.Remove(fileName); rerr != nil {
				return "", fmt.Errorf("failed to delete partially written snapshot: %w: initial error: %w", rerr, err)
			}
			return "", fmt.Errorf("failed to write to snapshot file: %w", err)
		}
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
		s.logger.Info("deleting old snapshot", slog.String("path", snap.path))
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
	defer file.Close()

	state := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			// potential empty lines or malformed data
			s.logger.Warn("malformed data in snapshot file", slog.String("line", line))
			continue
		}
		state[parts[0]] = parts[1]
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read snapshot file: %w", err)
	}

	return state, nil
}
