package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── Modelos ───────────────────────────────────────────────────────────────────

type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Subcategory string  `json:"subcategory"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	Unit        string  `json:"unit"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type dbStore struct {
	Items []Item `json:"items"`
}

// ── Estructuras XML ───────────────────────────────────────────────────────────

type XMLReport struct {
	XMLName     xml.Name      `xml:"Inventario"`
	TotalItems  int           `xml:"totalArticulos,attr"`
	TotalValue  float64       `xml:"valorTotal,attr"`
	GeneratedAt string        `xml:"generadoEn,attr"`
	Categories  []XMLCategory `xml:"Categoria"`
}

type XMLCategory struct {
	Name          string           `xml:"nombre,attr"`
	Articles      int              `xml:"articulos,attr"`
	Units         int              `xml:"unidades,attr"`
	Value         float64          `xml:"valor,attr"`
	Percentage    string           `xml:"porcentaje,attr"`
	Subcategories []XMLSubcategory `xml:"Subcategoria"`
}

type XMLSubcategory struct {
	Name       string    `xml:"nombre,attr"`
	Articles   int       `xml:"articulos,attr"`
	Units      int       `xml:"unidades,attr"`
	Value      float64   `xml:"valor,attr"`
	Percentage string    `xml:"porcentaje,attr"`
	Items      []XMLItem `xml:"Articulo"`
}

type XMLItem struct {
	ID       string  `xml:"id,attr"`
	Name     string  `xml:"nombre,attr"`
	Quantity int     `xml:"cantidad,attr"`
	Unit     string  `xml:"unidad,attr"`
	Price    float64 `xml:"precio,attr"`
	Value    float64 `xml:"valor,attr"`
}

// ── Persistencia JSON ─────────────────────────────────────────────────────────

const dataFile = "inventory.json"

var mu sync.RWMutex

func loadItems() ([]Item, error) {
	mu.RLock()
	defer mu.RUnlock()
	f, err := os.Open(dataFile)
	if os.IsNotExist(err) {
		return []Item{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var s dbStore
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return []Item{}, nil
	}
	return s.Items, nil
}

func saveItems(items []Item) error {
	mu.Lock()
	defer mu.Unlock()
	f, err := os.Create(dataFile)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(dbStore{Items: items})
}

func genID() string { return strconv.FormatInt(time.Now().UnixNano(), 36) }

// ── Helpers ───────────────────────────────────────────────────────────────────

func r2(v float64) float64 { return math.Round(v*100) / 100 }

func pct(part, total float64) float64 {
	if total == 0 {
		return 0
	}
	return part / total * 100
}

// ── Datos para plantillas ─────────────────────────────────────────────────────

type pageData struct {
	Items      []Item
	TotalValue float64
	TotalItems int
}

// ── Plantillas globales ───────────────────────────────────────────────────────

var indexTmpl = template.Must(
	template.New("index").Funcs(template.FuncMap{
		"itemValue": func(qty int, price float64) string {
			return fmt.Sprintf("%.2f", float64(qty)*price)
		},
	}).Parse(indexHTML),
)

var editTmpl = template.Must(template.New("edit").Parse(editHTML))

// ── Handlers ──────────────────────────────────────────────────────────────────

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	items, err := loadItems()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	data := pageData{Items: items, TotalItems: len(items)}
	for _, it := range items {
		data.TotalValue += float64(it.Quantity) * it.Price
	}
	data.TotalValue = r2(data.TotalValue)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := indexTmpl.Execute(w, data); err != nil {
		log.Println("template error:", err)
	}
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	qty, _ := strconv.Atoi(r.FormValue("quantity"))
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	item := Item{
		ID:          genID(),
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
		http.Error(w, "nombre y categoría son requeridos", 400)
		return
	}
	items, _ := loadItems()
	items = append(items, item)
	if err := saveItems(items); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/items/edit/")
	if id == "" {
		http.Error(w, "ID requerido", 400)
		return
	}
	items, _ := loadItems()
	var found *Item
	for i := range items {
		if items[i].ID == id {
			found = &items[i]
			break
		}
	}
	if found == nil {
		http.Error(w, "artículo no encontrado", 404)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	editTmpl.Execute(w, found)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/items/update/")
	if id == "" {
		http.Error(w, "ID requerido", 400)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	qty, _ := strconv.Atoi(r.FormValue("quantity"))
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	items, _ := loadItems()
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
		http.Error(w, "artículo no encontrado", 404)
		return
	}
	if err := saveItems(items); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/items/delete/")
	if id == "" {
		http.Error(w, "ID requerido", 400)
		return
	}
	items, _ := loadItems()
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if it.ID != id {
			out = append(out, it)
		}
	}
	if err := saveItems(out); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	items, _ := loadItems()

	totalVal := 0.0
	for _, it := range items {
		totalVal += float64(it.Quantity) * it.Price
	}

	// árbol: categoría → subcategoría → []Item
	tree := map[string]map[string][]Item{}
	for _, it := range items {
		if tree[it.Category] == nil {
			tree[it.Category] = map[string][]Item{}
		}
		sub := it.Subcategory
		if sub == "" {
			sub = "(sin subcategoría)"
		}
		tree[it.Category][sub] = append(tree[it.Category][sub], it)
	}

	catNames := make([]string, 0, len(tree))
	for c := range tree {
		catNames = append(catNames, c)
	}
	sort.Strings(catNames)

	xmlCats := make([]XMLCategory, 0, len(catNames))
	for _, catName := range catNames {
		subMap := tree[catName]
		subNames := make([]string, 0, len(subMap))
		for s := range subMap {
			subNames = append(subNames, s)
		}
		sort.Strings(subNames)

		catVal := 0.0
		catArticles := 0
		catUnits := 0
		xmlSubs := make([]XMLSubcategory, 0, len(subNames))

		for _, subName := range subNames {
			subItems := subMap[subName]
			subVal := 0.0
			subUnits := 0
			xmlIts := make([]XMLItem, 0, len(subItems))
			for _, it := range subItems {
				v := float64(it.Quantity) * it.Price
				subVal += v
				subUnits += it.Quantity
				xmlIts = append(xmlIts, XMLItem{
					ID:       it.ID,
					Name:     it.Name,
					Quantity: it.Quantity,
					Unit:     it.Unit,
					Price:    r2(it.Price),
					Value:    r2(v),
				})
			}
			catVal += subVal
			catArticles += len(subItems)
			catUnits += subUnits
			xmlSubs = append(xmlSubs, XMLSubcategory{
				Name:       subName,
				Articles:   len(subItems),
				Units:      subUnits,
				Value:      r2(subVal),
				Percentage: fmt.Sprintf("%.2f%%", pct(subVal, totalVal)),
				Items:      xmlIts,
			})
		}

		xmlCats = append(xmlCats, XMLCategory{
			Name:          catName,
			Articles:      catArticles,
			Units:         catUnits,
			Value:         r2(catVal),
			Percentage:    fmt.Sprintf("%.2f%%", pct(catVal, totalVal)),
			Subcategories: xmlSubs,
		})
	}

	report := XMLReport{
		TotalItems:  len(items),
		TotalValue:  r2(totalVal),
		GeneratedAt: time.Now().Format(time.RFC3339),
		Categories:  xmlCats,
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="informe_inventario.xml"`)
	fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(report); err != nil {
		log.Println("XML error:", err)
	}
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/items/create", createHandler)
	mux.HandleFunc("/items/edit/", editHandler)
	mux.HandleFunc("/items/update/", updateHandler)
	mux.HandleFunc("/items/delete/", deleteHandler)
	mux.HandleFunc("/report", reportHandler)

	addr := ":8080"
	log.Printf("Servidor de inventario iniciado → http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// ── Plantillas HTML ───────────────────────────────────────────────────────────

const indexHTML = `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Gestión de Inventario</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,-apple-system,sans-serif;background:#f1f5f9;color:#1e293b;min-height:100vh}
header{background:linear-gradient(135deg,#1e40af,#3b82f6);color:#fff;padding:1rem 2rem;display:flex;justify-content:space-between;align-items:center;box-shadow:0 2px 8px rgba(0,0,0,.25)}
header h1{font-size:1.3rem;font-weight:700;letter-spacing:.3px}
.btn{display:inline-flex;align-items:center;gap:.3rem;padding:.45rem 1rem;border-radius:6px;border:none;cursor:pointer;font-size:.85rem;font-weight:600;text-decoration:none;transition:filter .15s}
.btn:hover{filter:brightness(.88)}
.btn-blue{background:#3b82f6;color:#fff}
.btn-green{background:#10b981;color:#fff}
.btn-amber{background:#f59e0b;color:#fff}
.btn-red{background:#ef4444;color:#fff}
.btn-sm{padding:.25rem .6rem;font-size:.78rem}
.container{max-width:1280px;margin:0 auto;padding:1.5rem}
.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:1rem;margin-bottom:1.5rem}
.stat{background:#fff;border-radius:10px;padding:1rem 1.25rem;box-shadow:0 1px 4px rgba(0,0,0,.08);border-left:4px solid #3b82f6}
.stat .v{font-size:1.6rem;font-weight:800;color:#1e40af}
.stat .l{font-size:.73rem;color:#64748b;margin-top:.15rem;text-transform:uppercase;letter-spacing:.4px}
.card{background:#fff;border-radius:10px;box-shadow:0 1px 4px rgba(0,0,0,.08);padding:1.25rem;margin-bottom:1.5rem}
.card h2{font-size:1rem;font-weight:700;color:#1e40af;margin-bottom:1rem;padding-bottom:.5rem;border-bottom:1px solid #e2e8f0}
.form-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(165px,1fr));gap:.75rem}
.fg{display:flex;flex-direction:column;gap:.3rem}
label{font-size:.77rem;font-weight:600;color:#475569;text-transform:uppercase;letter-spacing:.3px}
input{padding:.42rem .65rem;border:1.5px solid #cbd5e1;border-radius:6px;font-size:.875rem;transition:border-color .15s,box-shadow .15s}
input:focus{outline:none;border-color:#3b82f6;box-shadow:0 0 0 3px rgba(59,130,246,.18)}
.form-actions{margin-top:1rem}
.table-wrap{overflow-x:auto}
table{width:100%;border-collapse:collapse;font-size:.85rem}
thead tr{background:#eff6ff}
th{padding:.65rem .8rem;text-align:left;font-weight:700;color:#1e40af;border-bottom:2px solid #bfdbfe;white-space:nowrap;font-size:.78rem;text-transform:uppercase;letter-spacing:.4px}
td{padding:.55rem .8rem;border-bottom:1px solid #f1f5f9;vertical-align:middle}
tr:last-child td{border-bottom:none}
tr:hover td{background:#f8fafc}
.badge{display:inline-block;padding:.15rem .55rem;border-radius:999px;font-size:.72rem;font-weight:700;background:#dbeafe;color:#1d4ed8}
.badge-sub{background:#f0fdf4;color:#15803d}
.actions{display:flex;gap:.35rem;align-items:center;flex-wrap:nowrap}
.empty{text-align:center;padding:3rem 1rem;color:#94a3b8}
.empty-icon{font-size:3rem;display:block;margin-bottom:.75rem}
tfoot td{font-weight:700;background:#eff6ff;color:#1e40af;border-top:2px solid #bfdbfe}
</style>
</head>
<body>
<header>
  <h1>Gestión de Inventario</h1>
  <a href="/report" class="btn btn-green">&#8659; Descargar Informe XML</a>
</header>
<div class="container">

  <div class="stats">
    <div class="stat">
      <div class="v">{{.TotalItems}}</div>
      <div class="l">Artículos registrados</div>
    </div>
    <div class="stat">
      <div class="v">$ {{printf "%.2f" .TotalValue}}</div>
      <div class="l">Valor total del inventario</div>
    </div>
  </div>

  <div class="card">
    <h2>Agregar artículo</h2>
    <form method="POST" action="/items/create">
      <div class="form-grid">
        <div class="fg"><label>Nombre *</label><input name="name" required placeholder="Ej: Laptop Dell"></div>
        <div class="fg"><label>Categoría *</label><input name="category" required placeholder="Ej: Electrónica"></div>
        <div class="fg"><label>Subcategoría</label><input name="subcategory" placeholder="Ej: Computadoras"></div>
        <div class="fg"><label>Cantidad</label><input name="quantity" type="number" min="0" value="1"></div>
        <div class="fg"><label>Precio unitario ($)</label><input name="price" type="number" step="0.01" min="0" value="0.00"></div>
        <div class="fg"><label>Unidad</label><input name="unit" placeholder="Ej: piezas, kg, lt"></div>
      </div>
      <div class="form-actions">
        <button type="submit" class="btn btn-blue">&#43; Agregar artículo</button>
      </div>
    </form>
  </div>

  <div class="card">
    <h2>Inventario ({{.TotalItems}} artículo{{if ne .TotalItems 1}}s{{end}})</h2>
    {{if .Items}}
    <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>Nombre</th>
          <th>Categoría</th>
          <th>Subcategoría</th>
          <th>Cantidad</th>
          <th>Unidad</th>
          <th>Precio unit.</th>
          <th>Valor total</th>
          <th>Acciones</th>
        </tr>
      </thead>
      <tbody>
      {{range .Items}}
      <tr>
        <td><strong>{{.Name}}</strong></td>
        <td><span class="badge">{{.Category}}</span></td>
        <td>{{if .Subcategory}}<span class="badge badge-sub">{{.Subcategory}}</span>{{else}}&mdash;{{end}}</td>
        <td>{{.Quantity}}</td>
        <td>{{if .Unit}}{{.Unit}}{{else}}&mdash;{{end}}</td>
        <td>$ {{printf "%.2f" .Price}}</td>
        <td>$ {{itemValue .Quantity .Price}}</td>
        <td>
          <div class="actions">
            <a href="/items/edit/{{.ID}}" class="btn btn-amber btn-sm">Editar</a>
            <form method="POST" action="/items/delete/{{.ID}}" onsubmit="return confirm('¿Eliminar &quot;{{.Name}}&quot;?')">
              <button type="submit" class="btn btn-red btn-sm">Eliminar</button>
            </form>
          </div>
        </td>
      </tr>
      {{end}}
      </tbody>
      <tfoot>
        <tr>
          <td colspan="6" style="text-align:right">Total del inventario:</td>
          <td>$ {{printf "%.2f" .TotalValue}}</td>
          <td></td>
        </tr>
      </tfoot>
    </table>
    </div>
    {{else}}
    <div class="empty">
      <span class="empty-icon">&#128230;</span>
      No hay artículos registrados.<br>¡Agrega el primero usando el formulario de arriba!
    </div>
    {{end}}
  </div>

</div>
</body>
</html>`

const editHTML = `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Editar Artículo &mdash; Inventario</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,-apple-system,sans-serif;background:#f1f5f9;color:#1e293b}
header{background:linear-gradient(135deg,#1e40af,#3b82f6);color:#fff;padding:1rem 2rem;display:flex;justify-content:space-between;align-items:center;box-shadow:0 2px 8px rgba(0,0,0,.25)}
header h1{font-size:1.3rem;font-weight:700}
.btn{display:inline-flex;align-items:center;gap:.3rem;padding:.45rem 1rem;border-radius:6px;border:none;cursor:pointer;font-size:.85rem;font-weight:600;text-decoration:none;transition:filter .15s}
.btn:hover{filter:brightness(.88)}
.btn-blue{background:#3b82f6;color:#fff}
.btn-gray{background:#64748b;color:#fff}
.container{max-width:740px;margin:2rem auto;padding:0 1.5rem}
.card{background:#fff;border-radius:10px;box-shadow:0 1px 4px rgba(0,0,0,.08);padding:1.5rem}
.card h2{font-size:1rem;font-weight:700;color:#1e40af;margin-bottom:1.25rem;padding-bottom:.5rem;border-bottom:1px solid #e2e8f0}
.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:.85rem}
.span2{grid-column:1/-1}
.fg{display:flex;flex-direction:column;gap:.3rem}
label{font-size:.77rem;font-weight:600;color:#475569;text-transform:uppercase;letter-spacing:.3px}
input{padding:.45rem .7rem;border:1.5px solid #cbd5e1;border-radius:6px;font-size:.9rem;transition:border-color .15s,box-shadow .15s}
input:focus{outline:none;border-color:#3b82f6;box-shadow:0 0 0 3px rgba(59,130,246,.18)}
.form-actions{margin-top:1.25rem;display:flex;gap:.5rem}
.meta{margin-top:1.25rem;padding-top:1rem;border-top:1px solid #e2e8f0;font-size:.73rem;color:#94a3b8;line-height:1.6}
</style>
</head>
<body>
<header>
  <h1>Editar artículo</h1>
  <a href="/" class="btn btn-gray">&#8592; Volver al inventario</a>
</header>
<div class="container">
  <div class="card">
    <h2>Modificar datos del artículo</h2>
    <form method="POST" action="/items/update/{{.ID}}">
      <div class="form-grid">
        <div class="fg span2">
          <label>Nombre *</label>
          <input name="name" value="{{.Name}}" required>
        </div>
        <div class="fg">
          <label>Categoría *</label>
          <input name="category" value="{{.Category}}" required>
        </div>
        <div class="fg">
          <label>Subcategoría</label>
          <input name="subcategory" value="{{.Subcategory}}">
        </div>
        <div class="fg">
          <label>Cantidad</label>
          <input name="quantity" type="number" min="0" value="{{.Quantity}}">
        </div>
        <div class="fg">
          <label>Precio unitario ($)</label>
          <input name="price" type="number" step="0.01" min="0" value="{{printf "%.2f" .Price}}">
        </div>
        <div class="fg">
          <label>Unidad</label>
          <input name="unit" value="{{.Unit}}">
        </div>
      </div>
      <div class="form-actions">
        <button type="submit" class="btn btn-blue">Guardar cambios</button>
        <a href="/" class="btn btn-gray">Cancelar</a>
      </div>
    </form>
    <div class="meta">
      ID: {{.ID}}<br>
      Creado: {{.CreatedAt}}<br>
      Última modificación: {{.UpdatedAt}}
    </div>
  </div>
</div>
</body>
</html>`
