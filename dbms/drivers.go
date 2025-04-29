package dbms

import (
	"github.com/lnksnk/lnksnk/ioext"
)

type Drivers interface {
	ioext.IterateMap[string, Driver]
	ioext.IterateMapEvents[string, Driver]
	Register(string, ...interface{}) Driver
	DefaultInvokable(func(string) (InvokeDB InvokeDBFunc, ParseSqlParam ParseSqlArgFunc))
}

type drivers struct {
	ioext.IterateMap[string, Driver]
	dfltinvkbl func(string) (InvokeDB InvokeDBFunc, ParseSqlParam ParseSqlArgFunc)
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
	if dfltinvkbl := dvrs.dfltinvkbl; dfltinvkbl != nil {
		dbinvk, prssqlarg := dfltinvkbl(alias)
		if dbinvk != nil {
			if itr := dvrs.IterateMap; itr != nil {
				dvr = NewDriver(alias, dbinvk, prssqlarg, a...)
				itr.Set(alias, dvr)
			}
		}
	}
	return
}

// Add implements Drivers.
func (dvrs *drivers) Add(string, Driver) {

}

// Changed implements Drivers.
func (dvrs *drivers) Changed(string, Driver, Driver) {

}

// Deleted implements Drivers.
func (dvrs *drivers) Deleted(map[string]Driver) {

}

// Disposed implements Drivers.
func (dvrs *drivers) Disposed(map[string]Driver) {

}

// Clear implements Drivers.
func (dvrs *drivers) Clear() {

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

// Delete implements Drivers.
func (dvrs *drivers) Delete(name ...string) {
	if dvrs == nil {
		return
	}
	if itr := dvrs.IterateMap; itr != nil {

	}
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

// Set implements Drivers.
func (dvrs *drivers) Set(name string, value Driver) {
	/*if dvrs == nil {
		return
	}
	if itr := dvrs.IterateMap; itr != nil {
		itr.Set(name, value)
	}*/
}

func NewDrivers() Drivers {
	drvs := &drivers{IterateMap: ioext.MapIterator[string, Driver]()}
	if itrevnts, _ := drvs.Events().(*ioext.MapIterateEvents[string, Driver]); itrevnts != nil {
		itrevnts.EventChanged = drvs.Changed
		itrevnts.EventDeleted = drvs.Deleted
		itrevnts.EventDisposed = drvs.Disposed
		itrevnts.EventAdd = drvs.Add
	}
	return drvs
}
