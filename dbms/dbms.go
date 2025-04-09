package dbms

import (
	"context"

	"github.com/lnksnk/lnksnk/fs"
)

type DBMS interface {
	Drivers() Drivers
	Handler(...interface{}) DBMSHandler
	Connections() Connections
	Query(string, string, ...interface{}) (Reader, error)
	QueryContext(context.Context, string, string, ...interface{}) (Reader, error)
	Execute(string, string, ...interface{}) (Result, error)
	ExecuteContext(context.Context, string, string, ...interface{}) (Result, error)
}

type dbms struct {
	drvrs  Drivers
	cnctns Connections
}

// Execute implements DBMS.
func (d *dbms) Execute(alias string, query string, a ...interface{}) (Result, error) {
	if d == nil {
		return nil, nil
	}
	return d.ExecuteContext(context.Background(), alias, query, a...)
}

// ExecuteContext implements DBMS.
func (d *dbms) ExecuteContext(ctx context.Context, alias string, query string, a ...interface{}) (result Result, err error) {
	if d == nil {
		return
	}
	if dh := d.Handler(a...); dh != nil {
		defer dh.Close()
		result, err = dh.ExecuteContext(ctx, alias, query, a...)
	}
	return
}

// Query implements DBMS.
func (d *dbms) Query(alias string, query string, a ...interface{}) (Reader, error) {
	if d == nil {
		return nil, nil
	}
	return d.QueryContext(context.Background(), alias, query, a...)
}

// QueryContext implements DBMS.
func (d *dbms) QueryContext(ctx context.Context, alias string, query string, a ...interface{}) (Reader, error) {
	if d == nil {
		return nil, nil
	}
	if dh := d.Handler(a...); dh != nil {
		rdr, rdrerr := dh.QueryContext(ctx, alias, query, a...)
		if rdrerr != nil {
			return nil, rdrerr
		}
		if rd, _ := rdr.(*reader); rd != nil {
			rd.dsphndlr = true
		}
		return rdr, nil
	}
	return nil, nil
}

// Connections implements DBMS.
func (d *dbms) Connections() Connections {
	if d == nil {
		return nil
	}
	return d.cnctns
}

// Drivers implements DBMS.
func (d *dbms) Drivers() Drivers {
	if d == nil {
		return nil
	}
	return d.drvrs
}

// Handler implements DBMS.
func (d *dbms) Handler(a ...interface{}) DBMSHandler {
	if d == nil {
		return nil
	}

	return NewDBMSHandler(d, a...)
}

func NewDBMS(fsys ...fs.MultiFileSystem) DBMS {
	drvrs := NewDrivers()
	cnctns := NewConnections(drvrs, fsys...)
	return &dbms{drvrs: drvrs, cnctns: cnctns}
}

var glbldbms DBMS

func GLOBALDBMS() DBMS {
	return glbldbms
}

func init() {
	glbldbms = NewDBMS()
}
