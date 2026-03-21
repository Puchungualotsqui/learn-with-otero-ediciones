package generics

import (
	"fmt"
	"frontend/database/sqlite"
	"frontend/internal/render"
	"frontend/storage"
	"frontend/templates/body"
	"frontend/templates/components/panelsContent"
	"frontend/templates/components/pdfViewer/pdfViewer"
	"frontend/templates/components/pdfViewer/pdfViewerFrame"
	"net/http"
	"strconv"
	"time"
)

func HandleAssetsDefault(store *sqlite.Store, w http.ResponseWriter, r *http.Request, classId int, professor bool) {
	fmt.Println("📥 [HandleAssetsDefault] Request received")

	classIdString := strconv.Itoa(classId)

	class, err := store.GetClass(classId)
	if err != nil {
		fmt.Printf("Error getting class: %v\n", err)
		http.Error(w, "Error getting class", http.StatusInternalServerError)
		return
	}

	assets, err := store.ListAssets(class.Subject, class.Grade)
	if err != nil {
		fmt.Printf("Error getting assets: %v\n", err)
		http.Error(w, "Error getting assets", http.StatusInternalServerError)
		return
	}

	assets = sqlite.FilterInvalidAssets(assets, professor)

	render.RenderWithLayout(
		w,
		r,
		panelsContent.PanelsContent(pdfViewer.PdfViewer("", assets, classIdString)),
		body.Home,
	)
}

func HandleGetAsset(store *sqlite.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request, classId int, assetName string) {
	fmt.Println("📥 [HandleGetAsset] Request received")

	class, err := store.GetClass(classId)
	if err != nil {
		fmt.Printf("Error getting class: %v\n", err)
		http.Error(w, "Error getting class", http.StatusInternalServerError)
		return
	}

	asset, err := store.GetAsset(class.Subject, class.Grade, assetName)
	if err != nil {
		fmt.Printf("Error getting asset: %v\n", err)
		http.Error(w, "Error getting asset", http.StatusInternalServerError)
		return
	}

	temporaryLink, err := storage.GetTemporaryLink(r.Context(), asset.OriginalName, 3*time.Minute)
	if err != nil {
		fmt.Printf("Error generating temporary link: %v\n", err)
		http.Error(w, "Error generating temporary link", http.StatusInternalServerError)
		return
	}

	pdfViewerFrame.PdfViewerFrame(temporaryLink).Render(r.Context(), w)
}
