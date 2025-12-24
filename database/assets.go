package database

import (
	"context"
	"encoding/json"
	"fmt"
	"frontend/database/models"
	"frontend/helper"
	"frontend/storage"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"go.etcd.io/bbolt"
)

func CreateAsset(s *Store, storage *storage.B2Storage, subject, grade, fileName string, studentVisibility, ProfessorVisibility bool, file io.Reader) (*models.Asset, error) {
	// Step 1. Apply watermark
	watermarkedPath, err := helper.AddWatermarkToPDF(file)
	if err != nil {
		return nil, fmt.Errorf("failed to watermark PDF: %w", err)
	}
	defer os.Remove(watermarkedPath)

	// Step 2. Reopen the processed (watermarked) file
	f, err := os.Open(watermarkedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open watermarked PDF: %w", err)
	}
	defer f.Close()

	// Step 3. Normalize and ensure correct naming
	safeName := helper.NormalizeFilename(fileName)
	if slash := strings.LastIndex(safeName, "/"); slash != -1 {
		safeName = safeName[slash+1:]
	}
	if !strings.HasSuffix(strings.ToLower(safeName), ".pdf") {
		safeName += ".pdf"
	}

	// Step 4. Compose storage and database keys
	storageKey := fmt.Sprintf("recursos/%s/%s/%s", subject, grade, safeName)
	dbKey := fmt.Sprintf("%s:%s:%s", subject, grade, strings.TrimSuffix(safeName, ".pdf"))
	originalName := storageKey // keep path for future temporary links

	// ✅ Step 5. Upload the *watermarked* file, not the original
	url, err := storage.UploadPrivateFile(context.Background(), storageKey, f)
	if err != nil {
		return nil, fmt.Errorf("failed to upload asset to storage: %w", err)
	}

	// Step 6. Save metadata
	asset := &models.Asset{
		Name:                strings.TrimSuffix(safeName, ".pdf"),
		OriginalName:        originalName,
		Url:                 url,
		StudentVisibility:   studentVisibility,
		ProfessorVisibility: ProfessorVisibility,
	}

	err = s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Buckets["assets"])
		if err != nil {
			return err
		}
		data, err := json.Marshal(asset)
		if err != nil {
			return err
		}
		return b.Put([]byte(dbKey), data)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save asset metadata: %w", err)
	}

	fmt.Printf("✅ Asset uploaded and registered: %s (%s/%s)\n", asset.OriginalName, subject, grade)
	return asset, nil
}

func CreateAssetFromURL(s *Store, subject, grade, name, url string, studentVisibility, professorVisibility bool) (*models.Asset, error) {
	asset := &models.Asset{Name: name, OriginalName: name, Url: url, StudentVisibility: studentVisibility, ProfessorVisibility: professorVisibility}
	dbKey := fmt.Sprintf("%s:%s:%s", subject, grade, name)

	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Buckets["assets"])
		if err != nil {
			return err
		}
		data, err := json.Marshal(asset)
		if err != nil {
			return err
		}
		return b.Put([]byte(dbKey), data)
	})
	return asset, err
}

func UpdateAssetVisibility(s *Store, subject, grade, name, target string, isVisible bool) error {
	dbKey := fmt.Sprintf("%s:%s:%s", subject, grade, name)

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(Buckets["assets"])
		if b == nil {
			println("assets bucket not found")
			return fmt.Errorf("assets bucket not found")
		}

		// 1. Get existing data
		data := b.Get([]byte(dbKey))
		if data == nil {
			println("asset not found %s", dbKey)
			return fmt.Errorf("asset not found: %s", dbKey)
		}

		// 2. Unmarshal into struct
		var asset models.Asset
		if err := json.Unmarshal(data, &asset); err != nil {
			println("failed to unmarshal asset: %w", err)
			return fmt.Errorf("failed to unmarshal asset: %w", err)
		}

		// 3. Modify the specific target
		switch target {
		case "student":
			asset.StudentVisibility = isVisible
		case "professor":
			asset.ProfessorVisibility = isVisible
		default:
			println("invalid visibility target: %q", target)
			return fmt.Errorf("invalid visibility target: %q", target)
		}

		// 4. Marshal and Save back
		updatedData, err := json.Marshal(asset)
		if err != nil {
			println("failed to marshal updated asset: %w", err)
			return fmt.Errorf("failed to marshal updated asset: %w", err)
		}

		return b.Put([]byte(dbKey), updatedData)
	})
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
			assets, err := ListByPrefix[models.Asset](store, Buckets["assets"], 200, subjectName, grade)
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
					if _, err := CreateAssetFromURL(store, subjectName, grade, f.FileName, f.DownloadURL, false, false); err != nil {
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
	var validExtensions = []string{".pdf"}
	filtered := make([]*models.Asset, 0, len(assets))

	for _, asset := range assets {
		ext := strings.ToLower(filepath.Ext(asset.OriginalName))
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

		fixed[i] = &models.Asset{Name: name, Url: asset.Url, StudentVisibility: asset.StudentVisibility, ProfessorVisibility: asset.ProfessorVisibility}
	}

	return fixed
}
