package render

import (
	"frontend/templates/views"
	"net/http"

	"github.com/a-h/templ"
)

func RenderWithLayout(
	w http.ResponseWriter,
	r *http.Request,
	content templ.Component,
	wrappers ...func(templ.Component) templ.Component,
) {
	if r.Header.Get("HX-Request") == "true" {
		content.Render(r.Context(), w)
		return
	}

	// Apply wrappers in order
	wrapped := content
	for _, wrap := range wrappers {
		wrapped = wrap(wrapped)
	}

	views.Layout(wrapped).Render(r.Context(), w)
}
