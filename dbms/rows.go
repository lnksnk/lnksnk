package dbms

import (
	"database/sql"
)

type Rows interface {
	IRows
	RowNR() int64
	First() bool
	Next() bool
	SelectNext(func(Rows) bool) bool
	Data() []interface{}
	Err() error
	Last() bool
	Events() *RowsEvents
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
	Error func(error)
}

func (r *RowsEvents) error(err error) {
	if r != nil && err != nil {
		if evterr := r.Error; evterr != nil {
			evterr(err)
		}
	}
}

type rows struct {
	cols    []string
	coltpes []ColumnType
	dta     []interface{}
	lstdta  []interface{}
	nxtdta  []interface{}
	swpdta  bool
	sctltd  func(Rows) bool
	IRows
	lsterr  error
	started bool
	first   bool
	last    bool
	rwnr    int64
	evnts   *RowsEvents
}

// RowNR implements Rows.
func (r *rows) RowNR() int64 {
	if r == nil {
		return -1
	}
	if r.swpdta {
		return r.rwnr + 1
	}
	return r.rwnr
}

// NextResultSet implements Rows.
// Subtle: this method shadows the method (IRows).NextResultSet of rows.IRows.
func (r *rows) NextResultSet() bool {
	panic("unimplemented")
}

// Scan implements Rows.
// Subtle: this method shadows the method (IRows).Scan of rows.IRows.
func (r *rows) Scan(dest ...any) error {
	panic("unimplemented")
}

func (r *rows) nextRec(irows IRows, dta, nxtdta, lstdta []interface{}) (nxt bool) {
	if r == nil {
		return
	}
	if r.last {
		return
	}
	evnts := r.evnts
	rdrc := false
	if r.started {
		if r.first {
			r.first = false
		}
		copy(lstdta, nxtdta)
		r.rwnr++
		if rdrc, r.lsterr = readRow(irows, evnts); r.lsterr != nil {
			r.Close()
			return
		}
		if !rdrc {
			r.last = true
			copy(dta, lstdta)
			return true
		}
		r.swpdta = true
	rescn2:
		if r.lsterr = scanRow(irows, nxtdta, evnts); r.lsterr != nil {
			r.Close()
			return
		}
		if !r.sctltd(r) {
			if rdrc, r.lsterr = readRow(irows, evnts); r.lsterr != nil {
				r.Close()
				return
			}
			if !rdrc {
				r.swpdta = false
				r.last = true
				copy(dta, lstdta)
				return true
			}
			goto rescn2
		}
		r.swpdta = false
		copy(dta, lstdta)
		return true
	}
	r.first = true
	r.started = true
	if rdrc, r.lsterr = readRow(irows, evnts); r.lsterr != nil {
		r.Close()
		return
	}
	if !rdrc {
		r.Close()
		return
	}
	r.swpdta = true
rescn:
	if r.lsterr = scanRow(irows, nxtdta, evnts); r.lsterr != nil {
		r.Close()
		return
	}
	if !r.sctltd(r) {
		if rdrc, r.lsterr = readRow(irows, evnts); r.lsterr != nil {
			r.Close()
			return
		}
		if !rdrc {
			r.Close()
			return
		}
		goto rescn
	}
	copy(lstdta, nxtdta)
	r.rwnr++
	if rdrc, r.lsterr = readRow(irows, evnts); r.lsterr != nil {
		r.Close()
		return
	}
	if !rdrc {
		r.swpdta = false
		r.last = true
		copy(dta, lstdta)
		return true
	}
rescnnxt:
	if r.lsterr = scanRow(irows, nxtdta, evnts); r.lsterr != nil {
		r.Close()
		return
	}
	if !r.sctltd(r) {
		if rdrc, r.lsterr = readRow(irows, evnts); r.lsterr != nil {
			r.Close()
			return
		}
		if !rdrc {
			r.swpdta = false
			r.last = true
			copy(dta, lstdta)
			return true
		}
		goto rescnnxt
	}
	r.swpdta = false
	copy(dta, lstdta)
	return true
}

func readRow(irows IRows, evnts *RowsEvents) (nxt bool, err error) {
	if nxt, err = irows.Next(), irows.Err(); err != nil && evnts != nil && evnts.Error != nil {
		evnts.error(err)
	}
	return
}

func scanRow(irows IRows, storedta []interface{}, evnts *RowsEvents) (err error) {
	if err = irows.Scan(storedta...); evnts != nil && evnts.Error != nil {
		evnts.Error(err)
	}
	return
}

// SelectNext implements Rows.
func (r *rows) SelectNext(slctd func(Rows) bool) (nxt bool) {
	if r == nil {
		return false
	}
	if len(r.cols) == 0 {
		if r.Columns(); r.lsterr != nil {
			return
		}
	}
	if !r.started {
		if r.sctltd == nil {
			if slctd == nil {
				r.sctltd = dummyslct
			} else {
				r.sctltd = slctd
			}
		}
	}
	if irows := r.IRows; irows != nil {
		nxt = r.nextRec(irows, r.dta, r.nxtdta, r.lstdta)
	}
	return
}
func dummyslct(r Rows) bool {
	return true
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
			if len(destref) != destl || (len(destref) == destl && destref[0] != &dest[0]) {
				if len(destref) != destl {
					destref = make([]interface{}, destl)
				}
				dbirws.dtarefs = destref
				for n := range destl {
					destref[n] = &dest[n]
				}
			}

			err = dbrows.Scan(destref...)
		}
	}
	return
}

// Events implements Rows.
func (r *rows) Events() *RowsEvents {
	if r == nil {

	}
	if evnts := r.evnts; evnts != nil {
		return evnts
	}
	r.evnts = &RowsEvents{}
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
	if r.swpdta {
		return r.nxtdta
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
			if evnts := r.evnts; evnts != nil {
				evnts.error(r.lsterr)
			}
			return nil, r.lsterr
		}
		if cltpsl := len(cltpes); cltpsl > 0 {
			cols = make([]string, cltpsl)
			for cn := range cltpsl {
				cols[cn] = cltpes[cn].Name()
			}
			r.cols = cols
			r.coltpes = cltpes
			dtal, lstdtal, nxtdtal := len(r.dta), len(r.lstdta), len(r.nxtdta)
			if dtal != cltpsl {
				r.dta = nil
				r.dta = make([]interface{}, cltpsl)
			}
			if lstdtal != cltpsl {
				r.lstdta = nil
				r.lstdta = make([]interface{}, cltpsl)
			}
			if nxtdtal != cltpsl {
				r.nxtdta = nil
				r.nxtdta = make([]interface{}, cltpsl)
			}
			return cols, nil
		}
		if cols, r.lsterr = irows.Columns(); r.lsterr != nil {
			if evnts := r.evnts; evnts != nil {
				evnts.error(r.lsterr)
			}
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
	return r.SelectNext(nil)
}
