package dbms

type Reader interface {
	Columns() []string
	ColumnTypes() []ColumnType
	Data() []interface{}
	Close()
	Next() bool
	First() bool
	Last() bool
	AttachHandler(DBMSHandler)
	Records() func(func(Record) bool)
	Events() interface{}
	Record() Record
}

type ReaderEvents struct {
	*RowsEvents
}

type reader struct {
	stmnt    Statement
	rws      Rows
	dbhndlr  DBMSHandler
	evnts    *ReaderEvents
	dsphndlr bool
	rc       *record
}

// Record implements Reader.
func (rdr *reader) Record() Record {
	if rdr == nil {
		return nil
	}
	return rdr.rc
}

// Events implements Reader.
func (rdr *reader) Events() interface{} {
	if rdr == nil {
		return nil
	}
	if evnts := rdr.evnts; evnts != nil {
		return evnts
	}
	if rws := rdr.rws; rws != nil {
		if rsevt, _ := rws.Events().(*RowsEvents); rsevt != nil {
			rdr.evnts = &ReaderEvents{RowsEvents: rsevt}
			return rdr.evnts
		}
	}
	return nil
}

// Records implements Reader.
func (rdr *reader) Records() func(func(Record) bool) {
	return func(nxtrc func(Record) bool) {
		if rdr == nil {
			return
		}
		if rdr.rws != nil && rdr.rws.Err() == nil {
			for rdr.Next() {
				/*rc.dta = rdr.Data()
				rc.first = rdr.First()
				rc.last = rdr.Last()
				rc.cnt++*/
				if !nxtrc(rdr.Record()) {
					return
				}
			}
		}
	}
}

// ColumnTypes implements Reader.
func (rdr *reader) ColumnTypes() []ColumnType {
	if rdr == nil {
		return nil
	}
	rws := rdr.rws
	if rws == nil {
		if stmt := rdr.stmnt; stmt != nil {
			if rws = stmt.Rows(); rws != nil {
				rdr.rws = rws
				coltpes, coltpserr := rws.ColumnTypes()
				if coltpserr == nil {
					return coltpes
				}
			}
		}
		return nil
	}
	coltpes, coltpserr := rws.ColumnTypes()
	if coltpserr == nil {
		return coltpes
	}
	return nil
}

// AttachHandler implements Reader.
func (rdr *reader) AttachHandler(dbhndlr DBMSHandler) {
	if rdr == nil {
		return
	}
	if rdr.dbhndlr != dbhndlr {
		rdr.dbhndlr = dbhndlr
	}
}

// First implements Reader.
func (rdr *reader) First() bool {
	if rdr == nil {
		return false
	}
	if rc := rdr.rc; rc != nil {
		return rc.first
	}
	return false
}

// Last implements Reader.
func (rdr *reader) Last() bool {
	if rdr == nil {
		return false
	}
	if rc := rdr.rc; rc != nil {
		return rc.last
	}
	return false
}

// Next implements Reader.
func (rdr *reader) Next() (nxt bool) {
	if rdr == nil {
		return false
	}
	rws := rdr.rws
	if rws == nil {
		if stmt := rdr.stmnt; stmt != nil {
			if rws = stmt.Rows(); rws != nil {
				rdr.rws = rws
			} else {
				rdr.Close()
				return false
			}
		} else {
			rdr.Close()
			return false
		}
	}
	if nxt = rws.Next(); nxt {
		rc := rdr.rc
	reinitrc:
		if rc == nil {
			cols, coltpes := rdr.Columns(), rdr.ColumnTypes()
			if nxt = rdr.rws.Err() == nil; !nxt {
				rdr.Close()
				return
			}
			rc = &record{rdr: rdr, cnt: 1, cols: cols, coltpes: coltpes, first: rws.First(), last: rws.Last(), dta: rws.Data()}
			rdr.rc = rc
			return
		}
		if rc.first = rws.First(); rc.first {
			rc.cols = nil
			rc.coltpes = nil
			rc.dta = nil
			rc = nil
			goto reinitrc
		}
		rc.last = rws.Last()
		rc.dta = rdr.Data()
		rc.cnt++
		return
	}
	rdr.Close()
	return
}

// Columns implements Reader.
func (rdr *reader) Columns() []string {
	if rdr == nil {
		return nil
	}

	rws := rdr.rws
	if rws == nil {
		if stmt := rdr.stmnt; stmt != nil {
			if rws = stmt.Rows(); rws != nil {
				rdr.rws = rws
				cols, colserr := rws.Columns()
				if colserr == nil {
					return cols
				}
			}
		}
		return nil
	}
	cols, colserr := rws.Columns()
	if colserr == nil {
		return cols
	}
	return nil
}

// Data implements Reader.
func (rdr *reader) Data() []interface{} {
	if rdr == nil {
		return nil
	}
	if rc := rdr.rc; rc != nil {
		return rc.dta
	}
	return nil
}

func (rdr *reader) Close() {
	if rdr == nil {
		return
	}
	stmnt := rdr.stmnt
	rc := rdr.rc
	rdr.rc = nil
	dbhndlr, dsphndlr := rdr.dbhndlr, rdr.dsphndlr
	rdr.dbhndlr = nil
	rdr.stmnt = nil
	rdr.rws = nil
	if stmnt != nil {
		stmnt.Close()
	}
	if dbhndlr != nil {
		if dsphndlr {
			dbhndlr.Close()
			return
		}
		dbhndlr.DettachReader(rdr)
	}
	if rc != nil {
		rc.cols = nil
		rc.coltpes = nil
		rc.dta = nil
	}
}
