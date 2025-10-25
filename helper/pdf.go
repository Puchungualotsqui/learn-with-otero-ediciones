package helper

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// AddWatermarkToPDF adds a random-angled “NO A LA PIRATERIA” watermark
// diagonally across each page of the PDF and returns the temp output path.
func AddWatermarkToPDF(input io.Reader) (string, error) {
	// Save uploaded file to a temp file
	tmpIn, err := os.CreateTemp("", "input-*.pdf")
	if err != nil {
		return "", err
	}
	defer tmpIn.Close()

	if _, err := io.Copy(tmpIn, input); err != nil {
		return "", err
	}

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("watermarked-%d.pdf", time.Now().UnixNano()))

	onTop := true
	update := false

	rot := rand.Intn(60) - 30 // between -30 and +30
	desc := fmt.Sprintf("font:Courier, points:48, col:1 0 0, rot:%d, scale:0.9 abs, op:.1", rot)

	wm, err := api.TextWatermark("NO A LA PIRATERIA", desc, onTop, update, types.POINTS)
	if err != nil {
		return "", fmt.Errorf("failed to create watermark: %w", err)
	}

	// Apply watermark to all pages
	if err := api.AddWatermarksFile(tmpIn.Name(), outPath, nil, wm, nil); err != nil {
		return "", fmt.Errorf("failed to apply watermark: %w", err)
	}

	return outPath, nil
}
