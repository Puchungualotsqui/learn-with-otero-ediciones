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

func SaveMany[T any](s *Store, bucket []byte, items map[string]T) error {
	if len(items) == 0 {
		return nil // nothing to save
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return fmt.Errorf("creating bucket %s: %w", bucket, err)
		}

		for key, value := range items {
			data, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("failed to marshal key %q: %w", key, err)
			}
			if err := b.Put([]byte(key), data); err != nil {
				return fmt.Errorf("failed to save key %q: %w", key, err)
			}
		}
		return nil
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

func GetMany[T any](s *Store, bucket []byte, keys ...string) ([]*T, error) {
	results := make([]*T, 0, len(keys))

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		for _, key := range keys {
			v := b.Get([]byte(key))
			if v == nil {
				// skip silently or log
				continue
			}

			var obj T
			if err := json.Unmarshal(v, &obj); err != nil {
				return fmt.Errorf("unmarshal %q: %w", key, err)
			}

			objCopy := obj
			results = append(results, &objCopy)
		}

		return nil
	})

	return results, err
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

func ExistsMany(s *Store, bucket []byte, keys []string) (map[string]bool, error) {
	results := make(map[string]bool, len(keys))

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		for _, key := range keys {
			if key == "" {
				results[key] = false
				continue
			}
			v := b.Get([]byte(key))
			results[key] = v != nil
		}
		return nil
	})

	return results, err
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

func List[T any](s *Store, bucketName []byte, limit int) ([]*T, error) {
	var out []*T
	stopErr := fmt.Errorf("stop iteration")

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		count := 0
		return b.ForEach(func(k, v []byte) error {
			var item T
			if err := json.Unmarshal(v, &item); err != nil {
				return err
			}
			copy := item
			out = append(out, &copy)

			if limit >= 0 { // only enforce when non-negative
				count++
				if count >= limit {
					return stopErr // stop iteration manually
				}
			}
			return nil
		})
	})

	// ignore the manual stop signal
	if err == stopErr {
		err = nil
	}

	return out, err
}

func ListByPrefix[T any](s *Store, bucket []byte, limit int, prefixes ...string) ([]*T, error) {
	var results []*T

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		c := b.Cursor()

		// Build composite prefix (if none, list all)
		prefix := ""
		if len(prefixes) > 0 {
			prefix = strings.Join(prefixes, ":") + ":"
		}
		p := []byte(prefix)

		count := 0
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if prefix != "" && !bytes.HasPrefix(k, p) {
				continue
			}

			var u T
			if err := json.Unmarshal(v, &u); err != nil {
				return fmt.Errorf("unmarshal %q: %w", k, err)
			}

			uCopy := u
			results = append(results, &uCopy)

			count++
			if limit > 0 && count >= limit {
				break
			}
		}
		return nil
	})

	return results, err
}

func ListByManyPrefix[T any](s *Store, bucket []byte, limit int, prefixes ...[]string) (map[string][]*T, error) {
	results := make(map[string][]*T)

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		c := b.Cursor()
		count := 0

		for k, v := c.First(); k != nil; k, v = c.Next() {
			keyStr := string(k)
			matchedPrefix := ""

			// Match key against all prefix groups
			for _, group := range prefixes {
				prefix := strings.Join(group, ":")
				if bytes.HasPrefix(k, []byte(prefix)) {
					// Check if exact match (no colon needed)
					// or if the next char is ":" to avoid partial matches
					if len(k) == len(prefix) || k[len(prefix)] == ':' {
						matchedPrefix = prefix
						break
					}
				}
			}

			// If no prefix list provided, everything matches
			if matchedPrefix == "" && len(prefixes) > 0 {
				continue
			}
			if matchedPrefix == "" {
				// Use base key (up to first ':') or full key if none
				parts := strings.SplitN(keyStr, ":", 2)
				matchedPrefix = parts[0]
			}

			var u T
			if err := json.Unmarshal(v, &u); err != nil {
				return fmt.Errorf("unmarshal %q: %w", k, err)
			}

			uCopy := u
			results[matchedPrefix] = append(results[matchedPrefix], &uCopy)

			count++
			if limit > 0 && count >= limit {
				break
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

func DeleteMany(s *Store, bucketName []byte, keys []string) error {
	if len(keys) == 0 {
		return nil // nothing to delete
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		for _, key := range keys {
			if err := b.Delete([]byte(key)); err != nil {
				return fmt.Errorf("failed to delete key %q: %w", key, err)
			}
		}
		return nil
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

func UpdateManyWithPrefix[T any](s *Store, bucket []byte, updater func(*T) error, ids []string, prefixes ...string) error {
	if len(ids) == 0 {
		return nil
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		for _, id := range ids {
			parts := append(prefixes, id)
			key := []byte(strings.Join(parts, ":"))

			v := b.Get(key)
			if v == nil {
				fmt.Printf("⚠️ record not found for key %s, skipping\n", key)
				continue
			}

			var obj T
			if err := json.Unmarshal(v, &obj); err != nil {
				return fmt.Errorf("failed to unmarshal %s: %w", key, err)
			}

			if err := updater(&obj); err != nil {
				return fmt.Errorf("updater error for %s: %w", key, err)
			}

			data, err := json.Marshal(obj)
			if err != nil {
				return fmt.Errorf("failed to marshal %s: %w", key, err)
			}

			if err := b.Put(key, data); err != nil {
				return fmt.Errorf("failed to save %s: %w", key, err)
			}
		}

		return nil
	})
}
