package dbms

import "strings"

type Record interface {
	First() bool
	Last() bool
	Data() []interface{}
	Columns() []string
	RowNR() int
	ColumnTypes() []ColumnType
	Field(interface{}) interface{}
	Reader(interface{}) Reader
}

type record struct {
	rdr     *reader
	cnt     int
	dta     []interface{}
	cols    []string
	coltpes []ColumnType
	first   bool
	last    bool
}

func (rc *record) First() bool {
	if rc == nil {
		return false
	}
	return rc.first
}

func (rc *record) Last() bool {
	if rc == nil {
		return false
	}
	return rc.last
}

func (rc *record) RowNR() int {
	if rc == nil {
		return 0
	}
	return rc.cnt
}

func (rc *record) Data() []interface{} {
	if rc == nil {
		return nil
	}
	return rc.dta
}

func (rc *record) Columns() []string {
	if rc == nil {
		return nil
	}
	return rc.cols
}

func (rc *record) ColumnTypes() []ColumnType {
	if rc == nil {
		return nil
	}
	return rc.coltpes
}

func (rc *record) Field(ref interface{}) interface{} {
	if rc == nil {
		return nil
	}
	if s, sk := ref.(string); sk {
		if s != "" {
			for ci, c := range rc.cols {
				if strings.EqualFold(c, s) {
					if ci < len(rc.dta) {
						return rc.dta[ci]
					}
					return nil
				}
			}
		}
	}
	if idx, _ := ref.(int); idx > -1 && idx < len(rc.dta) {
		return rc.dta[idx]
	}
	return nil
}

func (rc *record) Reader(ref interface{}) Reader {
	if rc == nil {
		return nil
	}

	if s, sk := ref.(string); sk {
		if s != "" {
			for ci, c := range rc.cols {
				if strings.EqualFold(c, s) {
					if ci < len(rc.dta) {
						rdr, _ := rc.dta[ci].(Reader)
						return rdr
					}
					return nil
				}
			}
		}
	}
	if idx, _ := ref.(int); idx > -1 && idx < len(rc.dta) {
		rdr, _ := rc.dta[idx].(Reader)
		return rdr
	}
	return nil
}
