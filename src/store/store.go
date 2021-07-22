package store

import (
	"git.mills.io/prologic/bitcask"
)

var (
	current Store
)

type Store interface {
	Set(string, string)
	Get(string) (string, bool)
	Close()
}

type bitStore struct {
	db *bitcask.Bitcask
}

func (s *bitStore) Set(key, val string) {
	_ = s.db.Put([]byte(key), []byte(val))
}

func (s *bitStore) Get(key string) (string, bool) {
	val, err := s.db.Get([]byte(key))
	if err == nil {
		return string(val), true
	}
	return "", false
}

func (s *bitStore) Close() {
	s.db.Close()
}

func Current() Store {
	return current
}

func Init() error {
	db, err := bitcask.Open("/tmp/db")
	if err != nil {
		return err
	}

	current = &bitStore{db}
	return nil
}

func Close() {
	if current != nil {
		current.Close()
		current = nil
	}
}
