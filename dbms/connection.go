package dbms

import (
	"context"
	"database/sql"

	"github.com/lnksnk/lnksnk/fs"
)

type Connection interface {
	Close()
	Query(string, ...interface{}) (Reader, error)
	QueryContext(context.Context, string, ...interface{}) (Reader, error)
	Execute(string, ...interface{}) (Result, error)
	ExecuteContext(context.Context, string, ...interface{}) (Result, error)
	Reload(string, ...Driver) error
	Driver() Driver
}

func (cn *connection) Reload(datasource string, drvr ...Driver) (err error) {
	if cn == nil {
		return
	}
	if len(drvr) > 0 && drvr[0] != nil {
		if cn.dvr != drvr[0] {
			cn.dvr = drvr[0]
			cn.datasource = datasource
			db := cn.db
			cn.db = nil
			if db != nil {
				go db.Close()
			}
			return
		}
	}
	if cn.datasource != datasource {
		cn.datasource = datasource
		db := cn.db
		cn.db = nil
		if db != nil {
			go db.Close()
		}
	}
	return
}

type connection struct {
	db *sql.DB
	//dblck      *sync.Mutex
	datasource string
	dvr        Driver
	fsys       fs.MultiFileSystem
}

// Execute implements Connection.
func (cn *connection) Execute(query string, a ...interface{}) (Result, error) {
	if cn == nil {
		return nil, nil
	}
	return cn.ExecuteContext(context.Background(), query, a...)
}

// ExecuteContext implements Connection.
func (cn *connection) ExecuteContext(ctx context.Context, query string, a ...interface{}) (result Result, err error) {
	db, dvr, dberr := nextdb(cn)
	if dberr != nil {
		return
	}
	if db != nil {
		s, serr := nextstatement(db, dvr, query, cn.fsys, a...)
		if err = serr; err != nil {
			return nil, err
		}
		if result, err = s.ExecuteContext(ctx, a...); err != nil {
			if cn.db == db {
				cn.db = nil
			}
			go db.Close()
		}
	}
	return
}

// Driver implements Connection.
func (cn *connection) Driver() Driver {
	if cn == nil {
		return nil
	}
	return cn.dvr
}

// Query implements Connection.
func (cn *connection) Query(query string, a ...interface{}) (rdr Reader, err error) {
	return cn.QueryContext(context.Background(), query, a...)
}

func nextdb(cn *connection) (db *sql.DB, dvr Driver, err error) {
	if cn == nil {
		return
	}
	if dvr = cn.dvr; dvr != nil {
		db = cn.db
		if db == nil {
			if dvr != nil {
				if db, err = dvr.Invoke(cn.datasource, cn.fsys); err != nil {
					return
				}
				cn.db = db
			}
		}
	}
	return
}

// QueryContext implements Connection.
func (cn *connection) QueryContext(ctx context.Context, query string, a ...interface{}) (rdr Reader, err error) {
	db, dvr, dberr := nextdb(cn)
	if dberr != nil {
		return
	}
	if db != nil {
		s, serr := nextstatement(db, dvr, query, cn.fsys, a...)
		if err = serr; err != nil {
			if cn.db == db {
				cn.db = nil
			}
			go db.Close()
			return nil, err
		}
		rdr, err = s.QueryContext(ctx, a...)
	}
	return
}

func (cn *connection) Close() {
	if cn == nil {

	}
	cn.dvr = nil
	db := cn.db
	cn.db = nil
	if db != nil {
		go db.Close()
	}
	cn.fsys = nil
}

func NewConnection(dvr Driver, datasource string, fsys ...fs.MultiFileSystem) Connection {
	return &connection{dvr: dvr, datasource: datasource, fsys: func() fs.MultiFileSystem {
		if len(fsys) > 0 && fsys[0] != nil {
			return fsys[0]
		}
		return nil
	}()}
}
