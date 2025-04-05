package dbms

import (
	"fmt"

	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
)

type Connections interface {
	ioext.IterateMap[string, Connection]
	ioext.IterateMapEvents[string, Connection]
	Register(string, string, string, ...interface{}) (bool, error)
}

type connections struct {
	drvrs Drivers
	ioext.IterateMap[string, Connection]
	fsys fs.MultiFileSystem
}

// Register implements Connections.
func (c *connections) Register(alias string, driver string, datasource string, a ...interface{}) (done bool, err error) {
	if c == nil {
		return
	}
	if itr := c.IterateMap; itr != nil {
		if cn, cnk := itr.Get(alias); cnk {
			cndvr := cn.Driver()
			if cndvr != nil {
				if cndvr.Name() != alias {
					cndvr, _ = c.drvrs.Get(alias)
				}
			}
			if cndvr == nil {
				c.Delete(alias)
				if cndvr, _ = c.drvrs.Get(alias); cndvr != nil {
					cn = NewConnection(cndvr, datasource, c.fsys)
					itr.Set(alias, cn)
					return true, nil
				}
				return false, fmt.Errorf("Unregistered driver %s", driver)
			}
			if err = cn.Reload(datasource, cndvr); err != nil {
				return
			}
			return true, nil
		}
		cndvr, _ := c.drvrs.Get(driver)
		if cndvr == nil {
			return false, fmt.Errorf("Unregistered driver %s", driver)
		}
		itr.Set(alias, NewConnection(cndvr, datasource, c.fsys))
		return true, nil
	}
	return false, fmt.Errorf("Unbale to register connection %s", alias)
}

// Changed implements Connections.
func (c *connections) Changed(alias string, prvcn Connection, cn Connection) {

}

// Clear implements Connections.
// Subtle: this method shadows the method (IterateMap).Clear of connections.IterateMap.
func (c *connections) Clear() {
	if c == nil {
		return
	}
	if itr := c.IterateMap; itr != nil {
		itr.Clear()
	}
}

// Close implements Connections.
// Subtle: this method shadows the method (IterateMap).Close of connections.IterateMap.
func (c *connections) Close() {
	if c == nil {
		return
	}
	itr := c.IterateMap
	c.IterateMap = nil
	if itr != nil {
		itr.Close()
	}
	c.drvrs = nil
}

// Contains implements Connections.
// Subtle: this method shadows the method (IterateMap).Contains of connections.IterateMap.
func (c *connections) Contains(name string) bool {
	if c == nil {
		return false
	}
	if itr := c.IterateMap; itr != nil {
		return itr.Contains(name)
	}
	return false
}

// Delete implements Connections.
// Subtle: this method shadows the method (IterateMap).Delete of connections.IterateMap.
func (c *connections) Delete(name ...string) {
	if c == nil {
		return
	}
	if itr := c.IterateMap; itr != nil {
		itr.Delete(name...)
	}
}

// Deleted implements Connections.
func (c *connections) Deleted(cnsdltd map[string]Connection) {
	for alias, cn := range cnsdltd {
		if alias != "" && cn != nil {
			cn.Close()
		}
	}
}

// Disposed implements Connections.
func (c *connections) Disposed(cnsdspsd map[string]Connection) {
	for alias, cn := range cnsdspsd {
		if alias != "" && cn != nil {
			cn.Close()
		}
	}
}

// Events implements Connections.
// Subtle: this method shadows the method (IterateMap).Events of connections.IterateMap.
func (c *connections) Events() ioext.IterateMapEvents[string, Connection] {
	if c == nil {
		return nil
	}
	if itr := c.IterateMap; itr != nil {
		return itr.Events()
	}
	return nil
}

// Get implements Connections.
// Subtle: this method shadows the method (IterateMap).Get of connections.IterateMap.
func (c *connections) Get(name string) (value Connection, found bool) {
	if c == nil {
		return
	}
	if itr := c.IterateMap; itr != nil {
		return itr.Get(name)
	}
	return
}

// Iterate implements Connections.
// Subtle: this method shadows the method (IterateMap).Iterate of connections.IterateMap.
func (c *connections) Iterate() func(func(string, Connection) bool) {
	if c == nil {
		return func(f func(string, Connection) bool) {}
	}
	if itr := c.IterateMap; itr != nil {
		return itr.Iterate()
	}
	return func(f func(string, Connection) bool) {}
}

// Set implements Connections.
// Subtle: this method shadows the method (IterateMap).Set of connections.IterateMap.
func (c *connections) Set(name string, value Connection) {

}

func NewConnections(drvrs Drivers, fsys ...fs.MultiFileSystem) Connections {
	cnctns := &connections{IterateMap: ioext.MapIterator[string, Connection](), drvrs: drvrs, fsys: func() fs.MultiFileSystem {
		if len(fsys) > 0 && fsys[0] != nil {
			return fsys[0]
		}
		return nil
	}()}
	if itrevnts, _ := cnctns.Events().(*ioext.MapIterateEvents[string, Connection]); itrevnts != nil {
		itrevnts.EventChanged = cnctns.Changed
		itrevnts.EventDeleted = cnctns.Deleted
		itrevnts.EventDisposed = cnctns.Disposed
	}
	return cnctns
}
