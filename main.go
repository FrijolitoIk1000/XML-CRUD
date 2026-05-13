package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/FrijolitoIk1000/XML-CRUD/handler"
	"github.com/FrijolitoIk1000/XML-CRUD/store"
)

//go:embed templates
var embeddedTemplates embed.FS

func main() {
	tmplFS, err := fs.Sub(embeddedTemplates, "templates")
	if err != nil {
		log.Fatal(err)
	}

	s := store.New("inventory.json")
	h := handler.New(s, tmplFS)

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.Index)
	mux.HandleFunc("/items/create", h.Create)
	mux.HandleFunc("/items/edit/", h.Edit)
	mux.HandleFunc("/items/update/", h.Update)
	mux.HandleFunc("/items/delete/", h.Delete)
	mux.HandleFunc("/report", h.Report)

	const addr = ":8080"
	log.Printf("Servidor de inventario iniciado → http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
