package store

import (
	"database/sql"
	"sync"

	_ "github.com/lib/pq"

	"github.com/FrijolitoIk1000/XML-CRUD/model"
)

type Store struct {
	mu sync.RWMutex
	db *sql.DB
}

func New(dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	return s, s.migrate()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS items (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		category    TEXT NOT NULL,
		subcategory TEXT,
		quantity    INTEGER NOT NULL DEFAULT 0,
		price       NUMERIC(12,2) NOT NULL DEFAULT 0,
		unit        TEXT,
		created_at  TEXT,
		updated_at  TEXT
	)`)
	return err
}

func (s *Store) Load() ([]model.Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(
		`SELECT id, name, category, COALESCE(subcategory,''), quantity, price, COALESCE(unit,''), COALESCE(created_at,''), COALESCE(updated_at,'')
		 FROM items ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Item
	for rows.Next() {
		var it model.Item
		if err := rows.Scan(&it.ID, &it.Name, &it.Category, &it.Subcategory,
			&it.Quantity, &it.Price, &it.Unit, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if items == nil {
		items = []model.Item{}
	}
	return items, rows.Err()
}

func (s *Store) Save(items []model.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM items`); err != nil {
		return err
	}

	for _, it := range items {
		_, err := tx.Exec(
			`INSERT INTO items (id, name, category, subcategory, quantity, price, unit, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			it.ID, it.Name, it.Category, it.Subcategory,
			it.Quantity, it.Price, it.Unit, it.CreatedAt, it.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
