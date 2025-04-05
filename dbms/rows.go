package dbms

import "database/sql"

type Rows interface {
	Columns() []string
	ColumnTypes() []ColumnType
	First() bool
	Next() bool
	Data() []interface{}
	Err() error
	Last() bool
	Close()
	Events() interface{}
}

type RowsEvents struct {
}

type rows struct {
	cols    []string
	coltpes []ColumnType
	dta     []interface{}
	dtaref  []interface{}
	dbrws   *sql.Rows
	lsterr  error
	first   bool
	last    bool
	evnts   *ReaderEvents
}

// Events implements Rows.
func (r *rows) Events() interface{} {
	if r == nil {

	}
	if evnts := r.evnts; evnts != nil {
		return evnts
	}
	r.evnts = &ReaderEvents{}
	return r.evnts
}

// ColumnTypes implements Rows.
func (r *rows) ColumnTypes() []ColumnType {
	if r == nil {

	}
	if len(r.cols) == 0 && r.dbrws != nil {
		r.Columns()
	}
	return r.coltpes
}

// Data implements Rows.
func (r *rows) Data() []interface{} {
	if r == nil {
		return nil
	}
	return r.dta
}

// Close implements Rows.
func (r *rows) Close() {
	if r == nil {
		return
	}
	dta := r.dta
	r.dta = nil
	r.dtaref = nil
	r.cols = nil
	r.dbrws = nil
	r.lsterr = nil
	for _, d := range dta {
		if rd, rdk := d.(Reader); rdk {
			if rd != nil {
				rd.Close()
			}
		}
	}
}

func (r *rows) First() (first bool) {
	if r == nil {
		return
	}
	return r.first
}

func (r *rows) Columns() []string {
	if r == nil {
		return nil
	}
	cols := r.cols
	if len(cols) > 0 {
		return cols
	}
	if dbrws := r.dbrws; dbrws != nil {
		cltpes, cltpeserr := dbrws.ColumnTypes()
		if cltpeserr == nil && len(cltpes) > 0 {
			cols = make([]string, len(cltpes))
			coltpes := make([]ColumnType, len(cltpes))
			for cn, cltp := range cltpes {
				coltpes[cn] = &columntype{dbcoltype: cltp}
				cols[cn] = coltpes[cn].Name()
			}
			r.cols = cols
			r.coltpes = coltpes
			return cols
		}
		if cols, r.lsterr = dbrws.Columns(); r.lsterr != nil {
			return nil
		}
		r.cols = cols
		return cols
	}
	return nil
}

func (r *rows) Last() (last bool) {
	if r == nil {
		return
	}
	return r.last
}

// Err implements Rows.
func (r *rows) Err() error {
	if r == nil {
		return nil
	}
	return r.lsterr
}

// Next implements Rows.
func (r *rows) Next() (nxt bool) {
	if r == nil {
		return false
	}
	if dbrws := r.dbrws; dbrws != nil {
		if !r.first {
			if len(r.cols) == 0 {
				if r.Columns(); r.lsterr != nil {
					return
				}
			}
			if nxt, r.lsterr = dbrws.Next(), dbrws.Err(); r.lsterr == nil && nxt {
				r.first = true
				dta, dtaref := r.dta, r.dtaref
				if len(dta) < len(r.cols) {
					dta = make([]interface{}, len(r.cols))
					r.dta = dta
					dtaref = make([]interface{}, len(dta))
					for dn := range len(dtaref) {
						dtaref[dn] = &dta[dn]
					}
					r.dtaref = dtaref
				}

				r.lsterr = dbrws.Scan(dtaref...)
				for n, d := range dta {
					if rw, rwk := d.(*sql.Rows); rwk {
						dta[n] = &reader{rws: &rows{dbrws: rw}}
					}
				}
				if nxt = r.lsterr == nil; nxt {
					r.last, r.lsterr = dbrws.Next(), dbrws.Err()
					nxt = r.lsterr == nil
				}
			}
			return
		}
		if nxt = r.last; nxt {
			dta, dtaref := r.dta, r.dtaref
			if cl, dtal, dtarefl := len(r.cols), len(dta), len(dtaref); cl > 0 && cl == dtal && dtarefl == dtal {
				r.lsterr = dbrws.Scan(dtaref...)
				if nxt = r.lsterr == nil; nxt {
					r.last, r.lsterr = dbrws.Next(), dbrws.Err()
					nxt = r.lsterr == nil
				}
			}
			return
		}
		nxt = false
	}
	return
}
