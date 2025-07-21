package tlog

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
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
	Start(ctx context.Context, s Store)
	Close() error
	WritePut(key, val string)
	WriteDelete(key string)
	ReadEvents() (<-chan event, <-chan error)
	Err() <-chan error
}

type logger struct {
	file    *os.File
	events  chan<- event
	errs    <-chan error
	lastSeq uint64
}

func NewFileTransactionalLogger(filename string) (TransactionsLogger, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to open transaction log file: %w", err)
	}

	return &logger{file: file}, nil
}

func MustCreateNewFileTransLog(filename string) TransactionsLogger {
	tl, err := NewFileTransactionalLogger(filename)
	if err != nil {
		panic("failed to create new file transaction logger")
	}
	return tl
}

func (l *logger) Start(ctx context.Context, s Store) {
	events := make(chan event, 16)
	l.events = events

	errs := make(chan error, 1)
	l.errs = errs

	l.restore(s)
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Transactional logger shutting down...")
				return
			case e := <-events:
				l.lastSeq++
				_, err := fmt.Fprintf(l.file, "%d\t%d\t%s\t%s\n", l.lastSeq, e.event, e.key, e.value)
				if err != nil {
					errs <- err
					return
				}
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
	if err != nil {
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
	l.events <- event{event: EventPut, key: key, value: val}
}

func (l *logger) WriteDelete(key string) {
	l.events <- event{event: EventDelete, key: key}
}

func (l *logger) Err() <-chan error {
	return l.errs
}

func (l *logger) Close() error {
	return l.file.Close()
}
