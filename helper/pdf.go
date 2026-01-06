package helper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// AddWatermarkToPDF adds a random-angled “NO A LA PIRATERIA” watermark
// diagonally across each page of the PDF and returns the temp output path.
func AddWatermarkToPDF(input io.Reader) (string, error) {
	fmt.Printf("⏱️ [WM-START] Beginning watermark process for new file\n")

	// Save uploaded file to a temp file
	tmpIn, err := os.CreateTemp("", "input-*.pdf")
	if err != nil {
		return "", err
	}
	defer tmpIn.Close()

	if _, err := io.Copy(tmpIn, input); err != nil {
		return "", err
	}
	fmt.Printf("⏱️ [WM-STEP 1] Saved original to temp disk\n")

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("watermarked-%d.pdf", time.Now().UnixNano()))

	onTop := true
	update := false
	rot := 45
	desc := fmt.Sprintf("font:Courier, points:48, col:1 0 0, rot:%d, scale:1.5 abs, op:.35", rot)

	wm, err := api.TextWatermark("NO A LA PIRATERIA", desc, onTop, update, types.POINTS)
	if err != nil {
		return "", fmt.Errorf("failed to create watermark: %w", err)
	}
	fmt.Printf("⏱️ [WM-STEP 2] Watermark object created\n")

	// Apply watermark to all pages
	fmt.Printf("⏱️ [WM-STEP 3] Calling api.AddWatermarksFile (This is the heavy part...)\n")

	// IMPORTANT: Track the time for the actual PDF modification
	heavyStart := time.Now()
	if err := api.AddWatermarksFile(tmpIn.Name(), outPath, nil, wm, nil); err != nil {
		return "", fmt.Errorf("failed to apply watermark: %w", err)
	}

	fmt.Printf("⏱️ [WM-FINISHED] PDF Modification complete. (Heavy logic took %v)\n", time.Since(heavyStart))
	return outPath, nil
}
