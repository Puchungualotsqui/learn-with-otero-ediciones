package admin

import (
	"fmt"
	"frontend/dto"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/admin/adminHome"
	"net/http"
)

var options []*dto.AdminOption = []*dto.AdminOption{
	&dto.AdminOption{
		Title:       "Gestión de Usuarios",
		Description: "Ver y modificar información de los usuarios",
		SubUrl:      "user",
		SubOptions: []*dto.AdminSubOptionSlot{
			&dto.AdminSubOptionSlot{
				Title: "Crear",
				Url:   "create",
			},
			&dto.AdminSubOptionSlot{
				Title: "Modificar",
				Url:   "modify",
			},
			&dto.AdminSubOptionSlot{
				Title: "Buscar",
				Url:   "search",
			},
		},
	},
	&dto.AdminOption{
		Title:       "Clases",
		Description: "Crear, buscar y administrar clases",
		SubUrl:      "class",
		SubOptions: []*dto.AdminSubOptionSlot{
			&dto.AdminSubOptionSlot{
				Title: "Crear",
				Url:   "create",
			},
			&dto.AdminSubOptionSlot{
				Title: "Modificar",
				Url:   "modify",
			},
			&dto.AdminSubOptionSlot{
				Title: "Buscar",
				Url:   "search",
			},
		},
	},
	&dto.AdminOption{
		Title:       "Gestión de Materias",
		Description: "Crear, editar y administrar información de materias",
		SubUrl:      "subject",
		SubOptions: []*dto.AdminSubOptionSlot{
			&dto.AdminSubOptionSlot{
				Title: "Administrar",
				Url:   "manage",
			},
			&dto.AdminSubOptionSlot{
				Title: "Renombrar",
				Url:   "rename",
			},
		},
	},
	&dto.AdminOption{
		Title:       "Recursos",
		Description: "Agregar y eliminar materiales y archivos",
		SubUrl:      "class",
		SubOptions: []*dto.AdminSubOptionSlot{
			&dto.AdminSubOptionSlot{
				Title: "Administrar",
				Url:   "manage",
			},
		},
	},
}

func HandleAdminDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminDefault] Request received")

	render.RenderWithLayout(w, r, adminHome.AdminHome(options), body.Home)
	fmt.Println("  ✔ Render complete")
}
