package dbms

import "database/sql"

type Driver interface {
	Invoke(string, ...interface{}) (*sql.DB, error)
	ParseSqlArg(int) string
	Dispose()
	Name() string
}

type driver struct {
	invokedb    func(string, ...interface{}) (*sql.DB, error)
	parseSqlArg func(int) string
	driver      string
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

func NewDriver(name string, invokedb func(string, ...interface{}) (*sql.DB, error), parseSqlArg func(int) string) Driver {
	return &driver{invokedb: invokedb, parseSqlArg: parseSqlArg, driver: name}
}
