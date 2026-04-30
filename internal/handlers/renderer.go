package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/ferdifir/jetlink/internal/models"
)


type TemplateData struct {
	Tenant   *models.Tenant
	Customer *models.Customer
	Flash    string
	Error    string
	Data     interface{}
}

var templateFuncs = template.FuncMap{
	"statusLabel": func(s string) string {
		switch s {
		case "waiting":
			return "Menunggu"
		case "called":
			return "Dipanggil"
		case "serving":
			return "Dilayani"
		case "done":
			return "Selesai"
		case "skipped":
			return "Dilewati"
		}
		return s
	},
	"statusColor": func(s string) string {
		switch s {
		case "waiting":
			return "bg-yellow-100 text-yellow-800"
		case "called":
			return "bg-blue-100 text-blue-800"
		case "serving":
			return "bg-indigo-100 text-indigo-800"
		case "done":
			return "bg-green-100 text-green-800"
		case "skipped":
			return "bg-gray-100 text-gray-500"
		}
		return ""
	},
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
	"mul": func(a, b int) int { return a * b },
}

func renderCustomer(w http.ResponseWriter, page string, data TemplateData) {
	renderWith(w, "customer.html", "customer/"+page, data)
}

func renderAdmin(w http.ResponseWriter, page string, data TemplateData) {
	renderWith(w, "admin.html", "admin/"+page, data)
}

func renderSuperadmin(w http.ResponseWriter, page string, data TemplateData) {
	renderWith(w, "admin.html", "superadmin/"+page, data)
}

func renderWith(w http.ResponseWriter, layout, page string, data TemplateData) {
	layoutPath := filepath.Join("web", "templates", "layouts", layout)
	pagePath := filepath.Join("web", "templates", page+".html")

	tmpl, err := template.New("base").Funcs(templateFuncs).ParseFiles(layoutPath, pagePath)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Render error: "+err.Error(), http.StatusInternalServerError)
	}
}

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func redirectf(w http.ResponseWriter, r *http.Request, url string) {
	http.Redirect(w, r, url, http.StatusFound)
}
