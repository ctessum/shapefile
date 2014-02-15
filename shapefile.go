package shapefile

import (
	"github.com/twpayne/gogeom/geom"
	"io"
)

type Shapefile struct {
	Header *ShapefileHeader
	rdr    io.Reader
	i      int32 // file cursor [words]
}

type ShapefileRecord struct {
	Type     ShapeType
	header   *shapefileRecordHeader
	Bounds   *geom.Bounds
	Geometry geom.T
}

// Open shapefile for reading.
func OpenShapefile(rdr io.Reader) (s *Shapefile, err error) {
	s = &Shapefile{}
	s.rdr = rdr

	var h *ShapefileHeader
	if h, err = newShapefileHeaderFromReader(s.rdr); err != nil {
		return
	}

	s.Header = h
	s.i = s.Header.FileLength - 50 // length of header = 100 bytes = 50 words
	return
}

// Get next record in file. If end of file, err=io.EOF.
func (s *Shapefile) NextRecord() (rec *ShapefileRecord, err error) {
	rec = new(ShapefileRecord)
	if s.i <= 0 {
		err = io.EOF
		return
	}
	if rec.header, err = newShapefileRecordHeaderFromReader(s.rdr); err != nil {
		return
	}
	if err = rec.recordContent(s.rdr); err != nil {
		return
	}
	s.i = s.i - rec.header.ContentLength - 4
	return
}
