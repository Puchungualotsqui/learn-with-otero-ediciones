package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

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

func GetWithPrefix[T any](s *Store, bucket []byte, id string, prefixes ...string) (*T, error) {
	var result *T

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		// build key: prefix1:prefix2:...:id
		parts := append(prefixes, id)
		key := []byte(strings.Join(parts, ":"))
		fmt.Printf("🔑 [GetWithPrefix] looking for key=%q\n", key)

		v := b.Get(key)
		if v == nil {
			return fmt.Errorf("record not found for key %s", key)
		}

		var out T
		if err := json.Unmarshal(v, &out); err != nil {
			return err
		}
		result = &out
		return nil
	})

	return result, err
}

func GetManyWithPrefix[T any](s *Store, bucket []byte, ids []string, prefixes ...string) ([]*T, error) {
	results := make([]*T, 0, len(ids)) // preserve order

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		for _, id := range ids {
			// Build key: prefix1:prefix2:...:id
			key := []byte(strings.Join(append(prefixes, id), ":"))

			v := b.Get(key)
			if v == nil {
				// skip silently
				continue
			}

			var out T
			if err := json.Unmarshal(v, &out); err != nil {
				return fmt.Errorf("unmarshal %q: %w", key, err)
			}

			// Keep results in same order as input IDs
			results = append(results, &out)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return results, nil
}

func Exists(s *Store, bucket []byte, key string) (bool, error) {
	var found bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		v := b.Get([]byte(key))
		found = v != nil
		return nil
	})
	return found, err
}

func ExistsWithPrefix(s *Store, bucket []byte, prefixes ...string) bool {
	var found bool
	_ = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		c := b.Cursor()

		prefix := strings.Join(prefixes, ":") + ":"
		p := []byte(prefix)

		k, _ := c.Seek(p)
		if k != nil && bytes.HasPrefix(k, p) {
			found = true
		}
		return nil
	})
	return found
}

func List[T any](s *Store, bucketName []byte) ([]*T, error) {
	var out []*T
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
			uCopy := u
			out = append(out, &uCopy)
			return nil
		})
	})
	return out, err
}

func ListByPrefix[T any](s *Store, bucket []byte, prefixes ...string) ([]*T, error) {
	var results []*T

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		c := b.Cursor()

		// join prefixes into one composite prefix
		prefix := strings.Join(prefixes, ":") + ":"
		p := []byte(prefix)

		for k, v := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, v = c.Next() {
			var u T
			if err := json.Unmarshal(v, &u); err != nil {
				return err
			}
			// ensure a new pointer is appended each iteration
			uCopy := u
			results = append(results, &uCopy)
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

func UpdateWithPrefix[T any](s *Store, bucket []byte, updater func(*T) error, id string, prefixes ...string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		// Build composite key like "prefix1:prefix2:id"
		parts := append(prefixes, id)
		key := []byte(strings.Join(parts, ":"))

		v := b.Get(key)
		if v == nil {
			return fmt.Errorf("record not found for key %s", key)
		}

		var obj T
		if err := json.Unmarshal(v, &obj); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}

		// Apply caller’s logic
		if err := updater(&obj); err != nil {
			return fmt.Errorf("updater error: %w", err)
		}

		data, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("failed to marshal updated object: %w", err)
		}

		if err := b.Put(key, data); err != nil {
			return fmt.Errorf("failed to update record: %w", err)
		}

		return nil
	})
}
