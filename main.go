package main

import (
	"encoding/json"
	"math"
	"strconv"
	"syscall/js"

	"github.com/akhenakh/oureadb/index/geodata"
	"github.com/akhenakh/oureadb/s2tools"
	"github.com/golang/geo/s2"
	"github.com/twpayne/go-geom/encoding/geojson"
)

const earthCircumferenceMeter = 40075017

var document js.Value

func init() {
	document = js.Global().Get("document")
}

func getCoverParams() (minLevel, maxLevel, maxCells int) {
	minS := document.Call("getElementById", "minRange").Get("value").String()
	minLevel, err := strconv.Atoi(minS)
	if err != nil {
		println(err.Error())
		return
	}

	maxS := document.Call("getElementById", "maxRange").Get("value").String()
	maxLevel, err = strconv.Atoi(maxS)
	if err != nil {
		println(err.Error())
		return
	}

	maxCS := document.Call("getElementById", "maxCellsRange").Get("value").String()
	maxCells, err = strconv.Atoi(maxCS)
	if err != nil {
		println(err.Error())
		return
	}

	return minLevel, maxLevel, maxCells
}

func geoFeaturesJSONToCells(i []js.Value) {
	var fc geojson.FeatureCollection
	b := js.ValueOf(i[0]).String()
	err := json.Unmarshal([]byte(b), &fc)
	if err != nil {
		println(err.Error())
		return
	}
	var res s2.CellUnion
	for _, f := range fc.Features {
		cu := computeFeatureCells(f)
		res = append(res, cu...)
	}

	jsonb := s2tools.CellUnionToGeoJSON(res)
	updateUIWithData(string(jsonb))
}

func geoCircleToCells(i []js.Value) {
	lng := js.ValueOf(i[0]).Float()
	lat := js.ValueOf(i[1]).Float()
	radius := js.ValueOf(i[2]).Float()

	center := s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lng))
	cap := s2.CapFromCenterArea(center, s2RadialAreaMeters(radius))

	minLevel, maxLevel, maxCells := getCoverParams()
	coverer := &s2.RegionCoverer{MinLevel: minLevel, MaxLevel: maxLevel, MaxCells: maxCells}
	cu := coverer.Covering(cap)
	jsonb := s2tools.CellUnionToGeoJSON(cu)
	updateUIWithData(string(jsonb))
}

func geoJSONToCells(i []js.Value) {
	var f geojson.Feature
	b := js.ValueOf(i[0]).String()
	err := json.Unmarshal([]byte(b), &f)
	if err != nil {
		println(err.Error())
		return
	}
	cu := computeFeatureCells(&f)
	jsonb := s2tools.CellUnionToGeoJSON(cu)
	updateUIWithData(string(jsonb))
}

func computeFeatureCells(f *geojson.Feature) s2.CellUnion {
	gd := &geodata.GeoData{}
	err := geodata.GeoJSONFeatureToGeoData(f, gd)
	if err != nil {
		println(err)
		return nil
	}

	minLevel, maxLevel, maxCells := getCoverParams()
	coverer := &s2.RegionCoverer{MinLevel: minLevel, MaxLevel: maxLevel, MaxCells: maxCells}

	cu, err := geodata.GeoDataToCellUnion(gd, coverer)
	if err != nil {
		println(err)
		return nil
	}
	return cu
}

func drawCells(i []js.Value) {
	un := make(map[s2.CellID]struct{})
	for _, cs := range i {
		cs := js.ValueOf(cs).String()
		if cs != "" {
			c := s2.CellIDFromToken(cs)
			un[c] = struct{}{}
		}
	}

	cells := make(s2.CellUnion, len(un))
	count := 0
	for c, _ := range un {
		cells[count] = c
		count++
	}
	b := s2tools.CellUnionToGeoJSON(cells)
	updateUIWithData(string(b))
}

func updateUIWithData(data string) {
	js.Global().Set("data", data)
	js.Global().Call("updateui")
}

func registerCallbacks() {
	js.Global().Set("drawcells", js.NewCallback(drawCells))
	js.Global().Set("circlecell", js.NewCallback(geoCircleToCells))
	js.Global().Set("geocell", js.NewCallback(geoJSONToCells))
	js.Global().Set("geofeaturescell", js.NewCallback(geoFeaturesJSONToCells))
}

func s2RadialAreaMeters(radius float64) float64 {
	r := (radius / earthCircumferenceMeter) * math.Pi * 2
	return math.Pi * r * r
}

func main() {
	c := make(chan struct{}, 0)
	println("Wasm ready")
	registerCallbacks()
	<-c
}
