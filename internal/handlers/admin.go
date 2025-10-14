package handlers

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/panelsContent"
	"frontend/templates/components/pdfViewer/pdfViewer"
	"net/http"
	"strconv"
)

type SubOptionSlot struct {
	Title string
	Url   string
}

func HandleAdminDefault(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminDefault] Request received")

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
