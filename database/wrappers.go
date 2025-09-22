package database

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

type Store struct {
	db *bbolt.DB
}

func New(path, bucketName string) (*Store, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() {
	s.db.Close()
}

func Save[T any](s *Store, bucket []byte, key string, value T) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), data)
	})
}

func Get[T any](s *Store, bucket []byte, key string) (*T, error) {
	var out T
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		v := b.Get([]byte(key))
		if v == nil {
			return fmt.Errorf("key %s not found", key)
		}
		return json.Unmarshal(v, &out)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func GetWithPrefix[T any](s *Store, bucket []byte, prefix string) ([]T, error) {
	var results []T

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		c := b.Cursor()
		p := []byte(prefix)

		for k, v := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, v = c.Next() {
			var out T
			if err := json.Unmarshal(v, &out); err != nil {
				return err
			}
			results = append(results, out)
		}

		return nil
	})

	return results, err
}

func Exists(s *Store, bucket []byte, key string) bool {
	var found bool
	_ = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		v := b.Get([]byte(key))
		found = v != nil
		return nil
	})
	return found
}

func ExistsWithPrefix(s *Store, bucket []byte, prefixes ...string) bool {
	var found bool
	_ = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		c := b.Cursor()
		for _, prefix := range prefixes {
			p := []byte(prefix)
			k, _ := c.Seek(p)
			if k != nil && bytes.HasPrefix(k, p) {
				found = true
				return nil // stop early, no need to keep searching
			}
		}
		return nil
	})
	return found
}

func List[T any](s *Store, bucketName []byte) ([]T, error) {
	var out []T
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}
		return b.ForEach(func(k, v []byte) error {
			var u T
			if err := json.Unmarshal(v, &u); err != nil {
				return err
			}
			out = append(out, u)
			return nil
		})
	})
	return out, err
}

func ListByPrefix[T any](s *Store, bucket []byte, prefixes ...string) ([]T, error) {
	var results []T

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		c := b.Cursor()

		for _, prefix := range prefixes {
			p := []byte(prefix)

			for k, v := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, v = c.Next() {
				var out T
				if err := json.Unmarshal(v, &out); err != nil {
					return err
				}
				results = append(results, out)
			}
		}
		return nil
	})

	return results, err
}

func Delete(s *Store, bucketName []byte, key string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}
		return b.Delete([]byte(key))
	})
}
