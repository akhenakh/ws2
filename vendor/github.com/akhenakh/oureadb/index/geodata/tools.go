package geodata

import (
	"fmt"

	"github.com/golang/geo/s2"
	"github.com/golang/protobuf/ptypes/struct"
	spb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// GeomToGeoData update gd with geo data gathered from g
func GeomToGeoData(g geom.T, gd *GeoData) error {
	geo := &Geometry{}

	switch g := g.(type) {
	case *geom.Point:
		geo.Coordinates = g.Coords()
		geo.Type = Geometry_POINT
		//case *geom.MultiPolygon:
		//	geo.Type = geodata.Geometry_MULTIPOLYGON

	case *geom.Polygon:
		// only supports outer ring
		geo.Type = Geometry_POLYGON
		geo.Coordinates = g.FlatCoords()

	case *geom.LineString:
		geo.Type = Geometry_LINESTRING
		geo.Coordinates = g.FlatCoords()

	default:
		return errors.Errorf("unsupported geo type %T", g)
	}

	gd.Geometry = geo
	return nil
}

// GeoDataToGeom converts GeoData to a geom.T representation
func GeoDataToGeom(gd *GeoData) (geom.T, error) {
	switch gd.Geometry.Type {
	case Geometry_POINT:
		return geom.NewPointFlat(geom.XY, gd.Geometry.Coordinates), nil
	case Geometry_POLYGON:
		return geom.NewPolygonFlat(geom.XY, gd.Geometry.Coordinates, []int{len(gd.Geometry.Coordinates)}), nil
	case Geometry_LINESTRING:
		return geom.NewLineStringFlat(geom.XY, gd.Geometry.Coordinates), nil
	default:
		return nil, errors.Errorf("unsupported geodata type")
	}
}

// GeoJSONFeatureToGeoData fill gd with the GeoJSON data f
func GeoJSONFeatureToGeoData(f *geojson.Feature, gd *GeoData) error {
	err := PropertiesToGeoData(f, gd)
	if err != nil {
		return errors.Wrap(err, "while converting feature properties to GeoData")
	}

	err = GeomToGeoData(f.Geometry, gd)
	if err != nil {
		return errors.Wrap(err, "while converting feature to GeoData")
	}

	return nil
}

// PropertiesToGeoData update gd.Properties with the properties found in f
func PropertiesToGeoData(f *geojson.Feature, gd *GeoData) error {
	m := make(map[string]*structpb.Value)
	for k, vi := range f.Properties {
		switch tv := vi.(type) {
		case bool:
			m[k] = &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: tv}}
		case int:
			m[k] = &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(tv)}}
		case string:
			m[k] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: tv}}
		case float64:
			m[k] = &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: tv}}
		case nil:
			// pass
		default:
			return fmt.Errorf("GeoJSON property %s unsupported type %T", k, tv)
		}
	}
	if gd.Properties == nil && len(m) > 0 {
		gd.Properties = make(map[string]*structpb.Value)
	}
	for k, v := range m {
		gd.Properties[k] = v
	}
	return nil
}

// GeoDataToRect generate a RectBound for GeoData gd
// only works with Polygons & LineString
func GeoDataToRect(gd *GeoData) (s2.Rect, error) {
	if gd.Geometry == nil {
		return s2.Rect{}, errors.New("invalid geometry")
	}
	switch gd.Geometry.Type {
	case Geometry_POINT:
		return s2.Rect{}, errors.New("point can't be rect bounded")

	case Geometry_POLYGON:
		l := LoopFromCoordinates(gd.Geometry.Coordinates)
		if l.IsEmpty() || l.IsFull() || l.ContainsOrigin() {
			return s2.Rect{}, errors.New("invalid polygon")
		}
		return l.RectBound(), nil

	case Geometry_MULTIPOLYGON:
		return s2.Rect{}, errors.New("multipolygon not supported")

	case Geometry_LINESTRING:
		if len(gd.Geometry.Coordinates)%2 != 0 {
			return s2.Rect{}, errors.New("invalid coordinates count for line")
		}

		pl := make(s2.Polyline, len(gd.Geometry.Coordinates)/2)
		for i := 0; i < len(gd.Geometry.Coordinates); i += 2 {
			ll := s2.LatLngFromDegrees(gd.Geometry.Coordinates[i+1], gd.Geometry.Coordinates[i])
			pl[i/2] = s2.PointFromLatLng(ll)
		}

		return pl.RectBound(), nil

	default:
		return s2.Rect{}, errors.New("unsupported data type")
	}

	return s2.Rect{}, nil
}

// GeoDataToCellUnion generate an s2 cover for GeoData gd
func GeoDataToCellUnion(gd *GeoData, coverer *s2.RegionCoverer) (s2.CellUnion, error) {
	if gd.Geometry == nil {
		return nil, errors.New("invalid geometry")
	}
	var cu s2.CellUnion
	switch gd.Geometry.Type {
	case Geometry_POINT:
		c := s2.CellIDFromLatLng(s2.LatLngFromDegrees(gd.Geometry.Coordinates[1], gd.Geometry.Coordinates[0]))
		cu = append(cu, c.Parent(coverer.MinLevel))

	case Geometry_POLYGON:
		cup, err := coverPolygon(gd.Geometry.Coordinates, coverer)
		if err != nil {
			return nil, errors.Wrap(err, "can't cover polygon")
		}
		cu = append(cu, cup...)

	case Geometry_MULTIPOLYGON:
		for _, g := range gd.Geometry.Geometries {
			cup, err := coverPolygon(g.Coordinates, coverer)
			if err != nil {
				return nil, errors.Wrap(err, "can't cover multipolygon")
			}

			cu = append(cu, cup...)
		}

	case Geometry_LINESTRING:
		if len(gd.Geometry.Coordinates)%2 != 0 {
			return nil, errors.New("invalid coordinates count for line")
		}

		pl := make(s2.Polyline, len(gd.Geometry.Coordinates)/2)
		for i := 0; i < len(gd.Geometry.Coordinates); i += 2 {
			ll := s2.LatLngFromDegrees(gd.Geometry.Coordinates[i+1], gd.Geometry.Coordinates[i])
			pl[i/2] = s2.PointFromLatLng(ll)
		}

		cupl := coverer.Covering(&pl)
		cu = append(cu, cupl...)

	default:
		return nil, errors.New("unsupported data type")
	}

	return cu, nil
}

// returns an s2 cover from a list of lng, lat forming a closed polygon
func coverPolygon(c []float64, coverer *s2.RegionCoverer) (s2.CellUnion, error) {
	if len(c) < 6 {
		return nil, errors.New("invalid polygons not enough coordinates for a closed polygon")
	}
	if len(c)%2 != 0 {
		return nil, errors.New("invalid polygons odd coordinates number")
	}
	l := LoopFromCoordinates(c)
	if l.IsEmpty() || l.IsFull() || l.ContainsOrigin() {
		return nil, errors.New("invalid polygons")
	}

	return coverer.Covering(l), nil
}

// ToGeoJSONFeatureCollection converts a list of GeoData to a GeoJSON Feature Collection
func ToGeoJSONFeatureCollection(geos []*GeoData) ([]byte, error) {
	fc := geojson.FeatureCollection{}
	for _, g := range geos {
		f := &geojson.Feature{}
		switch g.Geometry.Type {
		case Geometry_POINT:
			ng := geom.NewPointFlat(geom.XY, g.Geometry.Coordinates)
			f.Geometry = ng
		case Geometry_POLYGON:
			ng := geom.NewPolygonFlat(geom.XY, g.Geometry.Coordinates, []int{len(g.Geometry.Coordinates)})
			f.Geometry = ng
		case Geometry_MULTIPOLYGON:
			mp := geom.NewMultiPolygon(geom.XY)
			for _, poly := range g.Geometry.Geometries {
				ng := geom.NewPolygonFlat(geom.XY, poly.Coordinates, []int{len(poly.Coordinates)})
				mp.Push(ng)
			}
			f.Geometry = mp
		case Geometry_LINESTRING:
			ls := geom.NewLineStringFlat(geom.XY, g.Geometry.Coordinates)
			f.Geometry = ls
		}
		f.Properties = PropertiesToJSONMap(g.Properties)
		fc.Features = append(fc.Features, f)
	}

	return fc.MarshalJSON()
}

// PointsToGeoJSONPolyLines converts a list of GeoData containing points to a polylines GeoJSON
func PointsToGeoJSONPolyLines(geos []*GeoData) ([]byte, error) {
	f := geojson.Feature{}
	var flatCoords []float64

	if len(geos) == 0 {
		return f.MarshalJSON()
	}

	for _, g := range geos {
		switch g.Geometry.Type {
		case Geometry_POINT:
			flatCoords = append(flatCoords, g.Geometry.Coordinates...)
		default:
			return nil, errors.Errorf("unsupported geometry")
		}

	}
	f.Properties = PropertiesToJSONMap(geos[0].Properties)
	g := geom.NewLineStringFlat(geom.XY, flatCoords)
	f.Geometry = g

	return f.MarshalJSON()
}

// PropertiesToJSONMap converts a protobuf map to it's JSON serializable map equivalent
func PropertiesToJSONMap(src map[string]*spb.Value) map[string]interface{} {
	res := make(map[string]interface{})

	for k, v := range src {
		switch x := v.Kind.(type) {
		case *spb.Value_NumberValue:
			res[k] = x.NumberValue
		case *spb.Value_StringValue:
			res[k] = x.StringValue
		case *spb.Value_BoolValue:
			res[k] = x.BoolValue
		}
	}
	return res
}
