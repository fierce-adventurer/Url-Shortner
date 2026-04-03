package store

import (
	"errors"
	"sync"
)

type URLStore interface {
	Save(shortCode string, originalUrl string) error
	Get(shortCode string) (string, error)
}

type InMemoryStore struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{urls: make(map[string]string)}
}

func (s *InMemoryStore) Save(shortCode string, originalUrl string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[shortCode] = originalUrl
	return nil
}

func (s *InMemoryStore) Get(shortCode string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.urls[shortCode]
	if !ok {
		return "", errors.New("short code not found")
	}
	return url, nil

}
