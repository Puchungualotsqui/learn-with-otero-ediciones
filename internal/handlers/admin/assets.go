package admin

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/internal/render"
	"frontend/storage"
	"frontend/templates/body"
	"frontend/templates/components/admin/adminAssetList"
	"frontend/templates/components/admin/adminAssetManager"
	"frontend/templates/components/panelsContent"
	"frontend/templates/components/pdfViewer/pdfViewerFrame"
	"io"
	"net/http"
	"os"
	"time"
)

func HandleAdminAssetManagerDefault(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminAssetManagerDefault] Request received")

	subjects, err := database.List[models.Subject](store, database.Buckets["subjects"], 200)
	if err != nil {
		http.Error(w, "Error fetching subjects", http.StatusInternalServerError)
		return
	}

	names := make([]string, len(subjects))
	for i, s := range subjects {
		names[i] = s.Name
	}

	render.RenderWithLayout(w, r, panelsContent.PanelsContent(adminAssetManager.AdminAssetManager(names), pdfViewerFrame.PdfViewerFrame("")), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminAssetList(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminAssetList]")

	subject := r.URL.Query().Get("subject")
	grade := r.URL.Query().Get("grade")

	if subject == "" || grade == "" {
		http.Error(w, "Missing subject or grade", http.StatusBadRequest)
		return
	}

	assets, err := database.ListByPrefix[models.Asset](store, database.Buckets["assets"], -1, subject, grade)
	if err != nil {
		http.Error(w, "Error listing assets", http.StatusInternalServerError)
		return
	}

	for _, asset := range assets {
		println("Visibilityprof: ", asset.ProfessorVisibility)
		println("visibilitystudent: ", asset.StudentVisibility)
	}

	adminAssetList.AdminAssetList(assets, subject, grade).Render(r.Context(), w)
}

func HandleAdminAssetManageUpload(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request) {
	fmt.Printf("📥 [START] Async Upload request received. Length: %v\n", r.Header.Get("Content-Length"))

	// Step 1: Parse Form (150MB limit to be safe with overhead)
	if err := r.ParseMultipartForm(150 << 20); err != nil {
		fmt.Printf("❌ [ERROR] ParseMultipartForm: %v\n", err)
		http.Error(w, "Error al procesar el formulario", http.StatusBadRequest)
		return
	}

	subject := r.FormValue("subject")
	grade := r.FormValue("grade")
	studentVis := r.FormValue("studentVisibility") == "on" || r.FormValue("studentVisibility") == "true"
	professorVis := r.FormValue("professorVisibility") == "on" || r.FormValue("professorVisibility") == "true"
	files := r.MultipartForm.File["uploads"]

	if subject == "" || grade == "" {
		http.Error(w, "Faltan datos (materia o grado)", http.StatusBadRequest)
		return
	}

	// Step 2: Save to Temp Storage and Fire Background Tasks
	for i, fHeader := range files {
		// We open the uploaded file
		src, err := fHeader.Open()
		if err != nil {
			fmt.Printf("❌ [FILE-%d] Failed to open uploaded file\n", i)
			continue
		}

		// We create a physical temp file on the VPS disk.
		dst, err := os.CreateTemp("", "otero-upload-*.pdf")
		if err != nil {
			fmt.Printf("❌ [FILE-%d] Failed to create temp file: %v\n", i, err)
			src.Close()
			continue
		}

		// Copy the data to disk immediately
		_, err = io.Copy(dst, src)
		src.Close()
		dst.Seek(0, 0) // Reset pointer to beginning
		tempPath := dst.Name()
		dst.Close()

		// Launch the heavy processing in the background
		go func(index int, path, fileName string) {
			fmt.Printf("🌀 [BG-%d] Starting background processing for: %s\n", index, fileName)

			// Re-open the temp file for the background process
			fileToProcess, err := os.Open(path)
			if err != nil {
				fmt.Printf("❌ [BG-%d] Failed to reopen temp file\n", index)
				os.Remove(path)
				return
			}
			defer fileToProcess.Close()
			defer os.Remove(path) // Cleanup disk when finished

			_, err = database.CreateAsset(store, storage, subject, grade, fileName, studentVis, professorVis, fileToProcess)
			if err != nil {
				fmt.Printf("❌ [BG-%d] Processing failed: %v\n", index, err)
				return
			}
			fmt.Printf("✅ [BG-%d] Successfully finished: %s\n", index, fileName)
		}(i, tempPath, fHeader.Filename)
	}

	// Since this is HTMX, we return a "success" message that replaces the form area.
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
        <div class="p-4 mb-4 text-sm text-blue-800 rounded-lg bg-blue-50 border border-blue-200 animate-pulse">
            <div class="flex items-center">
                <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                <span><strong>Procesando archivos...</strong> Se están aplicando las marcas de agua y subiendo a B2 en segundo plano. Refresca la página en un minuto para ver los cambios.</span>
            </div>
        </div>
    `))
}

func HandleAdminAssetManageDelete(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request) {
	subject := r.FormValue("subject")
	grade := r.FormValue("grade")
	name := r.FormValue("name")

	if subject == "" || grade == "" || name == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s:%s:%s", subject, grade, name)
	filePath := fmt.Sprintf("assets/%s/%s/%s.pdf", subject, grade, name)

	_ = storage.DeleteFile(r.Context(), filePath)
	_ = database.Delete(store, database.Buckets["assets"], key)

	HandleAdminAssetList(store, w, r)
}

func HandleAdminAssetManageVisibility(store *database.Store, w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	subject := r.URL.Query().Get("subject")
	grade := r.URL.Query().Get("grade")
	target := r.URL.Query().Get("target")

	// This is the part that was failing; fetch() will now send "?value=true"
	isVisible := r.URL.Query().Get("value") == "true"

	fmt.Printf("📥 Visibility Update: Name=%s, Target=%s, Visible=%t\n", name, target, isVisible)

	err := database.UpdateAssetVisibility(store, subject, grade, name, target, isVisible)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func HandleAdminAssetView(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminAssetView] Request received")

	subject := r.URL.Query().Get("subject")
	grade := r.URL.Query().Get("grade")
	name := r.URL.Query().Get("name")

	if subject == "" || grade == "" || name == "" {
		fmt.Printf(
			"❌ Missing parameter(s): subject=%q | grade=%q | name=%q\n",
			subject, grade, name,
		)
		http.Error(w, "Missing subject, grade, or name", http.StatusBadRequest)
		return
	}

	asset, err := database.GetWithPrefix[models.Asset](store, database.Buckets["assets"], name, subject, grade)
	if err != nil || asset == nil {
		fmt.Printf("❌ Error getting asset: %v\n", err)
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}

	fmt.Println("original name: ", asset.OriginalName)

	temporaryLink, err := storage.GetTemporaryLink(r.Context(), asset.OriginalName, 3*time.Minute)
	if err != nil {
		fmt.Printf("Error generating temporary link: %v\n", err)
		http.Error(w, "Error generating temporary link", http.StatusInternalServerError)
		return
	}

	fmt.Printf("✅ Temporary link generated: %s\n", temporaryLink)

	pdfViewerFrame.PdfViewerFrame(temporaryLink).Render(r.Context(), w)
}

func HandleAdminAssetRefresh(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminAssetRefresh] Request received")

	go database.RefreshAssets(store, storage)

	fmt.Printf("Assets refreshed\n")
}
