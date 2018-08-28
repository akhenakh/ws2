package s2tools

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/akhenakh/oureadb/index/geodata"
	"github.com/golang/geo/s2"
	"github.com/gorilla/mux"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// S2CellQueryHandler returns a GeoJSON containing the cells passed in the query
// ?cells=TokenID,...
func S2CellQueryHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	sval := query.Get("cells")
	cells := strings.Split(sval, ",")
	if len(cells) == 0 {
		http.Error(w, "invalid parameters", 400)
		return
	}

	cu := make(s2.CellUnion, len(cells))

	for i, cs := range cells {
		c := s2.CellIDFromToken(cs)
		cu[i] = c
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(CellUnionToGeoJSON(cu))
}

// GeoJSONToCellHandler expect GeoJSON POST at a given URL:
// /{min_level:[0-9]+}/{max_level:[0-9]+}/{max_cells:[0-9]+}/
// GeoJSON as body, with only one feature inside the file
// curl --data "@test.geojson"  http://localhost:8000/api/geojson/4/10/0 -X POST
func GeoJSONToCellHandler(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()

	var fc geojson.FeatureCollection
	gd := &geodata.GeoData{}

	vars := mux.Vars(r)

	vals, err := func(args []string) ([]int, error) {
		res := make([]int, len(args))
		for i, arg := range args {
			vs := vars[arg]
			v, err := strconv.Atoi(vs)
			if err != nil {
				return res, err
			}
			if v < 0 {
				return res, errors.New("invalid parameter")
			}
			res[i] = v
		}
		return res, nil
	}([]string{"min_level", "max_level", "max_cells"})

	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	err = dec.Decode(&fc)
	if err != nil {
		http.Error(w, err.Error()+" can't unmasrhal JSON", 400)
		return
	}

	if len(fc.Features) != 1 {
		http.Error(w, "no or more than one feature", 400)
		return
	}

	f := fc.Features[0]
	err = geodata.GeoJSONFeatureToGeoData(f, gd)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	coverer := &s2.RegionCoverer{
		MinLevel: vals[0],
		MaxLevel: vals[1],
		MaxCells: vals[2],
	}

	cu, err := geodata.GeoDataToCellUnion(gd, coverer)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	j := CellUnionToGeoJSON(cu)
	w.Write(j)
}
