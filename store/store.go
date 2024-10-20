package store

import "errors"

type Store[T any] interface {
	Put(key string, value T) error
	Get(key string) (T, error)
	List() ([]T, error)
	Count() (int, error)
}

var (
	ErrNotFound = errors.New("not found")
)
