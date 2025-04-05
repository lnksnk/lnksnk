package dbms

import "github.com/lnksnk/lnksnk/fs"

type DBMS interface {
	Drivers() Drivers
	Handler(...interface{}) DBMSHandler
	Connections() Connections
}

type dbms struct {
	drvrs  Drivers
	cnctns Connections
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
