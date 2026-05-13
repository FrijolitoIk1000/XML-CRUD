package handler

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FrijolitoIk1000/XML-CRUD/model"
	"github.com/FrijolitoIk1000/XML-CRUD/report"
	"github.com/FrijolitoIk1000/XML-CRUD/store"
)

type Handler struct {
	store     *store.Store
	indexTmpl *template.Template
	editTmpl  *template.Template
}

type pageData struct {
	Items      []model.Item
	TotalValue float64
	TotalItems int
}

func New(s *store.Store, fsys fs.FS) *Handler {
	fns := template.FuncMap{
		"itemValue": func(qty int, price float64) string {
			return fmt.Sprintf("%.2f", float64(qty)*price)
		},
	}
	indexTmpl := template.Must(
		template.New("index.html").Funcs(fns).ParseFS(fsys, "index.html"),
	)
	editTmpl := template.Must(
		template.New("edit.html").ParseFS(fsys, "edit.html"),
	)
	return &Handler{store: s, indexTmpl: indexTmpl, editTmpl: editTmpl}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	items, err := h.store.Load()
	if err != nil {
		h.serverError(w, err)
		return
	}
	data := pageData{Items: items, TotalItems: len(items)}
	for _, it := range items {
		data.TotalValue += float64(it.Quantity) * it.Price
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.indexTmpl.Execute(w, data); err != nil {
		log.Println("template error:", err)
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	qty, _ := strconv.Atoi(r.FormValue("quantity"))
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	item := model.Item{
		ID:          model.NewID(),
		Name:        strings.TrimSpace(r.FormValue("name")),
		Category:    strings.TrimSpace(r.FormValue("category")),
		Subcategory: strings.TrimSpace(r.FormValue("subcategory")),
		Quantity:    qty,
		Price:       price,
		Unit:        strings.TrimSpace(r.FormValue("unit")),
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}
	if item.Name == "" || item.Category == "" {
		http.Error(w, "nombre y categoría son requeridos", http.StatusBadRequest)
		return
	}
	items, err := h.store.Load()
	if err != nil {
		h.serverError(w, err)
		return
	}
	if err := h.store.Save(append(items, item)); err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/items/edit/")
	if id == "" {
		http.Error(w, "ID requerido", http.StatusBadRequest)
		return
	}
	items, err := h.store.Load()
	if err != nil {
		h.serverError(w, err)
		return
	}
	var found *model.Item
	for i := range items {
		if items[i].ID == id {
			found = &items[i]
			break
		}
	}
	if found == nil {
		http.Error(w, "artículo no encontrado", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.editTmpl.Execute(w, found); err != nil {
		log.Println("template error:", err)
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/items/update/")
	if id == "" {
		http.Error(w, "ID requerido", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	qty, _ := strconv.Atoi(r.FormValue("quantity"))
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)

	items, err := h.store.Load()
	if err != nil {
		h.serverError(w, err)
		return
	}
	found := false
	for i := range items {
		if items[i].ID == id {
			items[i].Name = strings.TrimSpace(r.FormValue("name"))
			items[i].Category = strings.TrimSpace(r.FormValue("category"))
			items[i].Subcategory = strings.TrimSpace(r.FormValue("subcategory"))
			items[i].Quantity = qty
			items[i].Price = price
			items[i].Unit = strings.TrimSpace(r.FormValue("unit"))
			items[i].UpdatedAt = time.Now().Format(time.RFC3339)
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "artículo no encontrado", http.StatusNotFound)
		return
	}
	if err := h.store.Save(items); err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/items/delete/")
	if id == "" {
		http.Error(w, "ID requerido", http.StatusBadRequest)
		return
	}
	items, err := h.store.Load()
	if err != nil {
		h.serverError(w, err)
		return
	}
	out := make([]model.Item, 0, len(items))
	for _, it := range items {
		if it.ID != id {
			out = append(out, it)
		}
	}
	if err := h.store.Save(out); err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.Load()
	if err != nil {
		h.serverError(w, err)
		return
	}
	rpt := report.Build(items)
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="informe_inventario.xml"`)
	fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(rpt); err != nil {
		log.Println("XML error:", err)
	}
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	log.Println("error:", err)
	http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
}
