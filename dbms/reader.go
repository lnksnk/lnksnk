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
}

type ReaderEvents struct {
	*RowsEvents
}

type reader struct {
	stmnt   Statement
	rws     Rows
	dbhndlr DBMSHandler
	evnts   *ReaderEvents
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
		if coltpes, cols := rdr.ColumnTypes(), rdr.Columns(); rdr.rws != nil && rdr.rws.Err() == nil {
			var rc *record
			for rdr.Next() {
				rc = &record{rdr: rdr, cnt: len(cols), cols: cols, coltpes: coltpes, dta: rdr.Data(), first: rdr.First(), last: rdr.Last()}
				if !nxtrc(rc) {
					break
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
	if rws := rdr.rws; rws != nil {
		return rws.First()
	}
	return false
}

// Last implements Reader.
func (rdr *reader) Last() bool {
	if rdr == nil {
		return false
	}
	if rws := rdr.rws; rws != nil {
		return rws.Last()
	}
	return false
}

// Next implements Reader.
func (rdr *reader) Next() bool {
	if rdr == nil {
		return false
	}
	rws := rdr.rws
	if rws == nil {
		if stmt := rdr.stmnt; stmt != nil {
			if rws = stmt.Rows(); rws != nil {
				rdr.rws = rws
				if rws.Next() {
					return true
				}
			}
			return false
		}
		return false
	}
	return rws.Next()
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
	if rws := rdr.rws; rws != nil {
		return rws.Data()
	}
	return nil
}

func (rdr *reader) Close() {
	if rdr == nil {
		return
	}
	stmnt := rdr.stmnt
	dbhndlr := rdr.dbhndlr
	rdr.dbhndlr = nil
	rdr.stmnt = nil
	rdr.rws = nil
	if stmnt != nil {
		stmnt.Close()
	}
	if dbhndlr != nil {
		dbhndlr.DettachReader(rdr)
	}
}
