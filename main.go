package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/FrijolitoIk1000/XML-CRUD/handler"
	"github.com/FrijolitoIk1000/XML-CRUD/store"
)

//go:embed templates
var embeddedTemplates embed.FS

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL no configurada")
	}

	s, err := store.New(dsn)
	if err != nil {
		log.Fatal("no se pudo conectar a la base de datos:", err)
	}

	tmplFS, err := fs.Sub(embeddedTemplates, "templates")
	if err != nil {
		log.Fatal(err)
	}

	h := handler.New(s, tmplFS)

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.Index)
	mux.HandleFunc("/items/create", h.Create)
	mux.HandleFunc("/items/edit/", h.Edit)
	mux.HandleFunc("/items/update/", h.Update)
	mux.HandleFunc("/items/delete/", h.Delete)
	mux.HandleFunc("/report", h.Report)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("Servidor de inventario iniciado → http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
