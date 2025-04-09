package dbdvl

import (
	"database/sql"
	"database/sql/driver"

	"github.com/lnksnk/lnksnk/ioext"
)

type Driver interface {
	driver.Driver
	ioext.IterateMap[string, driver.Conn]
}

type dvlDriver struct {
	ioext.IterateMap[string, driver.Conn]
}

// Clear implements Driver.
// Subtle: this method shadows the method (IterateMap).Clear of dvlDriver.IterateMap.
func (d *dvlDriver) Clear() {
	panic("unimplemented")
}

// Close implements Driver.
// Subtle: this method shadows the method (IterateMap).Close of dvlDriver.IterateMap.
func (d *dvlDriver) Close() {
	panic("unimplemented")
}

// Contains implements Driver.
// Subtle: this method shadows the method (IterateMap).Contains of dvlDriver.IterateMap.
func (d *dvlDriver) Contains(name string) bool {
	panic("unimplemented")
}

// Delete implements Driver.
// Subtle: this method shadows the method (IterateMap).Delete of dvlDriver.IterateMap.
func (d *dvlDriver) Delete(name ...string) {
	panic("unimplemented")
}

// Events implements Driver.
// Subtle: this method shadows the method (IterateMap).Events of dvlDriver.IterateMap.
func (d *dvlDriver) Events() ioext.IterateMapEvents[string, driver.Conn] {
	if d == nil {
		return nil
	}
	if itr := d.IterateMap; itr != nil {
		return itr.Events()
	}
	return nil
}

// Get implements Driver.
// Subtle: this method shadows the method (IterateMap).Get of dvlDriver.IterateMap.
func (d *dvlDriver) Get(name string) (value driver.Conn, found bool) {
	panic("unimplemented")
}

// Iterate implements Driver.
// Subtle: this method shadows the method (IterateMap).Iterate of dvlDriver.IterateMap.
func (d *dvlDriver) Iterate() func(func(string, driver.Conn) bool) {
	panic("unimplemented")
}

// Set implements Driver.
// Subtle: this method shadows the method (IterateMap).Set of dvlDriver.IterateMap.
func (d *dvlDriver) Set(name string, value driver.Conn) {
	panic("unimplemented")
}

func NewDriver() Driver {
	return &dvlDriver{IterateMap: ioext.MapIterator[string, driver.Conn]()}
}

// Open implements driver.Driver.
func (d *dvlDriver) Open(name string) (conn driver.Conn, err error) {
	if d == nil {
		return
	}
	if itr := d.IterateMap; itr != nil {
		if cnfnd, cnfndk := itr.Get(name); cnfndk {
			return cnfnd, err
		}
		conn = &dvlConn{lkupPath: name, dvr: d}
		itr.Set(name, conn)
	}
	return
}

func RegisterDriver(regname string) {
	sql.Register(regname, NewDriver())
}
