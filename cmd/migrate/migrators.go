package main

import (
	"fmt"
	olddb "frontend/database"
	"frontend/database/models"
	"frontend/database/sqlite"

	"go.etcd.io/bbolt"
)

func migrateSubjects(oldStore *olddb.Store, newStore *sqlite.Store) error {
	subjects, err := olddb.List[models.Subject](oldStore, olddb.Buckets["subjects"], 0)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(subjects))
	for _, s := range subjects {
		names = append(names, s.Name)
	}

	return newStore.CreateSubjects(names)
}

func migrateUsers(oldStore *olddb.Store, newStore *sqlite.Store) error {
	users, err := olddb.List[models.User](oldStore, olddb.Buckets["users"], 0)
	if err != nil {
		return err
	}

	for _, u := range users {
		if err := newStore.InsertMigratedUser(u); err != nil {
			return fmt.Errorf("user %s: %w", u.Username, err)
		}
	}

	return nil
}

func migrateClasses(oldStore *olddb.Store, newStore *sqlite.Store) error {
	classes, err := olddb.List[models.Class](oldStore, olddb.Buckets["classes"], 0)
	if err != nil {
		return err
	}

	for _, c := range classes {
		if err := newStore.InsertMigratedClass(c); err != nil {
			return fmt.Errorf("class %d: %w", c.Id, err)
		}
	}

	return nil
}

func migrateClassUsers(oldStore *olddb.Store, newStore *sqlite.Store) error {
	classes, err := olddb.List[models.Class](oldStore, olddb.Buckets["classes"], 0)
	if err != nil {
		return err
	}

	for _, c := range classes {
		for _, username := range c.Users {
			_, err := olddb.Get[models.User](oldStore, olddb.Buckets["users"], username)
			if err != nil {
				fmt.Printf("⚠️ stale BBolt class membership: class=%d username=%s not found in users bucket\n", c.Id, username)
				continue
			}

			if err := newStore.AddUserToClass(c.Id, username); err != nil {
				return fmt.Errorf("class %d user %s: %w", c.Id, username, err)
			}
		}
	}

	return nil
}

func migrateAssignments(oldStore *olddb.Store, newStore *sqlite.Store) error {
	classes, err := olddb.List[models.Class](oldStore, olddb.Buckets["classes"], 0)
	if err != nil {
		return err
	}

	for _, c := range classes {
		assignments, err := olddb.ListByPrefix[models.Assignment](oldStore, olddb.Buckets["assignments"], 0, fmt.Sprintf("%d", c.Id))
		if err != nil {
			return fmt.Errorf("class %d assignments: %w", c.Id, err)
		}

		for _, a := range assignments {
			if err := newStore.InsertMigratedAssignment(c.Id, a); err != nil {
				return fmt.Errorf("class %d assignment %d: %w", c.Id, a.Id, err)
			}
		}
	}

	return nil
}

func migrateSubmissions(oldStore *olddb.Store, newStore *sqlite.Store) error {
	classes, err := olddb.List[models.Class](oldStore, olddb.Buckets["classes"], 0)
	if err != nil {
		return err
	}

	for _, c := range classes {
		assignments, err := olddb.ListByPrefix[models.Assignment](oldStore, olddb.Buckets["assignments"], 0, fmt.Sprintf("%d", c.Id))
		if err != nil {
			return err
		}

		for _, a := range assignments {
			subs, err := olddb.ListByPrefix[models.Submission](oldStore, olddb.Buckets["submissions"], 0, fmt.Sprintf("%d", c.Id), fmt.Sprintf("%d", a.Id))
			if err != nil {
				return fmt.Errorf("submissions class %d assignment %d: %w", c.Id, a.Id, err)
			}

			for _, sub := range subs {
				if err := newStore.InsertMigratedSubmission(c.Id, a.Id, sub); err != nil {
					return fmt.Errorf("submission class %d assignment %d user %s: %w", c.Id, a.Id, sub.Username, err)
				}
			}
		}
	}

	return nil
}

func migrateAssets(oldStore *olddb.Store, newStore *sqlite.Store) error {
	for _, subject := range olddb.SubjectsNames {
		for _, grade := range olddb.Grades {
			assets, err := olddb.ListByPrefix[models.Asset](oldStore, olddb.Buckets["assets"], 0, subject, grade)
			if err != nil {
				return fmt.Errorf("assets %s/%s: %w", subject, grade, err)
			}

			for _, a := range assets {
				if err := newStore.InsertMigratedAsset(subject, grade, a); err != nil {
					return fmt.Errorf("asset %s/%s/%s: %w", subject, grade, a.Name, err)
				}
			}
		}
	}

	return nil
}

func migrateSessions(oldStore *olddb.Store, newStore *sqlite.Store) error {
	return oldStore.DB().View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(olddb.Buckets["sessions"])
		if b == nil {
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if err := newStore.InsertMigratedSession(string(k), string(v)); err != nil {
				return fmt.Errorf("session %s: %w", string(k), err)
			}
		}
		return nil
	})
}

func verifyMigration(oldStore *olddb.Store, newStore *sqlite.Store) error {
	oldUsers, err := olddb.List[models.User](oldStore, olddb.Buckets["users"], 0)
	if err != nil {
		return err
	}
	newUsers, err := newStore.CountTable("users")
	if err != nil {
		return err
	}
	fmt.Printf("users: old=%d new=%d\n", len(oldUsers), newUsers)

	oldSubjects, err := olddb.List[models.Subject](oldStore, olddb.Buckets["subjects"], 0)
	if err != nil {
		return err
	}
	newSubjects, err := newStore.CountTable("subjects")
	if err != nil {
		return err
	}
	fmt.Printf("subjects: old=%d new=%d\n", len(oldSubjects), newSubjects)

	oldClasses, err := olddb.List[models.Class](oldStore, olddb.Buckets["classes"], 0)
	if err != nil {
		return err
	}
	newClasses, err := newStore.CountTable("classes")
	if err != nil {
		return err
	}
	fmt.Printf("classes: old=%d new=%d\n", len(oldClasses), newClasses)

	newClassUsers, err := newStore.CountTable("class_users")
	if err != nil {
		return err
	}
	fmt.Printf("class_users: new=%d\n", newClassUsers)

	newAssignments, err := newStore.CountTable("assignments")
	if err != nil {
		return err
	}
	fmt.Printf("assignments: new=%d\n", newAssignments)

	newSubmissions, err := newStore.CountTable("submissions")
	if err != nil {
		return err
	}
	fmt.Printf("submissions: new=%d\n", newSubmissions)

	newAssets, err := newStore.CountTable("assets")
	if err != nil {
		return err
	}
	fmt.Printf("assets: new=%d\n", newAssets)

	newSessions, err := newStore.CountTable("sessions")
	if err != nil {
		return err
	}
	fmt.Printf("sessions: new=%d\n", newSessions)

	return nil
}
