package dbms

import (
	"context"

	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
)

type DBMS interface {
	Load(...interface{})
	Unload(...interface{})
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

// Unload implements DBMS.
func (d *dbms) Unload(a ...interface{}) {
	if d == nil {
		return
	}
	drvrs := d.drvrs
	cnss := d.cnctns
	if len(a) == 1 {
		if mpd, mpk := a[0].(map[string]interface{}); mpk && len(mpd) == 0 {
			a = nil
		}
	}
	if len(a) == 0 {
		if cnss != nil {
			var cnssevt = cnss.Events().(*ioext.MapIterateEvents[string, Connection])
			ctxcns, cnclcns := context.WithCancel(context.Background())
			cnssevt.EventDisposed = func(dspm map[string]Connection) {
				defer cnclcns()
				for _, cn := range dspm {
					cn.Close()
				}
			}
			cnss.Close()
			<-ctxcns.Done()
		}
		if drvrs != nil {
			var drvsevt = drvrs.Events().(*ioext.MapIterateEvents[string, Driver])
			ctxdrvs, cncldvrs := context.WithCancel(context.Background())
			drvsevt.EventDisposed = func(dspm map[string]Driver) {
				defer cncldvrs()
				for _, dvr := range dspm {
					dvr.Dispose()
				}
			}
			drvrs.Close()
			<-ctxdrvs.Done()
		}
		return
	}
	var dmsunldcnf map[string]interface{}
	ai := 0
	al := len(a)
	for ai < al {
		if mpd, mpk := a[ai].(map[string]interface{}); mpk {
			if len(mpd) > 0 {
				if dmsunldcnf == nil {
					dmsunldcnf = mpd
				} else {
					for k, v := range mpd {
						dmsunldcnf[k] = v
					}
				}
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
	}
	if len(a) > 0 {
		in, _ := ioext.NewBuffer(a...).Reader(true).Marshal()
		if mpd, mpk := in.(map[string]interface{}); mpk {
			if len(mpd) > 0 {
				if dmsunldcnf == nil {
					dmsunldcnf = mpd
				} else {
					for k, v := range mpd {
						dmsunldcnf[k] = v
					}
				}
			}
		}
	}
	var delcons []string
	var deldrvrs []string
	for k, v := range dmsunldcnf {
		if k == "connections" || k == "conns" {
			if arrs, arrsk := v.([]string); arrsk {
				if len(arrs) == 0 {
					continue
				}
				for _, as := range arrs {
					if as != "" {
						delcons = append(delcons, as)
					}
				}
				continue
			}
			if arri, arrik := v.([]interface{}); arrik {
				for _, ai := range arri {
					if as, _ := ai.(string); as != "" {
						delcons = append(delcons, as)
					}
				}
				continue
			}
			continue
		}
		if k == "drivers" {
			if arrs, arrsk := v.([]string); arrsk {
				if len(arrs) == 0 {
					continue
				}
				for _, as := range arrs {
					if as != "" {
						deldrvrs = append(deldrvrs, as)
					}
				}
				continue
			}
			if arri, arrik := v.([]interface{}); arrik {
				for _, ai := range arri {
					if as, _ := ai.(string); as != "" {
						deldrvrs = append(deldrvrs, as)
					}
				}
				continue
			}
			continue
		}
	}
	if len(delcons) > 0 {
		delevtns := cnss.Events().(*ioext.MapIterateEvents[string, Connection])
		delconsctx, delconscancel := context.WithCancel(context.Background())
		delevtns.EventDeleted = func(dlm map[string]Connection) {
			for _, dlcn := range dlm {
				dlcn.Close()
			}
			delconscancel()
		}
		cnss.Delete(delcons...)
		<-delconsctx.Done()
	}
	if len(deldrvrs) > 0 {
		delevtns := drvrs.Events().(*ioext.MapIterateEvents[string, Driver])
		deldvrctx, deldvrcancel := context.WithCancel(context.Background())
		delevtns.EventDeleted = func(dlm map[string]Driver) {
			for _, dldvr := range dlm {
				dldvr.Dispose()
			}
			deldvrcancel()
		}
		drvrs.Delete(deldrvrs...)
		<-deldvrctx.Done()
	}
}

// Load implements DBMS.
func (d *dbms) Load(a ...interface{}) {
	if d == nil {
		return
	}
	var config map[string]interface{}
	ai := 0
	if al := len(a); al > 0 {
		for ai < al {
			if confd, confdk := a[ai].(map[string]interface{}); confdk {
				if len(confd) > 0 {
					if config == nil {
						config = confd
					} else {
						for ck, cv := range confd {
							config[ck] = cv
						}
					}
				}
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			ai++
		}
		if al > 0 {
			in, _ := ioext.NewBuffer(a...).Reader(true).Marshal()
			if cnfd, cnfk := in.(map[string]interface{}); cnfk {
				if len(cnfd) > 0 {
					if config == nil {
						config = cnfd
					} else {
						for ck, cv := range cnfd {
							config[ck] = cv
						}
					}
				}
			}
		}
	}
	if len(config) == 0 {
		return
	}
	for k, v := range config {
		if k == "connections" || k == "conns" {
			if cncnfd, cncnfk := v.(map[string]interface{}); cncnfk {
				conns := d.Connections()
				for cnme, cv := range cncnfd {
					if cvcnf, _ := cv.(map[string]interface{}); len(cvcnf) > 0 {
						dtasrc, _ := cvcnf["datasource"].(string)
						if dtasrc == "" {
							if dtasrc, _ = cvcnf["datassrc"].(string); dtasrc == "" {
								if dtasrc, _ = cvcnf["connectionstring"].(string); dtasrc == "" {
									dtasrc, _ = cvcnf["cnstring"].(string)
								}
							}
						}
						dvr, _ := cvcnf["driver"].(string)
						if dtasrc != "" && dvr != "" {
							conns.Register(cnme, dvr, dtasrc)
						}
					}
				}
			}
			continue
		}
		if k == "drivers" {
			drvrs := d.Drivers()
			if drvscnfd, drvrscnfk := v.(map[string]interface{}); drvrscnfk {
				for dk, dv := range drvscnfd {
					if arrv, arrk := dv.([]interface{}); arrk && len(arrv) > 0 {
						drvrs.Register(dk, arrv...)
					}
				}
				continue
			}
			if drvsarr, drvsarrk := v.([]interface{}); drvsarrk {
				if len(drvsarr) > 0 {
					drvrnms := []string{}
					for _, dnme := range drvsarr {
						if ds, _ := dnme.(string); ds != "" {
							drvrnms = append(drvrnms, ds)
						}
					}
					if len(drvrnms) > 0 {
						for _, dnm := range drvrnms {
							drvrs.Register(dnm)
						}
					}
				}
				continue
			}
			continue
		}
	}
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
