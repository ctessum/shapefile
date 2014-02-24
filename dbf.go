package shapefile

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// DBF is documented here: http://www.clicketyclick.dk/databases/xbase/format/dbf.html

type DBFFile struct {
	DBFFileHeader    *DBFFileHeader
	FieldDescriptors []FieldDescriptor
	FieldIndicies    map[string]int // indicies of each field by name
	countRead        uint32
	r                io.Reader
}

func OpenDBFFile(r io.Reader) (dbf *DBFFile, err error) {
	dbf = &DBFFile{}
	dbf.r = r
	if dbf.DBFFileHeader, err = newDBFFileHeader(r); err != nil {
		return
	}
	len_fd := dbf.DBFFileHeader.LenHeader - 32 // the fixed portion of the header is 32 bytes
	num_fd := (int)(len_fd / 32)               // each field descriptor are 32 bytes each, see below.
	dbf.FieldIndicies = make(map[string]int)
	var fd FieldDescriptor
	for i := 0; i != num_fd; i++ {
		if err = binary.Read(dbf.r, l, &fd); err != nil {
			return
		}
		dbf.FieldDescriptors = append(dbf.FieldDescriptors, fd)
		dbf.FieldIndicies[fd.fieldName()] = i
	}
	bullshitByte := make([]byte, 1)
	var n int
	if n, err = r.Read(bullshitByte); err != nil || n != 1 {
		if err == nil {
			err = fmt.Errorf("couldn't read bullshit byte!")
		}
		return
	}
	dbf.countRead = (uint32)(0)
	return
}

// Get next record in file. If end of file, err=io.EOF.
func (dbf *DBFFile) NextRecord() (entry []interface{}, err error) {
	if dbf.countRead == dbf.DBFFileHeader.NumRecords {
		err = io.EOF
		return
	}

	rawEntry := make([]byte, dbf.DBFFileHeader.LenRecord)
	var n int
	if n, err = dbf.r.Read(rawEntry); (err != nil) || n != (int)(dbf.DBFFileHeader.LenRecord) {
		if err == nil {
			err = fmt.Errorf("expected %d bytes, read: %d", dbf.DBFFileHeader.LenRecord, n)
		}
		return
	}
	if 0x2a == rawEntry[0] { // record deleted
		return
	}

	entry = make([]interface{}, len(dbf.FieldDescriptors))
	var offset = 1

	for i, desc := range dbf.FieldDescriptors {
		rawField := rawEntry[offset : offset+(int)(desc.FieldLength)]
		offset += (int)(desc.FieldLength)

		switch desc.FieldType {
		case Character, VarCharVar:
			entry[i] = strings.TrimSpace((string)(rawField))
		case Number, Integer:
			if desc.DecimalCount == 0 {
				var val int64
				numberStr := strings.TrimSpace((string)(rawField))
				if val, err = strconv.ParseInt(numberStr, 10, 64); err != nil {
					return
				}
				entry[i] = int(val)
				break
			}
			// handle it like a float ...
			fallthrough
		case Float, Double:
			numberStr := strings.TrimSpace((string)(rawField))
			if entry[i], err = strconv.ParseFloat(numberStr, 64); err != nil {
				// If the float isn't valid, return a the error message
				// in the data field and let the calling program handle
				// it.
				entry[i] = err
				err = nil
			}
		case Logical:
			switch (string)(rawField) {
			case "1", "T", "t", "Y", "y":
				entry[i] = true
			case "0", "F", "f", "N", "n":
				entry[i] = false
			default:
				err = fmt.Errorf("Unsupported logical value `%v`",
					(string)(rawField))
				return
			}
		default:
			err = fmt.Errorf("unsupported type: %c", desc.FieldType)
		}
	}
	dbf.countRead++
	return
}

// http://www.clicketyclick.dk/databases/xbase/format/dbf.html#DBF_STRUCT

type DBFFileHeader struct {
	Version        byte
	LastUpdate     [3]uint8 // YY MM DD (YY = years since 1900)
	NumRecords     uint32   // LittleEndian
	LenHeader      uint16
	LenRecord      uint16
	_              [2]byte // reserved
	IncompleteTx   byte
	EncFlag        byte
	FreeRecThread  uint32 // ...
	_              [8]byte
	MDXFlag        byte
	LanguageDriver byte
	_              [2]byte
}

func (hdr *DBFFileHeader) String() string {
	str := fmt.Sprintf("Version     : %d\n", hdr.Version)
	str += fmt.Sprintf("Last Update : %d %d %d\n", hdr.LastUpdate[0], hdr.LastUpdate[1], hdr.LastUpdate[2])
	str += fmt.Sprintf("Num Records : %d\n", hdr.NumRecords)
	str += fmt.Sprintf("Len Header  : %d\n", hdr.LenHeader)
	str += fmt.Sprintf("Len Record  : %d\n", hdr.LenRecord)
	return str
}

func newDBFFileHeader(r io.Reader) (hdr *DBFFileHeader, err error) {
	hdr = &DBFFileHeader{}
	err = binary.Read(r, l, hdr)
	return
}

type FieldType byte

const (
	Character FieldType = 'C'
	Number              = 'N'
	Logical             = 'L'
	Date                = 'D'
	Memo                = 'M'
	Float               = 'F'
	// VarChar = ???
	Binary        = 'B'
	General       = 'G'
	Picture       = 'P'
	Currency      = 'Y'
	DateTime      = 'T'
	Integer       = 'I'
	VariField     = 'V'
	VarCharVar    = 'X'
	Timestamp     = '@'
	Double        = 'O' // 8 bytes
	Autoincrement = '+'
)

type FieldDescriptor struct {
	FieldName_     [11]byte
	FieldType      FieldType
	FieldDataAddr  uint32
	FieldLength    uint8
	DecimalCount   uint8
	_              [2]byte
	WorkAreaID     byte
	_              [2]byte
	FlagSetField   byte
	_              [7]byte
	IndexFieldFlag byte
}

func (f *FieldDescriptor) String() string {
	str := fmt.Sprintf("Name : %s\n", f.fieldName())
	str += fmt.Sprintf("Type : %c\n", f.FieldType)
	str += fmt.Sprintf("Len  : %d\n", f.FieldLength)
	str += fmt.Sprintf("Count: %d\n", f.DecimalCount)
	return str
}
func (f *FieldDescriptor) fieldName() string {
	for i, b := range f.FieldName_ {
		if b == '\000' {
			return strings.TrimSpace((string)(f.FieldName_[0:i]))
		}
	}
	return ""
}
