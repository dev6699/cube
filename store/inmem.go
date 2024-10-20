package store

type InMemoryStore[T any] struct {
	Db map[string]T
}

func NewInMemoryStore[T any]() *InMemoryStore[T] {
	return &InMemoryStore[T]{
		Db: make(map[string]T),
	}
}

func (i *InMemoryStore[T]) Put(key string, value T) error {
	i.Db[key] = value
	return nil
}

func (i *InMemoryStore[T]) Get(key string) (T, error) {
	var v T
	var ok bool
	v, ok = i.Db[key]
	if !ok {
		return v, ErrNotFound
	}
	return v, nil
}

func (i *InMemoryStore[T]) List() ([]T, error) {
	values := []T{}
	for _, v := range i.Db {
		values = append(values, v)
	}
	return values, nil
}

func (i *InMemoryStore[T]) Count() (int, error) {
	return len(i.Db), nil
}
