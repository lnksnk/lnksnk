package dbms

import (
	"database/sql"

	"github.com/lnksnk/lnksnk/ioext"
)

type Drivers interface {
	ioext.IterateMap[string, Driver]
	Empty() bool
	Exist(string) bool
	Register(string, ...interface{}) Driver
	DefaultInvokable(func(string) (InvokeDB InvokeDBFunc, ParseSqlParam ParseSqlArgFunc))
}

type drivers struct {
	ioext.IterateMap[string, Driver]
	dfltinvkbl func(string) (InvokeDB InvokeDBFunc, ParseSqlParam ParseSqlArgFunc)
}

// Empty implements Drivers.
// Subtle: this method shadows the method (IterateMap).Empty of drivers.IterateMap.
func (dvrs *drivers) Empty() bool {
	if dvrs == nil {
		return true
	}
	if itr := dvrs.IterateMap; itr != nil {
		return itr.Empty()
	}
	return true
}

// Exist implements Drivers.
func (dvrs *drivers) Exist(name string) bool {
	if dvrs == nil {
		return true
	}
	if itr := dvrs.IterateMap; itr != nil {
		return itr.Contains(name)
	}
	return true
}

// DefaultInvokable implements Drivers.
func (dvrs *drivers) DefaultInvokable(dfltinvkbl func(string) (InvokeDB InvokeDBFunc, ParseSqlParam ParseSqlArgFunc)) {
	if dvrs == nil {
		return
	}
	if dfltinvkbl != nil {
		dvrs.dfltinvkbl = dfltinvkbl
	}
}

// Register implements Drivers.
func (dvrs *drivers) Register(alias string, a ...interface{}) (dvr Driver) {
	if dvrs == nil {
		return nil
	}
	if itr := dvrs.IterateMap; itr != nil {
		var dbinvk InvokeDBFunc
		var prssqlarg ParseSqlArgFunc
		if len(a) > 0 {
			for _, d := range a {
				if ainvkdb, ainvkdbk := d.(InvokeDBFunc); ainvkdbk {
					if dbinvk == nil {
						dbinvk = ainvkdb
					}
					continue
				}
				if ainvkdb, ainvkdbk := d.(func(string, ...interface{}) (*sql.DB, error)); ainvkdbk {
					if dbinvk == nil {
						dbinvk = ainvkdb
					}
					continue
				}
				if aprsarg, aprsargk := d.(ParseSqlArgFunc); aprsargk {
					if prssqlarg == nil {
						prssqlarg = aprsarg
					}
					continue
				}
				if aprsarg, aprsargk := d.(func(int) string); aprsargk {
					if prssqlarg == nil {
						prssqlarg = aprsarg
					}
					continue
				}
			}
		} else if dfltinvkbl := dvrs.dfltinvkbl; dfltinvkbl != nil {
			dbinvk, prssqlarg = dfltinvkbl(alias)
		}
		if dbinvk != nil {
			dvr = NewDriver(alias, dbinvk, prssqlarg, a...)
			itr.Set(alias, dvr)
		}
	}
	return
}

// Clear implements Drivers.
func (dvrs *drivers) Clear() {
	if dvrs == nil {
		return
	}
	if iter := dvrs.IterateMap; iter != nil {
		iter.Clear()
	}
}

// Close implements Drivers.
func (dvrs *drivers) Close() {
	if dvrs == nil {
		return
	}
	itr := dvrs.IterateMap
	dvrs.IterateMap = nil
	if itr != nil {
		itr.Close()
	}
}

// Contains implements Drivers.
func (dvrs *drivers) Contains(name string) bool {
	if dvrs == nil {
		return false
	}
	if itr := dvrs.IterateMap; itr != nil {
		return itr.Contains(name)
	}
	return false
}

// Events implements Drivers.
func (dvrs *drivers) Events() ioext.IterateMapEvents[string, Driver] {
	if dvrs == nil {
		return nil
	}
	if itr := dvrs.IterateMap; itr != nil {
		return itr.Events()
	}
	return nil
}

// Get implements Drivers.
func (dvrs *drivers) Get(name string) (value Driver, found bool) {
	if dvrs == nil {
		return
	}
	if itr := dvrs.IterateMap; itr != nil {
		return itr.Get(name)
	}
	return
}

// Iterate implements Drivers.
func (dvrs *drivers) Iterate() func(func(string, Driver) bool) {
	if dvrs == nil {
		return func(f func(string, Driver) bool) {
		}
	}
	if itr := dvrs.IterateMap; itr != nil {
		return itr.Iterate()
	}
	return func(f func(string, Driver) bool) {
	}
}

func NewDrivers() Drivers {
	drvs := &drivers{IterateMap: ioext.MapIterator[string, Driver]()}
	return drvs
}
