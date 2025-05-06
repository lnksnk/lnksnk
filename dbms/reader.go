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
	Events() *ReaderEvents
	Record() Record
}

type ReaderEvents struct {
	*RowsEvents
	ReaderError func(error)
	Select      func(Record) bool
	Disposed    func()
}

func (rdrevnts *ReaderEvents) calselect(rc Record) bool {
	if rdrevnts == nil {
		return true
	}
	if slctevt := rdrevnts.Select; slctevt != nil {
		return slctevt(rc)
	}
	return true
}

func (rdrevnts *ReaderEvents) rowsErr(rwserr error) {
	if rdrevnts == nil {
		return
	}
	if rdrerrevt := rdrevnts.ReaderError; rdrerrevt != nil {
		rdrerrevt(rwserr)
	}
}

func (rdrevnts *ReaderEvents) disposed() {
	if rdrevnts == nil {
		return
	}
	if rdrdspseevt := rdrevnts.Disposed; rdrdspseevt != nil {
		rdrdspseevt()
	}
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
func (rdr *reader) Events() *ReaderEvents {
	if rdr == nil {
		return nil
	}
	if evnts := rdr.evnts; evnts != nil {
		return evnts
	}
	if rws := rdr.rws; rws != nil {
		if rsevt := rws.Events(); rsevt != nil {
			rdr.evnts = &ReaderEvents{RowsEvents: rsevt}
			rsevt.Error = rdr.evnts.rowsErr
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

func (rdr *reader) prepRecord(rws Rows) (prepped bool) {
	if rdr == nil || rws == nil {
		return
	}
	rc := rdr.rc
	if rc == nil {
		cols, coltpes := rdr.Columns(), rdr.ColumnTypes()
		if prepped = rdr.rws.Err() == nil; !prepped {
			rdr.Close()
			return
		}
		rc = &record{rdr: rdr, rwnr: rws.RowNR(), cols: cols, coltpes: coltpes, first: rws.First(), last: rws.Last(), dta: rws.Data()}
		rdr.rc = rc
		return
	}
	rc.first = rws.First()
	rc.last = rws.Last()
	rc.dta = rws.Data()
	rc.rwnr = rws.RowNR()
	return true
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
	evnts := rdr.evnts
	if nxt = ((evnts == nil || evnts.Select == nil) && rws.Next()) || (evnts != nil && evnts.Select != nil && rws.SelectNext(func(r Rows) bool {
		return rdr.prepRecord(r) && evnts.calselect(rdr.rc)
	})); nxt {
		if nxt = rdr.prepRecord(rws) && rws.Err() == nil; nxt {
			return
		}
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
	evnts := rdr.evnts
	rdr.evnts = nil
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
	if evnts != nil {
		evnts.disposed()
	}
}
