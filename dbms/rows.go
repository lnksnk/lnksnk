package dbms

import "database/sql"

type Rows interface {
	IRows
	First() bool
	Next() bool
	Data() []interface{}
	Err() error
	Last() bool
	Events() interface{}
}

type IRows interface {
	Close() error
	ColumnTypes() ([]ColumnType, error)
	Columns() ([]string, error)
	Err() error
	Next() bool
	NextResultSet() bool
	Scan(dest ...any) error
}

type RowsEvents struct {
}

type rows struct {
	cols    []string
	coltpes []ColumnType
	dta     []interface{}
	IRows
	lsterr error
	first  bool
	last   bool
	evnts  *ReaderEvents
}

type dbirows struct {
	dbrows  *sql.Rows
	dtarefs []interface{}
}

func (dbirws *dbirows) Close() (err error) {
	if dbirws == nil {
		return
	}
	dbrows := dbirws.dbrows
	dbirws.dtarefs = nil
	dbirws.dbrows = nil
	if dbrows != nil {
		err = dbrows.Close()
	}
	return
}

func (dbirws *dbirows) ColumnTypes() (cltps []ColumnType, err error) {
	if dbirws == nil {
		return
	}
	if dbrows := dbirws.dbrows; dbrows != nil {
		dbcltps, dberr := dbrows.ColumnTypes()
		if err = dberr; err != nil {
			return
		}
		if cltpsl := len(dbcltps); cltpsl > 0 {
			cltps = make([]ColumnType, cltpsl)
			for cn := range cltpsl {
				cltps[cn] = dbcltps[cn]
			}
		}
	}
	return
}

func (dbirws *dbirows) Columns() (cls []string, err error) {
	if dbirws == nil {
		return
	}
	if dbrows := dbirws.dbrows; dbrows != nil {
		cls, err = dbirws.Columns()
	}
	return
}

func (dbirws *dbirows) Err() (err error) {
	if dbirws == nil {
		return
	}
	if dbrows := dbirws.dbrows; dbrows != nil {
		err = dbrows.Err()
	}
	return
}

func (dbirws *dbirows) Next() (nxt bool) {
	if dbirws == nil {
		return
	}
	if dbrows := dbirws.dbrows; dbrows != nil {
		nxt = dbrows.Next()
	}
	return
}

func (dbirws *dbirows) NextResultSet() (nxtrstst bool) {
	if dbirws == nil {
		return
	}
	if dbrows := dbirws.dbrows; dbrows != nil {
		nxtrstst = dbrows.NextResultSet()
	}
	return
}

func (dbirws *dbirows) Scan(dest ...any) (err error) {
	if dbirws == nil {
		return
	}
	if dbrows := dbirws.dbrows; dbrows != nil {
		if destl := len(dest); destl > 0 {
			destref := dbirws.dtarefs
			if len(destref) != destl {
				destref = make([]interface{}, destl)
				dbirws.dtarefs = destref
			}
			for n := range destl {
				destref[n] = &dest[n]
			}
			err = dbrows.Scan(destref...)
		}
	}
	return
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
func (r *rows) ColumnTypes() (cltps []ColumnType, err error) {
	if r == nil {
		return
	}
	if len(r.cols) == 0 && r.IRows != nil {
		r.Columns()
	}
	return r.coltpes, r.lsterr
}

// Data implements Rows.
func (r *rows) Data() []interface{} {
	if r == nil {
		return nil
	}
	return r.dta
}

// Close implements Rows.
func (r *rows) Close() (err error) {
	if r == nil {
		return
	}
	dta := r.dta
	r.dta = nil
	r.cols = nil
	irows := r.IRows
	r.IRows = nil
	r.lsterr = nil

	if irows != nil {
		irows.Close()
	}
	for _, d := range dta {
		if rd, rdk := d.(Reader); rdk {
			if rd != nil {
				rd.Close()
			}
		}
	}
	return
}

func (r *rows) First() (first bool) {
	if r == nil {
		return
	}
	return r.first
}

func (r *rows) Columns() ([]string, error) {
	if r == nil {
		return nil, nil
	}
	cols := r.cols
	if len(cols) > 0 {
		return cols, nil
	}
	if irows := r.IRows; irows != nil {
		cltpes, cltpeserr := irows.ColumnTypes()
		if cltpeserr != nil {
			r.lsterr = cltpeserr
			return nil, r.lsterr
		}
		if cltpsl := len(cltpes); cltpsl > 0 && cltpeserr == nil {
			cols = make([]string, cltpsl)
			for cn, cltp := range cltpes {
				cols[cn] = cltp.Name()
			}
			r.cols = cols
			r.coltpes = cltpes
			return cols, nil
		}
		if cols, r.lsterr = irows.Columns(); r.lsterr != nil {
			return nil, r.lsterr
		}
		r.cols = cols
		return cols, nil
	}
	return nil, nil
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
	if irows := r.IRows; irows != nil {
		if !r.first {
			if len(r.cols) == 0 {
				if r.Columns(); r.lsterr != nil {
					return
				}
			}
			if nxt, r.lsterr = irows.Next(), irows.Err(); r.lsterr == nil && nxt {
				r.first = true
				dta := r.dta
				if len(dta) < len(r.cols) {
					dta = make([]interface{}, len(r.cols))
					r.dta = dta
				}

				r.lsterr = irows.Scan(dta...)
				/*for n, d := range dta {
					if rw, rwk := d.(*sql.Rows); rwk {
						dta[n] = &reader{rws: &rows{irows: &dbirows{dbrows: rw}}}
					}
				}*/
				if nxt = r.lsterr == nil; nxt {
					r.last, r.lsterr = irows.Next(), irows.Err()
					nxt = r.lsterr == nil
				}
			}
			return
		}
		if nxt = r.last; nxt {
			dta := r.dta
			if cl, dtal := len(r.cols), len(dta); cl > 0 && cl == dtal {
				r.lsterr = irows.Scan(dta...)
				if nxt = r.lsterr == nil; nxt {
					r.last, r.lsterr = irows.Next(), irows.Err()
					nxt = r.lsterr == nil
				}
			}
			return
		}
		nxt = false
	}
	return
}
