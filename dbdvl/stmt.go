package dbdvl

import (
	"context"
	"database/sql/driver"
	"io"

	"github.com/lnksnk/lnksnk/fs"
)

type Stmt interface {
	driver.Stmt
	driver.StmtQueryContext
}

type dlvStmnt struct {
	fsys  fs.MultiFileSystem
	r     io.Reader
	rnr   io.RuneReader
	ctx   context.Context
	conn  *dvlConn
	query string
	input int
	conf  map[string]interface{}
}

// Close implements driver.Stmt.
func (d *dlvStmnt) Close() (err error) {
	if d == nil {
		return
	}
	d.conn = nil
	d.ctx = nil
	d.fsys = nil
	return
}

// Exec implements driver.Stmt.
func (d *dlvStmnt) Exec(args []driver.Value) (driver.Result, error) {
	panic("unimplemented")
}

// NumInput implements driver.Stmt.
func (d *dlvStmnt) NumInput() int {
	if d == nil {
		return -1
	}
	return d.input
}

func (d *dlvStmnt) QueryContext(ctx context.Context, args []driver.NamedValue) (rws driver.Rows, err error) {
	if d == nil {
		return
	}
	close := false
	if fsys, conn := d.fsys, d.conn; fsys != nil && conn != nil {
		if fi := fsys.Stat(conn.lkupPath + "/" + d.query); fi != nil {
			d.r = fi.Reader()
			d.rnr, _ = d.r.(io.RuneReader)
			close = true
		}
	}
	if r, rnr := d.r, d.rnr; r != nil || rnr != nil {
		if r != nil {
			rws = newgoreader(r, d.conf, close)
		}
	}
	return
}

// Query implements driver.Stmt.
func (d *dlvStmnt) Query(args []driver.Value) (driver.Rows, error) {
	panic("unimplemented")
}

func (d *dlvStmnt) CheckNamedValue(value *driver.NamedValue) (err error) {
	if d == nil {
		return
	}
	return
}
