package store

import "errors"

var (
	ErrorNoSuchKey = errors.New("no such key")
)

type Store interface {
	Put(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
}

type store struct {
	storage map[string]string
}

func NewStore() Store {
	return &store{
		storage: make(map[string]string),
	}
}

func (s *store) Put(key, value string) error {
	s.storage[key] = value
	return nil
}

func (s *store) Get(key string) (string, error) {
	val, ok := s.storage[key]
	if !ok {
		return "", ErrorNoSuchKey
	}
	return val, nil
}

func (s *store) Delete(key string) error {
	delete(s.storage, key)
	return nil
}
