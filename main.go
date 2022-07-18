package main

import (
	"encoding/json"
	"fmt"
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

func getCoverParams() (minLevel, maxLevel, maxCells int, inside bool) {
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

	coverCS := document.Call("getElementById", "icover").Get("checked").Bool()
	if coverCS {
		inside = true
	}

	return minLevel, maxLevel, maxCells, inside
}

func geoFeaturesJSONToCells(this js.Value, i []js.Value) interface{} {
	var fc geojson.FeatureCollection
	b := js.ValueOf(i[0]).String()

	err := json.Unmarshal([]byte(b), &fc)
	if err != nil {
		println(err.Error())
		return nil
	}
	var res s2.CellUnion
	for _, f := range fc.Features {
		cu, err := computeFeatureCells(f)
		if err != nil {
			println("error computing cells", err)
			return nil
		}
		res = append(res, cu...)
	}

	jsonb := s2tools.CellUnionToGeoJSON(res)
	if len(jsonb) == 0 {
		println("empty result")
		return nil
	}
	updateUIWithData(string(jsonb))
	return nil
}

func geoCircleToCells(this js.Value, i []js.Value) interface{} {
	lng := js.ValueOf(i[0]).Float()
	lat := js.ValueOf(i[1]).Float()
	radius := js.ValueOf(i[2]).Float()

	center := s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lng))
	cap := s2.CapFromCenterArea(center, s2RadialAreaMeters(radius))

	minLevel, maxLevel, maxCells, inside := getCoverParams()
	coverer := &s2.RegionCoverer{MinLevel: minLevel, MaxLevel: maxLevel, MaxCells: maxCells}
	var cu s2.CellUnion
	if inside {
		cu = coverer.InteriorCovering(cap)
	} else {
		cu = coverer.Covering(cap)
	}
	jsonb := s2tools.CellUnionToGeoJSON(cu)
	updateUIWithData(string(jsonb))
	return nil
}

func geoJSONToCells(this js.Value, i []js.Value) interface{} {
	var f geojson.Feature
	b := js.ValueOf(i[0]).String()

	err := json.Unmarshal([]byte(b), &f)
	if err != nil {
		println(err.Error())
		return nil
	}
	cu, err := computeFeatureCells(&f)
	if err != nil {
		println("error computing cells", err)
		return nil
	}
	jsonb := s2tools.CellUnionToGeoJSON(cu)
	if len(jsonb) == 0 {
		println("can't generate cells")
		return nil
	}

	updateUIWithData(string(jsonb))
	return nil
}

func computeFeatureCells(f *geojson.Feature) (s2.CellUnion, error) {
	gd := &geodata.GeoData{}
	err := geodata.GeoJSONFeatureToGeoData(f, gd)
	if err != nil {
		return nil, err
	}

	minLevel, maxLevel, maxCells, insideCover := getCoverParams()
	coverer := &s2.RegionCoverer{MinLevel: minLevel, MaxLevel: maxLevel, MaxCells: maxCells}

	var cu s2.CellUnion
	if insideCover {
		icu, err := gd.InteriorCover(coverer)
		if err != nil {
			return nil, fmt.Errorf("error in Interior Cover %w", err)
		}
		cu = icu
	} else {
		icu, err := gd.Cover(coverer)
		if err != nil {
			return nil, fmt.Errorf("error in Exterior Cover %w", err)
		}
		cu = icu
	}

	return cu, nil
}

func drawCells(this js.Value, i []js.Value) interface{} {
	un := make(map[s2.CellID]struct{})
	for _, cs := range i {
		cs := js.ValueOf(cs).String()
		if cs != "" {
			c := s2tools.ParseCellID(cs)
			if c == nil {
				continue
			}
			un[*c] = struct{}{}
		}
	}

	cells := make(s2.CellUnion, len(un))
	count := 0
	for c := range un {
		cells[count] = c
		count++
	}
	b := s2tools.CellUnionToGeoJSON(cells)
	updateUIWithData(string(b))
	return nil
}

func updateUIWithData(data string) {
	js.Global().Set("data", data)
	js.Global().Call("updateui")
}

func registerCallbacks() {
	js.Global().Set("drawcells", js.FuncOf(drawCells))
	js.Global().Set("circlecell", js.FuncOf(geoCircleToCells))
	js.Global().Set("geocell", js.FuncOf(geoJSONToCells))
	js.Global().Set("geofeaturescell", js.FuncOf(geoFeaturesJSONToCells))
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
