package handlers

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
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

func HandleAssetsDefault(store *database.Store, w http.ResponseWriter, r *http.Request, classId int) {
	fmt.Println("📥 [HandleAssetsDefault] Request received")

	classIdString := strconv.Itoa(classId)
	class, err := database.Get[models.Class](store, database.Buckets["classes"], classIdString)
	if err != nil {
		fmt.Printf("Error getting user: %v \n", err)
	}

	assets, err := database.ListByPrefix[models.Asset](store, database.Buckets["assets"], class.Subject, class.Grade)
	if err != nil {
		fmt.Printf("Error getting assets: %v \n", err)
	}

	assets = database.FilterInvalidAssets(assets)

	render.RenderWithLayout(w, r, panelsContent.PanelsContent(pdfViewer.PdfViewer("", assets, classIdString)), body.Home)
}

func HandleGetAsset(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request, classId int, assetName string) {
	fmt.Println("📥 [HandleGetAsset] Request received")

	classIdString := strconv.Itoa(classId)

	class, err := database.Get[models.Class](store, database.Buckets["classes"], classIdString)
	if err != nil {
		fmt.Printf("Error getting class: %v \n", err)
	}

	asset, err := database.GetWithPrefix[models.Asset](store, database.Buckets["assets"], assetName, class.Subject, class.Grade)
	if err != nil {
		fmt.Printf("Error getting asset: %v \n", err)
	}

	temporaryLink, err := storage.GetTemporaryLink(r.Context(), asset.OriginalName, 3*time.Minute)
	if err != nil {
		fmt.Printf("Error generating temporary link: %v \n", err)
	}

	pdfViewerFrame.PdfViewerFrame(temporaryLink).Render(r.Context(), w)
}
