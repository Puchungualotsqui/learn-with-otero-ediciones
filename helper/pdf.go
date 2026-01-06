package helper

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// AddWatermarkToPDF adds a random-angled “NO A LA PIRATERIA” watermark
// diagonally across each page of the PDF and returns the temp output path.
func AddWatermarkToPDF(input io.Reader) (string, error) {
	// 1. Save original to temp
	tmpIn, _ := os.CreateTemp("", "input-*.pdf")
	defer tmpIn.Close()
	io.Copy(tmpIn, input)

	// 2. Define output path
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("watermarked-%d.pdf", time.Now().UnixNano()))

	// 3. Execute QPDF
	// Order is crucial:
	// qpdf [input] --overlay [overlay-file] --repeat=1 -- [output] --linearize
	cmd := exec.Command("qpdf",
		tmpIn.Name(),
		"--overlay", "static/watermark.pdf", "--repeat=1",
		"--", // This terminates the overlay options
		outPath,
		"--linearize",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	fmt.Printf("🎬 [QPDF] Executing multi-page overlay on %s\n", tmpIn.Name())
	err := cmd.Run()
	if err != nil {
		// If it fails again, we catch the exact error message
		return "", fmt.Errorf("qpdf error: %v, stderr: %s", err, stderr.String())
	}

	// Cleanup original temp
	os.Remove(tmpIn.Name())

	return outPath, nil
}
