package dbdvl

import (
	"context"
	"database/sql/driver"
	"io"

	"github.com/lnksnk/lnksnk/fs"
)

type dvlConn struct {
	lkupPath string
	dvr      *dvlDriver
}

func (d *dvlConn) CheckNamedValue(value *driver.NamedValue) (err error) {
	if d == nil {
		return
	}
	return
}

// Begin implements driver.Conn.
func (d *dvlConn) Begin() (driver.Tx, error) {
	panic("unimplemented")
}

// Close implements driver.Conn.
func (d *dvlConn) Close() error {
	if d == nil {
		return nil
	}
	if drv := d.dvr; drv != nil {
		drv.Delete(d.lkupPath)
	}
	return nil
}

func (d *dvlConn) PrepareContext(ctx context.Context, query string) (stmt driver.Stmt, err error) {
	if d == nil {
		return
	}
	stmt = &dlvStmnt{query: query, conn: d, ctx: ctx, input: 1}
	return
}

// Prepare implements driver.Conn.
func (d *dvlConn) Prepare(query string) (stmt driver.Stmt, err error) {
	if d == nil {
		return
	}
	stmt = &dlvStmnt{query: query, conn: d, input: 1}
	return
}

func (d *dvlConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	if d == nil {
		return
	}
	ai := 0
	al := len(args)
	var fsys fs.MultiFileSystem
	var rnr io.RuneReader
	var r io.Reader
	var conf map[string]interface{}
	if al > 0 {
		for ai < al {
			if fsysd, fsysk := args[ai].Value.(fs.MultiFileSystem); fsysk {
				if fsys == nil && fsysd != nil {
					fsys = fsysd
				}
				for _, nmdv := range args[ai+1:] {
					nmdv.Ordinal--
				}
				args = append(args[:ai], args[ai+1:]...)
				al--
				continue
			}
			if rnrd, rnrk := args[ai].Value.(io.RuneReader); rnrk {
				if rnr == nil && rnrd != nil {
					rnr = rnrd
					if rd, rk := rnr.(io.Reader); rk {
						if r == nil && rd != nil {
							r = rd
						}
					}
				}
				for _, nmdv := range args[ai+1:] {
					nmdv.Ordinal--
				}
				args = append(args[:ai], args[ai+1:]...)
				al--
				continue
			}
			if rd, rk := args[ai].Value.(io.Reader); rk {
				if r == nil && rd != nil {
					r = rd
					if rnrd, rnrk := r.(io.RuneReader); rnrk {
						if rnr == nil && rnrd != nil {
							rnr = rnrd
						}
					}
				}
				for _, nmdv := range args[ai+1:] {
					nmdv.Ordinal--
				}
				args = append(args[:ai], args[ai+1:]...)
				al--
				continue
			}
			if confd, confk := args[ai].Value.(map[string]interface{}); confk {
				if conf == nil && len(confd) > 0 {
					conf = map[string]interface{}{}
					for cfk, cfv := range confd {
						conf[cfk] = cfv
					}
				}
				args = append(args[:ai], args[ai+1:]...)
				al--
				continue
			}
			ai++
		}
		var stmt Stmt = &dlvStmnt{conf: conf, query: query, conn: d, fsys: fsys, r: r, rnr: rnr}
		defer stmt.Close()
		rows, err = stmt.QueryContext(ctx, args)
	}
	return
}
