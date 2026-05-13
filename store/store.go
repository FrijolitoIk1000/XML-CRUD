package store

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/FrijolitoIk1000/XML-CRUD/model"
)

type Store struct {
	mu   sync.RWMutex
	path string
}

func New(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() ([]model.Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return []model.Item{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var db model.DBStore
	if err := json.NewDecoder(f).Decode(&db); err != nil {
		return []model.Item{}, nil
	}
	return db.Items, nil
}

func (s *Store) Save(items []model.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(model.DBStore{Items: items})
}
