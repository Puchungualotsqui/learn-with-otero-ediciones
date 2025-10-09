package database

import (
	"encoding/json"
	"fmt"
	"frontend/database/models"
	"frontend/storage"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"go.etcd.io/bbolt"
)

func CreateAsset(s *Store, subject, grade, name, url string) (*models.Asset, error) {
	var sub *models.Asset

	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Buckets["assets"])
		if err != nil {
			return err
		}

		// Composite key: classId:assignmentId:username
		key := fmt.Sprintf("%s:%s:%s", subject, grade, name)

		sub = &models.Asset{
			Name: name,
			Url:  url,
		}

		data, err := json.Marshal(sub)
		if err != nil {
			return err
		}

		return b.Put([]byte(key), data)
	})

	if err != nil {
		return nil, err
	}
	return sub, nil
}

func RefreshAssets(store *Store, storage *storage.B2Storage) {
	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup

	type Job struct {
		Subject string
		Grade   string
	}

	jobs := make([]Job, len(SubjectsNames)*len(Grades))
	for i, _ := range SubjectsNames {
		for j, _ := range Grades {
			jobs[i+j] = Job{SubjectsNames[i], Grades[j]}
		}
	}

	progress := 0
	progressMu := sync.Mutex{}

	for _, job := range jobs {
		wg.Add(1)
		go func(subjectName, grade string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			prefix := fmt.Sprintf("recursos/%s/%s/", subjectName, grade)
			assets, err := ListByPrefix[models.Asset](store, Buckets["assets"], subjectName, grade)
			if err == nil {
				for _, asset := range assets {
					Delete(store, Buckets["assets"], fmt.Sprintf("%s:%s:%s", subjectName, grade, asset.Name))
				}
			}

			files, err := storage.ListFiles(prefix)
			if err != nil {
				fmt.Printf("⚠️ [%s/%s] failed to list files: %v\n", subjectName, grade, err)
			} else {
				for _, f := range files {
					if _, err := CreateAsset(store, subjectName, grade, f.FileName, f.DownloadURL); err != nil {
						fmt.Printf("⚠️ [%s/%s] failed to create asset %s: %v\n", subjectName, grade, f.FileName, err)
					}
				}
			}

			progressMu.Lock()
			progress++
			progressMu.Unlock()
		}(job.Subject, job.Grade)
	}

	wg.Wait()
}

func FilterInvalidAssets(assets []*models.Asset) []*models.Asset {
	validExtensions := []string{".pdf"}
	filtered := make([]*models.Asset, 0, len(assets))

	for _, asset := range assets {
		ext := strings.ToLower(filepath.Ext(asset.Name))
		if slices.Contains(validExtensions, ext) {
			filtered = append(filtered, asset)
		}
	}

	return filtered
}

func FixNames(assets []*models.Asset) []*models.Asset {
	fixed := make([]*models.Asset, len(assets))

	for i, asset := range assets {
		name := asset.Name
		if slash := strings.LastIndex(name, "/"); slash != -1 {
			name = name[slash+1:]
		}
		if dot := strings.LastIndex(name, "."); dot != -1 {
			name = name[:dot]
		}

		fixed[i] = &models.Asset{Name: name, Url: asset.Url}
	}

	return fixed
}
