package dbms

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/parameters"
)

type DBMSHandler interface {
	DBMS() DBMS
	Close() error
	Query(string, string, ...interface{}) (Reader, error)
	QueryContext(context.Context, string, string, ...interface{}) (Reader, error)
	Execute(string, string, ...interface{}) (Result, error)
	ExecuteContext(context.Context, string, string, ...interface{}) (Result, error)
	Params() parameters.Parameters
	AttachReader(Reader)
	DettachReader(Reader)
	Events() *DBMSHandlerEvents
}

type DBMSHandlerEvents struct {
	AttachedReader func(Reader)
	DetachedReader func(Reader)
	QueryError     func(error)
	ExecuteError   func(error)
	Error          func(error)
	Disposed       func()
}

func (dbmshnldevnts *DBMSHandlerEvents) dispose() {
	if dbmshnldevnts == nil {
		return
	}
	if disposed := dbmshnldevnts.Disposed; disposed != nil {
		disposed()
	}
	dbmshnldevnts.AttachedReader = nil
	dbmshnldevnts.DetachedReader = nil
	dbmshnldevnts.Disposed = nil
	dbmshnldevnts.Error = nil
	dbmshnldevnts.ExecuteError = nil
	dbmshnldevnts.QueryError = nil
}

func (dbmshnldevnts *DBMSHandlerEvents) error(err error) {
	if dbmshnldevnts == nil {
		return
	}
	if error := dbmshnldevnts.Error; error != nil {
		error(err)
	}
}

func (dbmshnldevnts *DBMSHandlerEvents) attachedRdr(rdr Reader) {
	if dbmshnldevnts == nil {
		return
	}
	if attachedRdr := dbmshnldevnts.AttachedReader; attachedRdr != nil {
		attachedRdr(rdr)
	}
}

func (dbmshnldevnts *DBMSHandlerEvents) detachedRdr(rdr Reader) {
	if dbmshnldevnts == nil {
		return
	}
	if detachedRdr := dbmshnldevnts.DetachedReader; detachedRdr != nil {
		detachedRdr(rdr)
	}
}

func (dbmshnldevnts *DBMSHandlerEvents) queryErr(err error) {
	if dbmshnldevnts == nil {
		return
	}
	if queryErr := dbmshnldevnts.QueryError; queryErr != nil {
		queryErr(err)
	}
}

func (dbmshnldevnts *DBMSHandlerEvents) executeErr(err error) {
	if dbmshnldevnts == nil {
		return
	}
	if executeErr := dbmshnldevnts.ExecuteError; executeErr != nil {
		executeErr(err)
	}
}

type dbmshandler struct {
	dbms        DBMS
	rdrs        *sync.Map
	params      parameters.Parameters
	fiqryreader func(fs.MultiFileSystem, fs.FileInfo, io.Writer)
	evnts       *DBMSHandlerEvents
}

// Events implements DBMSHandler.
func (dh *dbmshandler) Events() *DBMSHandlerEvents {
	if dh == nil {
		return nil
	}
	return dh.evnts
}

// AttachReader implements DBMSHandler.
func (dh *dbmshandler) AttachReader(rdr Reader) {
	if dh == nil {
		return
	}
	if rdrs := dh.rdrs; rdrs != nil {
		rdrs.Store(rdr, rdr)
		if evnts := dh.evnts; evnts != nil {
			evnts.attachedRdr(rdr)
		}
	}
}

// DettachReader implements DBMSHandler.
func (dh *dbmshandler) DettachReader(rdr Reader) {
	if dh == nil {
		return
	}
	if rdrs := dh.rdrs; rdrs != nil {
		if rdrs.CompareAndDelete(rdr, rdr) {
			if evnts := dh.evnts; evnts != nil {
				evnts.detachedRdr(rdr)
			}
		}
	}
}

// Params implements DBMSHandler.
func (dh *dbmshandler) Params() parameters.Parameters {
	if dh == nil {
		return nil
	}
	return dh.params
}

// Execute implements DBMSHandler.
func (dh *dbmshandler) Execute(alias string, query string, a ...interface{}) (result Result, err error) {
	if dh == nil {
		return
	}
	return dh.ExecuteContext(context.Background(), alias, query, a...)
}

// ExecuteContext implements DBMSHandler.
func (dh *dbmshandler) ExecuteContext(ctx context.Context, alias string, query string, a ...interface{}) (result Result, err error) {
	if dh == nil {
		return nil, fmt.Errorf("%s", "Invalid Execution")
	}
	if dbms := dh.dbms; dbms != nil {
		if cn, ck := dbms.Connections().Get(alias); ck {
			if result, err = cn.ExecuteContext(ctx, query, append(a, dh.params, dh.fiqryreader)...); err != nil {
				if evnts := dh.evnts; evnts != nil {
					evnts.executeErr(err)
				}
			}
			return
		}
	}
	return
}

// Query implements DBMSHandler.
func (dh *dbmshandler) Query(alias string, stmnt string, a ...interface{}) (Reader, error) {
	if dh == nil {
		return nil, fmt.Errorf("%s", "Empty Reader")
	}
	return dh.QueryContext(context.Background(), alias, stmnt, a...)
}

// QueryContext implements DBMSHandler.
func (dh *dbmshandler) QueryContext(ctx context.Context, alias string, stmnt string, a ...interface{}) (rdr Reader, err error) {
	if dh == nil {
		return nil, fmt.Errorf("%s", "Empty Reader")
	}
	if dbms, rdrs := dh.dbms, dh.rdrs; dbms != nil {
		if cn, ck := dbms.Connections().Get(alias); ck {
			if rdr, err = cn.QueryContext(ctx, stmnt, append(a, dh.params, dh.fiqryreader)...); err == nil {
				if rdf, rdk := rdr.(*reader); rdk && rdrs != nil {
					rdrs.Store(rdf, rdf)
					if evnts := dh.evnts; evnts != nil {
						evnts.attachedRdr(rdr)
					}
				}
				return
			}
			if evnts := dh.evnts; evnts != nil {
				evnts.queryErr(err)
			}
			return
		}
	}
	return
}

// Close implements DBMSHandler.
func (dh *dbmshandler) Close() (err error) {
	if dh == nil {
		return
	}
	evnts := dh.evnts
	dh.evnts = nil
	rdrs := dh.rdrs
	dh.rdrs = nil
	if rdrs != nil {
		var rdrsfnd []Reader
		rdrs.Range(func(key, value any) bool {
			if rdr, _ := value.(Reader); rdr != nil {
				rdrsfnd = append(rdrsfnd, rdr)
			}
			return true
		})
		rdrs.Clear()
		for _, rdr := range rdrsfnd {
			rdr.Close()
		}
	}
	if evnts != nil {
		evnts.dispose()
	}
	return
}

func (dh *dbmshandler) DBMS() DBMS {
	if dh == nil {
		return nil
	}
	return dh.dbms
}

func NewDBMSHandler(dbms DBMS, a ...interface{}) DBMSHandler {
	if dbms == nil {
		return nil
	}
	var params parameters.Parameters
	var fiqryreader func(fs.MultiFileSystem, fs.FileInfo, io.Writer)
	for _, d := range a {
		if paramsd, prmsk := d.(parameters.Parameters); prmsk {
			if params == nil {
				params = paramsd
			}
			continue
		}
		if fiqryoutd, fioutk := d.(func(fs.MultiFileSystem, fs.FileInfo, io.Writer)); fioutk {
			if fiqryreader == nil && fiqryoutd != nil {
				fiqryreader = fiqryoutd
			}
			continue
		}
	}
	return &dbmshandler{dbms: dbms, rdrs: &sync.Map{}, params: params, fiqryreader: fiqryreader}
}
