package shapefile

import (
	"encoding/binary"
	"fmt"
	"github.com/twpayne/gogeom/geom"
	"io"
)

var l = binary.LittleEndian
var b = binary.BigEndian

func (rec *ShapefileRecord) recordContent(r io.Reader) (err error) {
	if err = binary.Read(r, l, &rec.Type); err != nil {
		return
	}
	// this implementation does not enforce the rule that all
	// records in a file must be the same type.
	switch rec.Type {
	case NULL_SHAPE:
		rec.Geometry, rec.Bounds, err = readNull(r)
	case POINT:
		rec.Geometry, rec.Bounds, err = readPoint(r)
	case POLY_LINE:
		rec.Geometry, rec.Bounds, err = readPolyLine(r)
	case POLYGON:
		rec.Geometry, rec.Bounds, err = readPolygon(r)
	case MULTI_POINT:
		rec.Geometry, rec.Bounds, err = readMultiPoint(r)
	case POINT_Z:
		rec.Geometry, rec.Bounds, err = readPointZ(r)
	case POLY_LINE_Z:
		rec.Geometry, rec.Bounds, err = readPolyLineZ(r)
	case POLYGON_Z:
		rec.Geometry, rec.Bounds, err = readPolygonZ(r)
	case MULTI_POINT_Z:
		rec.Geometry, rec.Bounds, err = readMultiPointZ(r)
	case POINT_M:
		rec.Geometry, rec.Bounds, err = readPointM(r)
	case POLY_LINE_M:
		rec.Geometry, rec.Bounds, err = readPolyLineM(r)
	case POLYGON_M:
		rec.Geometry, rec.Bounds, err = readPolygonM(r)
	case MULTI_POINT_M:
		rec.Geometry, rec.Bounds, err = readMultiPointM(r)
	// drop multipatch capability for now because don't know
	// what to do with the triangle patches.
	//case MULTI_PATCH:
	//	rec.Geometry, rec.Bounds, err = readMultiPatch(r)
	default:
		err = fmt.Errorf("unknown shape type: %d", rec.Type)
		return
	}
	return
}

func readNull(r io.Reader) (geom.T, *geom.Bounds, error) {
	return nil, nil, nil
}

func readPoint(r io.Reader) (geom.T, *geom.Bounds, error) {
	p := new(geom.Point)
	err := binary.Read(r, binary.LittleEndian, p)
	return *p, nil, err
}

// reads a succession of numPoints, Point[numPoints]...
func readNumPoints(r io.Reader) (points []geom.Point, err error) {
	var i int32
	if err = binary.Read(r, l, &i); err != nil {
		return
	}
	points = make([]geom.Point, i)
	err = binary.Read(r, l, points)
	return
}

func readMultiPoint(r io.Reader) (geom.T, *geom.Bounds, error) {
	mp := new(geom.MultiPoint)
	bounds := new(geom.Bounds)
	var err error
	if err = binary.Read(r, binary.LittleEndian, bounds); err != nil {
		return nil, nil, err
	}
	if mp.Points, err = readNumPoints(r); err != nil {
		return nil, nil, err
	}

	return *mp, bounds, nil
}

func readBoundsPartsPoints(r io.Reader) (bounds *geom.Bounds,
	parts []int32, points []geom.Point, err error) {
	bounds = new(geom.Bounds)
	if err = binary.Read(r, l, bounds); err != nil {
		return
	}
	var nprts int32
	var npts int32
	if err = binary.Read(r, l, &nprts); err != nil {
		return
	}
	if err = binary.Read(r, l, &npts); err != nil {
		return
	}

	parts = make([]int32, nprts)
	if err = binary.Read(r, l, parts); err != nil {
		return
	}
	points = make([]geom.Point, npts)
	err = binary.Read(r, l, points)
	return
}

type xrange struct {
	min float64
	max float64
}

func readBoundsPartsPointsM(r io.Reader) (bounds *geom.Bounds,
	parts []int32, points []geom.Point, M []float64, err error) {
	bounds, parts, points, err = readBoundsPartsPoints(r)
	if err != nil {
		return
	}
	mrange := new(xrange)
	if err = binary.Read(r, l, mrange); err != nil {
		return
	}
	M = make([]float64, len(points))
	err = binary.Read(r, l, M)
	return
}

func readBoundsPartsPointsZM(r io.Reader) (bounds *geom.Bounds,
	parts []int32, points []geom.Point, Z, M []float64, err error) {
	bounds, parts, points, err = readBoundsPartsPoints(r)
	if err != nil {
		return
	}
	zrange := new(xrange)
	if err = binary.Read(r, l, zrange); err != nil {
		return
	}
	Z = make([]float64, len(points))
	err = binary.Read(r, l, Z)
	mrange := new(xrange)
	if err = binary.Read(r, l, mrange); err != nil {
		return
	}
	M = make([]float64, len(points))
	err = binary.Read(r, l, M)
	return
}

func getStartEnd(parts []int32, points []geom.Point, i int) (start, end int) {
	start = int(parts[i])
	if i == len(parts)-1 {
		end = len(points)
	} else {
		end = int(parts[i+1])
	}
	return
}

func readPolyLine(r io.Reader) (geom.T, *geom.Bounds, error) {
	pl := new(geom.MultiLineString)
	bounds, parts, points, err := readBoundsPartsPoints(r)
	if err != nil {
		return nil, nil, err
	}
	pl.LineStrings = make([]geom.LineString, len(parts))
	for i := 0; i < len(parts); i++ {
		start, end := getStartEnd(parts, points, i)
		pl.LineStrings[i].Points = make([]geom.Point, end-start)
		for j := start; j < end; j++ {
			pl.LineStrings[i].Points[j-start] = points[j]
		}
	}
	return *pl, bounds, nil

}

func readPolygon(r io.Reader) (geom.T, *geom.Bounds, error) {
	pg := new(geom.Polygon)
	bounds, parts, points, err := readBoundsPartsPoints(r)
	if err != nil {
		return nil, nil, err
	}
	pg.Rings = make([][]geom.Point, len(parts))
	for i := 0; i < len(parts); i++ {
		start, end := getStartEnd(parts, points, i)
		pg.Rings[i] = make([]geom.Point, end-start)
		// Go backwards around the rings to switch to OGC format
		for j := end - 1; j >= start; j-- {
			pg.Rings[i][j-start] = points[j]
		}
	}
	return *pg, bounds, nil
}

func readPointM(r io.Reader) (geom.T, *geom.Bounds, error) {
	pm := new(geom.PointM)
	err := binary.Read(r, l, pm)
	return pm, nil, err
}

func readMultiPointM(r io.Reader) (geom.T, *geom.Bounds, error) {
	var err error
	mp := new(geom.MultiPointM)
	bounds := new(geom.Bounds)
	if err := binary.Read(r, l, bounds); err != nil {
		return nil, nil, err
	}
	var points []geom.Point
	if points, err = readNumPoints(r); err != nil {
		return nil, nil, err
	}
	mr := new(xrange)
	if err = binary.Read(r, l, mr); err != nil {
		return nil, nil, err
	}
	marray := make([]float64, len(mp.Points))
	if err = binary.Read(r, l, marray); err != nil {
		return nil, nil, err
	}
	mp.Points = make([]geom.PointM, len(points))
	for i, point := range points {
		p := new(geom.PointM)
		p.X = point.X
		p.Y = point.Y
		p.M = marray[i]
		mp.Points[i] = *p
	}
	return *mp, bounds, nil
}

func readPolyLineM(r io.Reader) (geom.T, *geom.Bounds, error) {
	pl := new(geom.MultiLineStringM)
	bounds, parts, points, M, err := readBoundsPartsPointsM(r)
	if err != nil {
		return nil, nil, err
	}
	pl.LineStrings = make([]geom.LineStringM, len(parts))
	for i := 0; i < len(parts); i++ {
		start, end := getStartEnd(parts, points, i)
		pl.LineStrings[i].Points = make([]geom.PointM, end-start)
		for j := start; j < end; j++ {
			p := new(geom.PointM)
			p.X = points[j].X
			p.Y = points[j].Y
			p.M = M[j]
			pl.LineStrings[i].Points[j-start] = *p
		}
	}
	return *pl, bounds, nil
}

func readPolygonM(r io.Reader) (geom.T, *geom.Bounds, error) {
	pg := new(geom.PolygonM)
	bounds, parts, points, M, err := readBoundsPartsPointsM(r)
	if err != nil {
		return nil, nil, err
	}
	pg.Rings = make([][]geom.PointM, len(parts))
	for i := 0; i < len(parts); i++ {
		start, end := getStartEnd(parts, points, i)
		pg.Rings[i] = make([]geom.PointM, end-start)
		// Go backwards around the rings to switch to OGC format
		for j := end - 1; j >= start; j-- {
			p := new(geom.PointM)
			p.X = points[j].X
			p.Y = points[j].Y
			p.M = M[j]
			pg.Rings[i][j-start] = *p
		}
	}
	return *pg, bounds, nil
}

func readPointZ(r io.Reader) (geom.T, *geom.Bounds, error) {
	pzm := new(geom.PointZM)
	err := binary.Read(r, l, pzm)
	return *pzm, nil, err
}

func readMultiPointZ(r io.Reader) (geom.T, *geom.Bounds, error) {
	var err error
	mp := new(geom.MultiPointZM)
	bounds := new(geom.Bounds)
	if err := binary.Read(r, l, bounds); err != nil {
		return nil, nil, err
	}
	var points []geom.Point
	if points, err = readNumPoints(r); err != nil {
		return nil, nil, err
	}
	zr := new(xrange)
	if err = binary.Read(r, l, zr); err != nil {
		return nil, nil, err
	}
	zarray := make([]float64, len(mp.Points))
	if err = binary.Read(r, l, zarray); err != nil {
		return nil, nil, err
	}
	mr := new(xrange)
	if err = binary.Read(r, l, mr); err != nil {
		return nil, nil, err
	}
	marray := make([]float64, len(mp.Points))
	if err = binary.Read(r, l, marray); err != nil {
		return nil, nil, err
	}
	mp.Points = make([]geom.PointZM, len(points))
	for i, point := range points {
		p := new(geom.PointZM)
		p.X = point.X
		p.Y = point.Y
		p.Z = zarray[i]
		p.M = marray[i]
		mp.Points[i] = *p
	}
	return *mp, bounds, nil
}

func readPolyLineZ(r io.Reader) (geom.T, *geom.Bounds, error) {
	pl := new(geom.MultiLineStringZM)
	bounds, parts, points, Z, M, err := readBoundsPartsPointsZM(r)
	if err != nil {
		return nil, nil, err
	}
	pl.LineStrings = make([]geom.LineStringZM, len(parts))
	for i := 0; i < len(parts); i++ {
		start, end := getStartEnd(parts, points, i)
		pl.LineStrings[i].Points = make([]geom.PointZM, end-start)
		for j := start; j < end; j++ {
			p := new(geom.PointZM)
			p.X = points[j].X
			p.Y = points[j].Y
			p.Z = Z[j]
			p.M = M[j]
			pl.LineStrings[i].Points[j-start] = *p
		}
	}
	return *pl, bounds, nil
}

func readPolygonZ(r io.Reader) (geom.T, *geom.Bounds, error) {
	pg := new(geom.PolygonZM)
	bounds, parts, points, Z, M, err := readBoundsPartsPointsZM(r)
	if err != nil {
		return nil, nil, err
	}
	pg.Rings = make([][]geom.PointZM, len(parts))
	for i := 0; i < len(parts); i++ {
		start, end := getStartEnd(parts, points, i)
		pg.Rings[i] = make([]geom.PointZM, end-start)
		// Go backwards around the rings to switch to OGC format
		for j := end - 1; j >= start; j-- {
			p := new(geom.PointZM)
			p.X = points[j].X
			p.Y = points[j].Y
			p.Z = Z[j]
			p.M = M[j]
			pg.Rings[i][j-start] = *p
		}
	}
	return *pg, bounds, nil
}

//type partType int32
//
//const (
//	tRIANGLE_STRIP PartType = iota
//	tRIANGLE_FAN
//	oUTER_RING
//	iNNER_RING
//	fIRST_RING
//	rING
//)
//
//func (p PartType) String() string {
//	switch p {
//	case tRIANGLE_STRIP:
//		return "TRIANGLE_STRIP"
//	case tRIANGLE_FAN:
//		return "TRIANGLE_FAN"
//	case oUTER_RING:
//		return "OUTER_RING"
//	case iNNER_RING:
//		return "INNER_RING"
//	case fIRST_RING:
//		return "FIRST_RING"
//	case rING:
//		return "RING"
//	default:
//		return "UNKNOWN"
//	}
//}
//
//type MultiPatch struct {
//	Box       Box
//	Parts     []int32
//	PartTypes []PartType
//	Points    []Point
//	ZRange    ZRange    // optional
//	MRange    MRange    // optional
//	MArray    []float64 // optional
//}
//
//func readMultiPatch(r io.Reader) (mp *MultiPatch, err error) {
//	mp = &MultiPatch{}
//	if err = binary.Read(r, L, mp.Box); err != nil {
//		return
//	}
//
//	var prts int32
//	if err = binary.Read(r, L, &prts); err != nil {
//		return
//	}
//	var pts int32
//	if err = binary.Read(r, L, &pts); err != nil {
//		return
//	}
//	mp.Parts = make([]int32, prts)
//	if err = binary.Read(r, L, &mp.Parts); err != nil {
//		return
//	}
//	mp.PartTypes = make([]PartType, prts)
//	if err = binary.Read(r, L, &mp.PartTypes); err != nil {
//		return
//	}
//	mp.Points = make([]Point, pts)
//	if err = binary.Read(r, L, mp.Points); err != nil {
//		return
//	}
//	if err = binary.Read(r, L, mp.ZRange); err != nil {
//		return
//	}
//	if err = binary.Read(r, L, mp.MRange); err != nil {
//		return
//	}
//	mp.MArray = make([]float64, pts)
//	err = binary.Read(r, L, mp.MArray)
//	return
//}
