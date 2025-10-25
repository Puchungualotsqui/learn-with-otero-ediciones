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
	fmt.Println("📥 [HandleAdminAssetList] Upload request received")

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

	adminAssetList.AdminAssetList(assets, subject, grade).Render(r.Context(), w)
}

func HandleAdminAssetManageUpload(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminAssetManageUpload] Upload request received")

	fmt.Println("✅ Multipart parsed")
	fmt.Printf("Subject: %q\n", r.FormValue("subject"))
	fmt.Printf("Grade: %q\n", r.FormValue("grade"))
	fmt.Printf("Uploads: %+v\n", r.MultipartForm.File["uploads"])

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	subject := r.FormValue("subject")
	grade := r.FormValue("grade")
	files := r.MultipartForm.File["uploads"]

	if subject == "" || grade == "" {
		http.Error(w, "Missing subject or grade", http.StatusBadRequest)
		fmt.Printf("Missing subject or grade\n")
		return
	}

	for _, f := range files {
		file, err := f.Open()
		if err != nil {
			http.Error(w, "Error opening file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		if _, err := database.CreateAsset(store, storage, subject, grade, f.Filename, file); err != nil {
			fmt.Printf("❌ Failed to create asset: %v\n", err)
			http.Error(w, "Failed to create asset", http.StatusInternalServerError)
			return
		}
	}

	// After upload → re-render list
	assets, err := database.ListByPrefix[models.Asset](store, database.Buckets["assets"], -1, subject, grade)
	if err != nil {
		http.Error(w, "Error listing assets", http.StatusInternalServerError)
		return
	}

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
