package store

import (
	"encoding/json"
	"os"

	"github.com/boltdb/bolt"
)

type BoltStore[T any] struct {
	Db     *bolt.DB
	Bucket string
}

func NewBoltStore[T any](file string, mode os.FileMode, bucket string) (*BoltStore[T], error) {
	db, err := bolt.Open(file, mode, nil)
	if err != nil {
		return nil, err
	}

	s := &BoltStore[T]{
		Db:     db,
		Bucket: bucket,
	}

	err = s.createBucket()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *BoltStore[T]) Close() error {
	return s.Db.Close()
}

func (s *BoltStore[T]) createBucket() error {
	return s.Db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(s.Bucket))
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *BoltStore[T]) Put(key string, value T) error {
	return s.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.Bucket))

		buf, err := json.Marshal(value)
		if err != nil {
			return err
		}

		return b.Put([]byte(key), buf)
	})
}

func (s *BoltStore[T]) Get(key string) (T, error) {
	var t T
	err := s.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.Bucket))
		v := b.Get([]byte(key))
		if v == nil {
			return ErrNotFound
		}

		return json.Unmarshal(v, &t)
	})
	return t, err
}

func (s *BoltStore[T]) List() ([]T, error) {
	values := []T{}
	err := s.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.Bucket))
		return b.ForEach(func(k, v []byte) error {
			var t T
			err := json.Unmarshal(v, &t)
			if err != nil {
				return err
			}
			values = append(values, t)
			return nil
		})
	})
	return values, err
}

func (s *BoltStore[T]) Count() (int, error) {
	taskCount := 0
	err := s.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.Bucket))
		return b.ForEach(func(k, v []byte) error {
			taskCount++
			return nil
		})
	})
	if err != nil {
		return -1, err
	}

	return taskCount, nil
}
