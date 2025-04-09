package dbms

import (
	"database/sql"
	"fmt"

	"github.com/lnksnk/lnksnk/fs"
)

type Driver interface {
	Invoke(string, ...interface{}) (*sql.DB, error)
	ParseSqlArg(int) string
	Dispose()
	Name() string
	DataSource() string
	FSys() fs.MultiFileSystem
}

type driver struct {
	invokedb    func(string, ...interface{}) (*sql.DB, error)
	parseSqlArg func(int) string
	driver      string
	fsys        fs.MultiFileSystem
	datasource  string
}

func (dvr *driver) FSys() fs.MultiFileSystem {
	if dvr == nil {
		return nil
	}
	return dvr.fsys
}

func (dvr *driver) DataSource() string {
	if dvr == nil {
		return ""
	}
	return dvr.datasource
}

func (dvr *driver) Name() string {
	if dvr == nil {
		return ""
	}
	return dvr.driver
}

func (dvr *driver) Invoke(datasource string, a ...interface{}) (db *sql.DB, err error) {
	if dvr != nil {
		if invokedb := dvr.invokedb; invokedb != nil {
			if datasource == "" {
				datasource = dvr.datasource
			}
			if datasource == "" {
				return nil, fmt.Errorf("no datasource for driver %s", dvr.driver)
			}
			db, err = invokedb(datasource, a...)
			if err != nil {
				return
			}
			return db, nil
		}
	}
	return
}

func (dvr *driver) Dispose() {

}

func (dvr *driver) ParseSqlArg(totalargs int) string {
	if dvr == nil {
		return ""
	}
	if parseSqlArg := dvr.parseSqlArg; parseSqlArg != nil {
		return parseSqlArg(totalargs)
	}
	return ""
}

func NewDriver(name string, invokedb func(string, ...interface{}) (*sql.DB, error), parseSqlArg func(int) string, a ...interface{}) Driver {
	var fsys fs.MultiFileSystem
	for _, d := range a {
		if fsysd, fsysk := d.(fs.MultiFileSystem); fsysk {
			if fsys == nil && fsysd != nil {
				fsys = fsysd
			}
			continue
		}
	}
	return &driver{invokedb: invokedb, parseSqlArg: parseSqlArg, driver: name, fsys: fsys}
}
