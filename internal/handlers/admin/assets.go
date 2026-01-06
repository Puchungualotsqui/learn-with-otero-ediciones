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
	"net/http"
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
	fmt.Printf("📥 [START] Synchronous Upload request received. Length: %v\n", r.Header.Get("Content-Length"))

	// Step 1: Parse Form
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

	// Step 2: Process files sequentially
	for i, fHeader := range files {
		fmt.Printf("🚀 [FILE-%d] Processing: %s\n", i, fHeader.Filename)

		file, err := fHeader.Open()
		if err != nil {
			fmt.Printf("❌ [FILE-%d] Error opening: %v\n", i, err)
			continue
		}

		// We call CreateAsset directly (No 'go' keyword)
		_, err = database.CreateAsset(store, storage, subject, grade, fHeader.Filename, studentVis, professorVis, file)
		file.Close()

		if err != nil {
			fmt.Printf("❌ [FILE-%d] CreateAsset Failed: %v\n", i, err)
			// Depending on preference, you can continue or return error
			continue
		}
	}

	// Step 3: Fetch updated list and Re-render
	// This allows HTMX to replace the UI with the actual new data
	assets, err := database.ListByPrefix[models.Asset](store, database.Buckets["assets"], -1, subject, grade)
	if err != nil {
		fmt.Printf("❌ [ERROR] Failed to list assets: %v\n", err)
		http.Error(w, "Error al listar recursos", http.StatusInternalServerError)
		return
	}

	fmt.Println("🎨 [FINISH] Rendering AdminAssetList component")
	adminAssetList.AdminAssetList(assets, subject, grade).Render(r.Context(), w)
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
