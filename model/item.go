package model

import (
	"strconv"
	"time"
)

type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Subcategory string  `json:"subcategory"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	Unit        string  `json:"unit"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type DBStore struct {
	Items []Item `json:"items"`
}

func NewID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
