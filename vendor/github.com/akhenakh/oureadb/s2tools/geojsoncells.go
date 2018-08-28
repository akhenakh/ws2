package s2tools

import (
	"github.com/golang/geo/s2"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// CellUnionToGeoJSON helpers to display s2 cells on maps with GeoJSON
// exports cell union into its GeoJSON representation
func CellUnionToGeoJSON(cu s2.CellUnion) []byte {
	fc := geojson.FeatureCollection{}
	for _, cid := range cu {
		f := &geojson.Feature{}
		f.Properties = make(map[string]interface{})
		f.Properties["id"] = cid.ToToken()
		f.Properties["uid"] = uint64(cid)
		f.Properties["level"] = cid.Level()

		c := s2.CellFromCellID(cid)
		coords := make([]float64, 5*2)
		for i := 0; i < 4; i++ {
			p := c.Vertex(i)
			ll := s2.LatLngFromPoint(p)
			coords[i*2] = ll.Lng.Degrees()
			coords[i*2+1] = ll.Lat.Degrees()
		}
		// last is first
		coords[8], coords[9] = coords[0], coords[1]
		ng := geom.NewPolygonFlat(geom.XY, coords, []int{10})
		f.Geometry = ng
		fc.Features = append(fc.Features, f)
	}
	b, _ := fc.MarshalJSON()
	return b
}

// CellUnionToTokens a cell union to a token string list
func CellUnionToTokens(cu s2.CellUnion) []string {
	res := make([]string, len(cu))

	for i, c := range cu {
		res[i] = c.ToToken()
	}
	return res
}
