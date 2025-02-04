package database

import (
	"context"
	"database/sql"

	"github.com/lnksnk/lnksnk/iorw/active"
	//"lnksnk/logging"
)

type Executor struct {
	prpOnly        bool
	stmnt          *Statement
	sqlresult      sql.Result
	eventinit      ExecutorInitFunc
	oninit         interface{}
	eventerror     ErrorFunc
	onerror        interface{}
	eventexec      ExecFunc
	onexec         interface{}
	eventexecerror ExecErrorFunc
	onexecerror    interface{}
	eventfinalize  ExecutorFinalizeFunc
	onfinalize     interface{}
	runtime        active.Runtime
	EventClose     func(*Executor)
	//LOG             logging.Logger
}

type ExecutorInitFunc func(*Executor) error
type ExecFunc func(*Executor, int64, int64) error
type ExecErrorFunc func(error, *Executor) (bool, error)
type ExecutorFinalizeFunc func(*Executor) error

func NewExecutor(stmnt *Statement, prepOnly bool, oninit interface{}, onexec interface{}, onexecerror interface{}, onerror interface{}, onfinalize interface{}, runtime active.Runtime /*, logger logging.Logger*/) (exectr *Executor) {
	exectr = &Executor{prpOnly: prepOnly, stmnt: stmnt, oninit: oninit, onerror: onerror, onexec: onexec, onexecerror: onexecerror, onfinalize: onfinalize /*, LOG: logger*/}
	if onerror == nil {
		exectr.eventerror = func(err error) {

		}
	}
	if onerror != nil {
		if donerror, _ := onerror.(ErrorFunc); donerror != nil {
			exectr.eventerror = donerror
		}
		if donerror, _ := onerror.(func(error)); donerror != nil {
			exectr.eventerror = donerror
		}
		if runtime != nil {
			exectr.eventerror = func(err error) {
				exectr.runtime.InvokeFunction(exectr.onerror, err)
			}
		}
	}
	if onfinalize == nil {
		exectr.eventfinalize = func(*Executor) error { return nil }
	} else {
		if donfinalize, _ := onfinalize.(ExecutorFinalizeFunc); donfinalize != nil {
			exectr.eventfinalize = donfinalize
		} else if donfinalize, _ := onfinalize.(func(*Executor) error); donfinalize != nil {
			exectr.eventfinalize = donfinalize
		} else if runtime != nil {
			exectr.eventfinalize = func(exctr *Executor) (err error) {
				exectr.runtime.InvokeFunction(exectr.onfinalize, exctr)
				return
			}
		}
	}
	if oninit == nil {
		exectr.eventinit = func(*Executor) error { return nil }
	}
	if oninit != nil {
		if doninit, _ := oninit.(ExecutorInitFunc); doninit != nil {
			exectr.eventinit = doninit
		}
		if doninit, _ := oninit.(func(*Executor) error); doninit != nil {
			exectr.eventinit = doninit
		}
		if exectr.eventinit == nil && runtime != nil {
			exectr.eventinit = func(exctr *Executor) (err error) {
				exectr.runtime.InvokeFunction(exectr.oninit, exctr)
				return
			}
		}
	}
	if onexec == nil {
		exectr.eventexec = func(*Executor, int64, int64) error { return nil }
	}
	if onexec != nil {
		if donexec, _ := onexec.(ExecFunc); donexec != nil {
			exectr.eventexec = donexec
		}
		if donexec, _ := onexec.(func(*Executor, int64, int64) error); donexec != nil {
			exectr.eventexec = donexec
		}
		if exectr.eventexec != nil && runtime != nil {
			exectr.eventexec = func(exctr *Executor, lastRowId int64, rowsAffected int64) (err error) {
				if invkresult := exectr.runtime.InvokeFunction(exectr.onexec, exctr, lastRowId, rowsAffected); invkresult != nil {

				}
				return
			}
		}
	}
	if onexecerror == nil {
		exectr.eventexecerror = func(err error, exctr *Executor) (bool, error) { return false, err }
	}
	if onexecerror != nil {
		if donexecerror, _ := onexecerror.(ExecErrorFunc); donexecerror != nil {
			exectr.eventexecerror = donexecerror
		}
		if donexecerror, _ := onexecerror.(func(error, *Executor) (bool, error)); donexecerror != nil {
			exectr.eventexecerror = donexecerror
		}
		if exectr.eventexecerror == nil && runtime != nil {
			exectr.eventexecerror = func(execerr error, exctr *Executor) (ignrerr bool, err error) {
				if invkresult := exectr.runtime.InvokeFunction(exectr.onexecerror, execerr, exctr); invkresult != nil {
					if ignrerrb, ignrerrok := invkresult.(bool); ignrerrok {
						ignrerr = ignrerrb
						return
					}
					if ignrerre, _ := invkresult.(error); ignrerre != nil {
						err = ignrerre
					}
				}
				return
			}
		}
	}
	return
}

func (exectr *Executor) Close() (err error) {
	if exectr != nil {
		if exectr.sqlresult != nil {
			exectr.sqlresult = nil
		}
		if exectr.stmnt != nil {
			err = exectr.stmnt.Close()
			exectr.stmnt = nil
		}
		if exectr.EventClose != nil {
			exectr.EventClose(exectr)
			exectr.EventClose = nil
		}
	}
	return
}

func (exectr *Executor) Exec() (err error) {
	if exectr != nil && exectr.stmnt != nil {
		if cn := exectr.stmnt.cn; cn != nil {
			if cn.isRemote() {

				return
			}
			if sqlcn := exectr.stmnt.sqlcn; sqlcn != nil {
				ctx := exectr.stmnt.ctx
				if ctx == nil {
					ctx = context.Background()
				}
				for _, sqls := range exectr.stmnt.stmnt {
					if exectr.sqlresult, err = sqlcn.ExecContext(ctx, sqls, exectr.stmnt.Arguments()...); err == nil {
						insertid, affected := int64(-1), int64(-1)
						if insertid, err = exectr.sqlresult.LastInsertId(); err != nil {
							insertid = -1
						}
						if affected, err = exectr.sqlresult.RowsAffected(); err != nil {
							affected = -1
							err = nil
						}
						exectr.eventexec(exectr, affected, insertid)
					}
				}
			}
		}
	}
	return
}
