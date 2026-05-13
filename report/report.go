package report

import (
	"encoding/xml"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/FrijolitoIk1000/XML-CRUD/model"
)

type Report struct {
	XMLName     xml.Name   `xml:"Inventario"`
	TotalItems  int        `xml:"totalArticulos,attr"`
	TotalValue  float64    `xml:"valorTotal,attr"`
	GeneratedAt string     `xml:"generadoEn,attr"`
	Categories  []Category `xml:"Categoria"`
}

type Category struct {
	Name          string        `xml:"nombre,attr"`
	Articles      int           `xml:"articulos,attr"`
	Units         int           `xml:"unidades,attr"`
	Value         float64       `xml:"valor,attr"`
	Percentage    string        `xml:"porcentaje,attr"`
	Subcategories []Subcategory `xml:"Subcategoria"`
}

type Subcategory struct {
	Name       string  `xml:"nombre,attr"`
	Articles   int     `xml:"articulos,attr"`
	Units      int     `xml:"unidades,attr"`
	Value      float64 `xml:"valor,attr"`
	Percentage string  `xml:"porcentaje,attr"`
	Items      []Entry `xml:"Articulo"`
}

type Entry struct {
	ID       string  `xml:"id,attr"`
	Name     string  `xml:"nombre,attr"`
	Quantity int     `xml:"cantidad,attr"`
	Unit     string  `xml:"unidad,attr"`
	Price    float64 `xml:"precio,attr"`
	Value    float64 `xml:"valor,attr"`
}

func Build(items []model.Item) Report {
	totalVal := 0.0
	for _, it := range items {
		totalVal += float64(it.Quantity) * it.Price
	}

	tree := map[string]map[string][]model.Item{}
	for _, it := range items {
		if tree[it.Category] == nil {
			tree[it.Category] = map[string][]model.Item{}
		}
		sub := it.Subcategory
		if sub == "" {
			sub = "(sin subcategoría)"
		}
		tree[it.Category][sub] = append(tree[it.Category][sub], it)
	}

	catNames := sortedKeys(tree)
	cats := make([]Category, 0, len(catNames))

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
		subs := make([]Subcategory, 0, len(subNames))

		for _, subName := range subNames {
			subItems := subMap[subName]
			subVal := 0.0
			subUnits := 0
			entries := make([]Entry, 0, len(subItems))
			for _, it := range subItems {
				v := round2(float64(it.Quantity) * it.Price)
				subVal += v
				subUnits += it.Quantity
				entries = append(entries, Entry{
					ID:       it.ID,
					Name:     it.Name,
					Quantity: it.Quantity,
					Unit:     it.Unit,
					Price:    round2(it.Price),
					Value:    v,
				})
			}
			catVal += subVal
			catArticles += len(subItems)
			catUnits += subUnits
			subs = append(subs, Subcategory{
				Name:       subName,
				Articles:   len(subItems),
				Units:      subUnits,
				Value:      round2(subVal),
				Percentage: pctStr(subVal, totalVal),
				Items:      entries,
			})
		}

		cats = append(cats, Category{
			Name:          catName,
			Articles:      catArticles,
			Units:         catUnits,
			Value:         round2(catVal),
			Percentage:    pctStr(catVal, totalVal),
			Subcategories: subs,
		})
	}

	return Report{
		TotalItems:  len(items),
		TotalValue:  round2(totalVal),
		GeneratedAt: time.Now().Format(time.RFC3339),
		Categories:  cats,
	}
}

func sortedKeys(m map[string]map[string][]model.Item) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func pctStr(part, total float64) string {
	if total == 0 {
		return "0.00%"
	}
	return fmt.Sprintf("%.2f%%", part/total*100)
}
