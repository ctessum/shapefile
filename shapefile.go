package shapefile

import (
	"io"
)

type Shapefile struct {
	Header  *MainFileHeader
	Records []*Record
	f       io.Reader
}

type Record struct {
	Type     ShapeType
	header   *mainFileRecordHeader
	Geometry geometry
}

// Convert record coordinates to well-known text (WKT)
func (rec *Record) ToWKT() {
	rec.geometry.towkt()
}

type geometry interface {
	towkt() string
}

func NewShapefile(rdr io.Reader) (s *Shapefile, err error) {
	s = &Shapefile{}

	var h *MainFileHeader
	if h, err = newMainFileHeaderFromReader(rdr); err != nil {
		return
	}

	s.Header = h
	i := s.Header.FileLength - 50 // length of header = 100 bytes = 50 words
	var rh *mainFileRecordHeader
	var rec *Record
	for {
		if i <= 0 {
			break
		}
		if rh, err = newMainFileRecordHeaderFromReader(rdr); err != nil {
			return
		}
		i = i - rh.ContentLength - 4
		rec = new(Record)
		rec.header = rh
		if err = rec.recordContent(rdr); err != nil {
			return
		}
		s.Records = append(s.Records, rec)
	}
	return
}
