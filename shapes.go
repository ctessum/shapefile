package shapefile

import (
	"encoding/binary"
	"fmt"
	"io"
)

var L = binary.LittleEndian
var B = binary.BigEndian

func (rec *Record) recordContent(r io.Reader) (err error) {
	if err = binary.Read(r, L, &rec.Type); err != nil {
		return err
	}
	// this implementation does not enforce the rule that all
	// records in a file must be the same type.
	switch rec.Type {
	case NULL_SHAPE:
		rec.content, err = readNull(r)
	case POINT:
		rec.content, err = readPoint(r)
	case POLY_LINE:
		rec.content, err = readPolyLine(r)
	case POLYGON:
		rec.content, err = readPolygon(r)
	case MULTI_POINT:
		rec.content, err = readMultiPoint(r)
	case POINT_Z:
		rec.content, err = readPointZ(r)
	case POLY_LINE_Z:
		rec.content, err = readPolyLineZ(r)
	case POLYGON_Z:
		rec.content, err = readPolygonZ(r)
	case MULTI_POINT_Z:
		rec.content, err = readMultiPointZ(r)
	case POINT_M:
		rec.content, err = readPointM(r)
	case POLY_LINE_M:
		rec.content, err = readPolyLineM(r)
	case POLYGON_M:
		rec.content, err = readPolygonM(r)
	case MULTI_POINT_M:
		rec.content, err = readMultiPointM(r)
	case MULTI_PATCH:
		rec.content, err = readMultiPatch(r)
	default:
		err = fmt.Errorf("unknown shape type: %d", rec.Type)
		return
	}
	return
}

type Null struct {
}

func readNull(r io.Reader) (n *Null, err error) {
	return &Null{}, nil
}

func (g *Null) towkt() string {
	return ""
}

type Point struct {
	//_ ShapeType // Point is an embedded type
	X float64
	Y float64
}

func (g *Point) towkt() string {
	return fmt.Sprintf("POINT (%v %v)", g.X, g.Y)
}

func readPoint(r io.Reader) (p *Point, err error) {
	p = &Point{}
	err = binary.Read(r, binary.LittleEndian, p)
	return
}

type Box struct {
	Xmin float64
	Ymin float64
	Xmax float64
	Ymax float64
}

func (g *Box) towkt() string {
	return fmt.Sprintf("POLYGON ((%v %v, %v %v, %v %v, %v %v, %v %v))",
		g.Xmin, g.Ymin, g.Xmax, g.Ymin, g.Xmax, g.Ymax,
		g.Xmin, g.Ymax, g.Xmin, g.Ymin)
}

// reads a succession of numPoints, Point[numPoints]...
func readNumPoints(r io.Reader) (points []Point, err error) {
	var i int32
	if err = binary.Read(r, L, &i); err != nil {
		return
	}
	points = make([]Point, i)
	err = binary.Read(r, L, points)
	return
}

type MultiPoint struct {
	Box    Box
	Points []Point
}

func readMultiPoint(r io.Reader) (mp *MultiPoint, err error) {
	mp = &MultiPoint{}
	if err = binary.Read(r, binary.LittleEndian, &mp.Box); err != nil {
		return
	}
	if mp.Points, err = readNumPoints(r); err != nil {
		return
	}
	return
}

func (g *MultiPoint) towkt() string {
	s := "MULTIPOINT ("
	for i, p := range g.Points {
		s += fmt.Sprintf("(%v %v)", p.X, p.Y)
		if i != len(g.Points)-1 {
			s += ","
		}
	}
	s += ")"
	return s
}

func readPartsPoints(r io.Reader) (parts []int32, points []Point, err error) {
	var nprts int32
	var npts int32
	if err = binary.Read(r, L, &nprts); err != nil {
		return
	}
	if err = binary.Read(r, L, &npts); err != nil {
		return
	}

	parts = make([]int32, nprts)
	if err = binary.Read(r, L, parts); err != nil {
		return
	}
	points = make([]Point, npts)
	err = binary.Read(r, L, points)
	return
}

type PolyLine struct {
	Box    Box
	Parts  []int32
	Points []Point
}

func readPolyLine(r io.Reader) (pl *PolyLine, err error) {
	pl = &PolyLine{}
	if err = binary.Read(r, L, &pl.Box); err != nil {
		return
	}
	pl.Parts, pl.Points, err = readPartsPoints(r)
	return
}

func (g *PolyLine) towkt() string {
	var start, end int32
	s := "MULTILINESTRING ("
	for i := 0; i < len(g.Parts); i++ {
		start = g.Parts[i]
		if i == len(g.Parts)-1 {
			end = len(g.Points)
		} else {
			end = g.Parts[i+1]
		}
		s += "("
		for j := start; j < end; j++ {

			if j != end-1 {
				s += ","
			}
		}
		s += ")"
		if i != len(g.Parts)-1 {
			s += ","
		}
	}
	for i, p := range g.Points {
		s += fmt.Sprintf("%v %v", p.X, p.Y)
		if i != len(g.Points)-1 {
			s += ","
		}
	}
	s += ")"
	return s
}

func (p *PolyLine) String() string {
	str := fmt.Sprintf("Box: \n%s", p.Box.String())
	//str += fmt.Sprintf("NumParts: %d\n", p.NumParts)
	//str += fmt.Sprintf("NumPoints: %d\n", p.NumPoints)
	for i, p := range p.Parts {
		str += fmt.Sprintf("part %d : %d\n", i, p)
	}
	for i, p := range p.Points {
		str += fmt.Sprintf("point %d : %s", i, p.String())
	}
	return str
}

type Polygon struct {
	PolyLine
}

func (p *Polygon) String() string {
	str := fmt.Sprintf("Box: %s\n", p.Box.String())
	//str += fmt.Sprintf("NumParts: %d\n", p.NumParts)
	//str += fmt.Sprintf("NumPoints: %d\n", p.NumPoints)
	for i, p := range p.Parts {
		str += fmt.Sprintf("part %d : %d\n", i, p)
	}
	for i, p := range p.Points {
		str += fmt.Sprintf("point %d : %s\n", i, p.String())
	}
	return str

}

func readPolygon(r io.Reader) (pg *Polygon, err error) {
	pg = &Polygon{}
	if err = binary.Read(r, L, &pg.Box); err != nil {
		return
	}
	pg.Parts, pg.Points, err = readPartsPoints(r)
	return

}

type PointM struct {
	X float64
	Y float64
	M float64
}

func readPointM(r io.Reader) (pm *PointM, err error) {
	pm = &PointM{}
	err = binary.Read(r, L, pm)
	return
}

type MRange struct {
	Mmin float64
	Mmax float64
}
type MultiPointM struct {
	Box    Box
	Points []Point
	MRange MRange    // optional
	MArray []float64 // optional
}

func readMultiPointM(r io.Reader) (mp *MultiPointM, err error) {
	mp = &MultiPointM{}
	if err = binary.Read(r, L, &mp.Box); err != nil {
		return
	}

	if mp.Points, err = readNumPoints(r); err != nil {
		return
	}

	if err = binary.Read(r, L, &mp.MRange); err != nil {
		return
	}
	mp.MArray = make([]float64, len(mp.Points))
	err = binary.Read(r, L, mp.MArray)
	return

}

type PolyLineM struct {
	Box    Box
	Parts  []int32
	Points []Point
	MRange MRange    // optional
	MArray []float64 // optional
}

func readPolyLineM(r io.Reader) (pl *PolyLineM, err error) {
	pl = &PolyLineM{}
	if err = binary.Read(r, L, pl.Box); err != nil {
		return
	}

	if pl.Parts, pl.Points, err = readPartsPoints(r); err != nil {
		return
	}

	if err = binary.Read(r, L, pl.MRange); err != nil {
		return
	}
	pl.MArray = make([]float64, len(pl.Points))

	err = binary.Read(r, L, pl.MArray)
	return
}

type PolygonM struct {
	PolyLineM
}

func readPolygonM(r io.Reader) (pg *PolygonM, err error) {
	pg = &PolygonM{}
	if err = binary.Read(r, L, pg.Box); err != nil {
		return
	}
	if pg.Parts, pg.Points, err = readPartsPoints(r); err != nil {
		return
	}

	if err = binary.Read(r, L, pg.MRange); err != nil {
		return
	}
	pg.MArray = make([]float64, len(pg.Points))

	err = binary.Read(r, L, pg.MArray)
	return
}

type PointZ struct {
	X float64
	Y float64
	Z float64
	M float64
}

func readPointZ(r io.Reader) (p *PointZ, err error) {
	p = &PointZ{}
	err = binary.Read(r, L, &p)
	return

}

type ZRange struct {
	Zmin float64
	Zmax float64
}
type MultiPointZ struct {
	Box    Box
	Points []Point
	ZRange ZRange
	ZArray []float64
	MRange MRange    // optional
	MArray []float64 // optional
}

func readMultiPointZ(r io.Reader) (mp *MultiPointZ, err error) {
	mp = &MultiPointZ{}
	if err = binary.Read(r, L, mp.Box); err != nil {
		return
	}
	if mp.Points, err = readNumPoints(r); err != nil {
		return
	}
	if err = binary.Read(r, L, mp.ZRange); err != nil {
		return
	}
	mp.ZArray = make([]float64, len(mp.Points))
	if err = binary.Read(r, L, mp.ZArray); err != nil {
		return
	}
	if err = binary.Read(r, L, mp.MRange); err != nil {
		return
	}
	mp.MArray = make([]float64, len(mp.Points))

	err = binary.Read(r, L, mp.MArray)
	return
}

type PolyLineZ struct {
	Box       Box
	NumParts  int32
	NumPoints int32
	Parts     []int32
	Points    []Point
	ZRange    ZRange
	ZArray    []float64 //optional
	MRange    MRange    // optional
	MArray    []float64 //optional
}

func readPolyLineZ(r io.Reader) (pl *PolyLineZ, err error) {
	pl = &PolyLineZ{}

	if err = binary.Read(r, L, pl.Box); err != nil {
		return
	}
	if pl.Parts, pl.Points, err = readPartsPoints(r); err != nil {
		return
	}

	if err = binary.Read(r, L, pl.ZRange); err != nil {
		return
	}

	pl.ZArray = make([]float64, len(pl.Points))
	if err = binary.Read(r, L, pl.ZArray); err != nil {
		return
	}
	if err = binary.Read(r, L, pl.MRange); err != nil {
		return
	}
	pl.MArray = make([]float64, len(pl.Points))
	if err = binary.Read(r, L, pl.MArray); err != nil {
		return
	}
	return

}

type PolygonZ struct {
	PolyLineZ
}

func readPolygonZ(r io.Reader) (pg *PolygonZ, err error) {
	pg = &PolygonZ{}

	if err = binary.Read(r, L, pg.Box); err != nil {
		return
	}
	if pg.Parts, pg.Points, err = readPartsPoints(r); err != nil {
		return
	}

	if err = binary.Read(r, L, pg.ZRange); err != nil {
		return
	}

	pg.ZArray = make([]float64, len(pg.Points))
	if err = binary.Read(r, L, pg.ZArray); err != nil {
		return
	}
	if err = binary.Read(r, L, pg.MRange); err != nil {
		return
	}
	pg.MArray = make([]float64, len(pg.Points))
	if err = binary.Read(r, L, pg.MArray); err != nil {
		return
	}
	return

}

type PartType int32

const (
	TRIANGLE_STRIP PartType = iota
	TRIANGLE_FAN
	OUTER_RING
	INNER_RING
	FIRST_RING
	RING
)

func (p PartType) String() string {
	switch p {
	case TRIANGLE_STRIP:
		return "TRIANGLE_STRIP"
	case TRIANGLE_FAN:
		return "TRIANGLE_FAN"
	case OUTER_RING:
		return "OUTER_RING"
	case INNER_RING:
		return "INNER_RING"
	case FIRST_RING:
		return "FIRST_RING"
	case RING:
		return "RING"
	default:
		return "UNKNOWN"
	}
}

type MultiPatch struct {
	Box       Box
	Parts     []int32
	PartTypes []PartType
	Points    []Point
	ZRange    ZRange    // optional
	MRange    MRange    // optional
	MArray    []float64 // optional
}

func readMultiPatch(r io.Reader) (mp *MultiPatch, err error) {
	mp = &MultiPatch{}
	if err = binary.Read(r, L, mp.Box); err != nil {
		return
	}

	var prts int32
	if err = binary.Read(r, L, &prts); err != nil {
		return
	}
	var pts int32
	if err = binary.Read(r, L, &pts); err != nil {
		return
	}
	mp.Parts = make([]int32, prts)
	if err = binary.Read(r, L, &mp.Parts); err != nil {
		return
	}
	mp.PartTypes = make([]PartType, prts)
	if err = binary.Read(r, L, &mp.PartTypes); err != nil {
		return
	}
	mp.Points = make([]Point, pts)
	if err = binary.Read(r, L, mp.Points); err != nil {
		return
	}
	if err = binary.Read(r, L, mp.ZRange); err != nil {
		return
	}
	if err = binary.Read(r, L, mp.MRange); err != nil {
		return
	}
	mp.MArray = make([]float64, pts)
	err = binary.Read(r, L, mp.MArray)
	return

}
