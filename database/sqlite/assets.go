package sqlite

import (
	"context"
	"database/sql"
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
	"time"
)

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func intToBool(v int) bool {
	return v != 0
}

func (s *Store) CreateAsset(
	storage *storage.B2Storage,
	subject, grade, fileName string,
	studentVisibility, professorVisibility bool,
	file io.Reader,
) (*models.Asset, error) {
	// Step 1. Apply watermark
	fmt.Printf("🔍 [PROCESS] Applying watermark to %s...\n", fileName)
	watermarkedPath, err := helper.AddWatermarkToPDF(file)
	if err != nil {
		return nil, fmt.Errorf("failed to watermark PDF: %w", err)
	}
	defer os.Remove(watermarkedPath)

	// Step 2. Reopen watermarked file
	f, err := os.Open(watermarkedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open watermarked PDF: %w", err)
	}
	defer f.Close()

	// Step 3. Clean filename
	safeName := helper.NormalizeFilename(fileName)
	if !strings.HasSuffix(strings.ToLower(safeName), ".pdf") {
		safeName += ".pdf"
	}

	// Step 4. Compose storage key
	timestamp := time.Now().Format("20060102150405")
	storageKey := fmt.Sprintf("recursos/%s/%s/%s-%s", subject, grade, timestamp, safeName)

	// Step 5. Upload to B2
	fmt.Printf("☁️ [UPLOAD] Sending %s to B2...\n", safeName)
	url, err := storage.UploadPrivateFile(context.Background(), storageKey, f)
	if err != nil {
		return nil, fmt.Errorf("failed to upload asset to storage: %w", err)
	}

	// Step 6. Save metadata in SQLite
	asset := &models.Asset{
		Name:                strings.TrimSuffix(safeName, ".pdf"),
		OriginalName:        storageKey,
		Url:                 url,
		StudentVisibility:   studentVisibility,
		ProfessorVisibility: professorVisibility,
	}

	_, err = s.DB.Exec(`
		INSERT INTO assets (
			subject,
			grade,
			name,
			original_name,
			url,
			student_visibility,
			professor_visibility
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(subject, grade, name) DO UPDATE SET
			original_name = excluded.original_name,
			url = excluded.url,
			student_visibility = excluded.student_visibility,
			professor_visibility = excluded.professor_visibility
	`,
		subject,
		grade,
		asset.Name,
		asset.OriginalName,
		asset.Url,
		boolToInt(asset.StudentVisibility),
		boolToInt(asset.ProfessorVisibility),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save asset metadata: %w", err)
	}

	fmt.Printf("✅ [FINISH] Asset ready: %s\n", safeName)
	return asset, nil
}

func (s *Store) CreateAssetFromURL(subject, grade, name, url string, studentVisibility, professorVisibility bool) (*models.Asset, error) {
	_, err := s.DB.Exec(`
		INSERT INTO assets (
			subject, grade, name, original_name, url,
			student_visibility, professor_visibility
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(subject, grade, name) DO UPDATE SET
			original_name = excluded.original_name,
			url = excluded.url,
			student_visibility = excluded.student_visibility,
			professor_visibility = excluded.professor_visibility
	`, subject, grade, name, name, url, boolToInt(studentVisibility), boolToInt(professorVisibility))
	if err != nil {
		return nil, err
	}

	return &models.Asset{
		Name:                name,
		OriginalName:        name,
		Url:                 url,
		StudentVisibility:   studentVisibility,
		ProfessorVisibility: professorVisibility,
	}, nil
}

func (s *Store) UpdateAssetVisibility(subject, grade, name, target string, visible bool) error {
	col := ""
	switch target {
	case "student":
		col = "student_visibility"
	case "professor":
		col = "professor_visibility"
	default:
		return nil
	}

	_, err := s.DB.Exec(`
		UPDATE assets
		SET `+col+` = ?
		WHERE subject = ? AND grade = ? AND name = ?
	`, boolToInt(visible), subject, grade, name)

	return err
}

func (s *Store) ListAssets(subject, grade string) ([]*models.Asset, error) {
	rows, err := s.DB.Query(`
		SELECT
			name,
			original_name,
			url,
			student_visibility,
			professor_visibility
		FROM assets
		WHERE subject = ? AND grade = ?
		ORDER BY name ASC
	`, subject, grade)
	if err != nil {
		return nil, fmt.Errorf("query assets: %w", err)
	}
	defer rows.Close()

	assets := make([]*models.Asset, 0)
	for rows.Next() {
		var a models.Asset
		var studentVis int
		var professorVis int

		if err := rows.Scan(
			&a.Name,
			&a.OriginalName,
			&a.Url,
			&studentVis,
			&professorVis,
		); err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}

		a.StudentVisibility = studentVis != 0
		a.ProfessorVisibility = professorVis != 0

		assets = append(assets, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assets: %w", err)
	}

	return assets, nil
}

func (s *Store) GetAsset(subject, grade, name string) (*models.Asset, error) {
	row := s.DB.QueryRow(`
		SELECT
			name,
			original_name,
			url,
			student_visibility,
			professor_visibility
		FROM assets
		WHERE subject = ? AND grade = ? AND name = ?
	`, subject, grade, name)

	var a models.Asset
	var studentVis int
	var professorVis int

	err := row.Scan(
		&a.Name,
		&a.OriginalName,
		&a.Url,
		&studentVis,
		&professorVis,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("asset not found")
		}
		return nil, fmt.Errorf("get asset: %w", err)
	}

	a.StudentVisibility = studentVis != 0
	a.ProfessorVisibility = professorVis != 0

	return &a, nil
}

func (s *Store) DeleteAsset(subject, grade, name string) error {
	res, err := s.DB.Exec(`
		DELETE FROM assets
		WHERE subject = ? AND grade = ? AND name = ?
	`, subject, grade, name)
	if err != nil {
		return fmt.Errorf("delete asset: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return fmt.Errorf("asset not found")
	}

	return nil
}

func (s *Store) RefreshAssets(storage *storage.B2Storage) {
	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup

	type Job struct {
		Subject string
		Grade   string
	}

	jobs := make([]Job, 0, len(SubjectsNames)*len(Grades))
	for _, subject := range SubjectsNames {
		for _, grade := range Grades {
			jobs = append(jobs, Job{
				Subject: subject,
				Grade:   grade,
			})
		}
	}

	progress := 0
	var progressMu sync.Mutex

	for _, job := range jobs {
		wg.Add(1)

		go func(subjectName, grade string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			prefix := fmt.Sprintf("recursos/%s/%s/", subjectName, grade)

			if err := s.DeleteAssetsBySubjectGrade(subjectName, grade); err != nil {
				fmt.Printf("⚠️ [%s/%s] failed to delete existing assets: %v\n", subjectName, grade, err)
			}

			files, err := storage.ListFiles(prefix)
			if err != nil {
				fmt.Printf("⚠️ [%s/%s] failed to list files: %v\n", subjectName, grade, err)
			} else {
				for _, f := range files {
					if _, err := s.CreateAssetFromURL(subjectName, grade, f.FileName, f.DownloadURL, false, false); err != nil {
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

func (s *Store) DeleteAssetsBySubjectGrade(subject, grade string) error {
	_, err := s.DB.Exec(`
		DELETE FROM assets
		WHERE subject = ? AND grade = ?
	`, subject, grade)
	if err != nil {
		return fmt.Errorf("delete assets by subject/grade: %w", err)
	}
	return nil
}

func FilterInvalidAssets(assets []*models.Asset, isProfessor bool) []*models.Asset {
	var validExtensions = []string{".pdf"}
	filtered := make([]*models.Asset, 0, len(assets))

	for _, asset := range assets {
		ext := strings.ToLower(filepath.Ext(asset.OriginalName))

		if !slices.Contains(validExtensions, ext) {
			continue
		}

		isVisible := (isProfessor && asset.ProfessorVisibility) || (!isProfessor && asset.StudentVisibility)
		if isVisible {
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

		fixed[i] = &models.Asset{
			Name:                name,
			Url:                 asset.Url,
			StudentVisibility:   asset.StudentVisibility,
			ProfessorVisibility: asset.ProfessorVisibility,
		}
	}

	return fixed
}
