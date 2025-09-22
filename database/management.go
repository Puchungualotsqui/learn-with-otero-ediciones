package database

import (
	"encoding/json"
	"fmt"
	"frontend/database/models"
	"frontend/helper"
	"slices"

	"go.etcd.io/bbolt"
)

func AddStudentToClass(s *Store, classId, studentId int) error {
	return updateClass(s, classId, func(c *models.Class) error {
		if !slices.Contains(c.StudentIds, studentId) {
			c.StudentIds = append(c.StudentIds, studentId)
		}
		return nil
	})
}

func RemoveStudentFromClass(s *Store, classId, studentId int) error {
	return updateClass(s, classId, func(c *models.Class) error {
		c.StudentIds = helper.Remove(c.StudentIds, studentId)
		return nil
	})
}

func AddTeacherToClass(s *Store, classId, teacherId int) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(Buckets["classes"])
		if b == nil {
			return fmt.Errorf("bucket %s not found", Buckets["classes"])
		}

		key := fmt.Appendf(nil, "%d", classId)
		v := b.Get(key)
		if v == nil {
			return fmt.Errorf("class %d not found", classId)
		}

		var c models.Class
		if err := json.Unmarshal(v, &c); err != nil {
			return err
		}

		if !slices.Contains(c.TeacherIds, teacherId) {
			c.TeacherIds = append(c.TeacherIds, teacherId)
		}

		data, _ := json.Marshal(c)
		key = fmt.Appendf(nil, "%d", classId)
		return b.Put(key, data)
	})
}

func RemoveTeacherFromClass(s *Store, classId, teacherId int) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(Buckets["classes"])
		if b == nil {
			return fmt.Errorf("bucket %s not found", Buckets["classes"])
		}

		key := fmt.Appendf(nil, "%d", classId)
		v := b.Get(key)
		if v == nil {
			return fmt.Errorf("class %d not found", classId)
		}

		var c models.Class
		if err := json.Unmarshal(v, &c); err != nil {
			return err
		}

		c.TeacherIds = helper.Remove(c.TeacherIds, teacherId)

		data, _ := json.Marshal(c)
		key = fmt.Appendf(nil, "%d", classId)
		return b.Put(key, data)
	})
}
