package dbms

import (
	"context"
	"database/sql"
	"io"
	"strings"

	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/parameters"
)

type Statement interface {
	Query(...interface{}) (Reader, error)
	QueryContext(context.Context, ...interface{}) (Reader, error)
	Execute(...interface{}) error
	ExecuteContext(context.Context, ...interface{}) error
	Rows() Rows
	Close()
}

type statement struct {
	dbcn *sql.DB
	//dbstmt   *sql.Stmt
	dbrows   *sql.Rows
	rows     Rows
	driver   string
	query    string
	err      error
	fsys     fs.MultiFileSystem
	fireader func(fs.MultiFileSystem, fs.FileInfo, io.Writer)
	params   parameters.ParametersAPI
	rdr      Reader
}

// Close implements Statement.
func (s *statement) Close() {
	if s == nil {
		return
	}
	dbrows, rows := s.dbrows, s.rows
	s.dbcn = nil
	s.dbrows = nil
	s.rows = nil
	if dbrows != nil {
		dbrows.Close()
	}
	if rows != nil {
		rows.Close()
	}
}

// Rows implements Statement.
func (s *statement) Rows() Rows {
	if s == nil {
		return nil
	}
	return s.rows
}

// Execute implements Statement.
func (s *statement) Execute(a ...interface{}) (err error) {
	if s == nil {
		return
	}
	return s.ExecuteContext(context.Background(), a...)
}

// ExecuteContext implements Statement.
func (s *statement) ExecuteContext(ctx context.Context, a ...interface{}) (err error) {
	if s == nil {
		return
	}

	return
}

// Query implements Statement.
func (s *statement) Query(a ...interface{}) (rdr Reader, err error) {
	if s == nil {
		return
	}
	return s.QueryContext(context.Background(), a...)
}

// QueryContext implements Statement.
func (s *statement) QueryContext(ctx context.Context, a ...interface{}) (rdr Reader, err error) {
	if s == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
		defer func() {
			ctx = nil
		}()
	}
	params := s.params
	srdr := s.rdr
	fireader := s.fireader
	if al := len(a); al > 0 {
		ai := 0
		for ai < al {
			if prmsd, prmsk := a[ai].(parameters.ParametersAPI); prmsk {
				if params == nil && prmsd != nil {
					params = prmsd
					s.params = params
				}
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			if rdrd, rdrk := a[ai].(Reader); rdrk {
				if srdr == nil && rdrd != nil {
					srdr = rdrd
					s.rdr = srdr
				}
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			if firdrd, firdrk := a[ai].(func(fs.MultiFileSystem, fs.FileInfo, io.Writer)); firdrk {
				if fireader == nil && firdrd != nil {
					fireader = firdrd
					s.fireader = fireader
				}
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			ai++
		}
	}
	dbcn, dbrows := s.dbcn, s.dbrows
	if dbcn != nil {
		if dbrows == nil {
			if fsys := s.fsys; fsys != nil {
				s.fsys = nil
				tstquery := s.query
				tstext := func() string {
					if si := strings.LastIndex(tstquery, "."); si > 0 {
						if tstquery[si:] == ".sql" {
							tstquery = tstquery[:si]
							return ".sql"
						}
						return ".sql"
					}
					return ".sql"
				}()
				fi := fsys.Stat(tstquery + tstext)

				if fi == nil {
					fi = fsys.Stat(tstquery + "." + s.driver + tstext)
				}
				if fi != nil {
					if fireader != nil {
						qrybf := iorw.NewBuffer()
						defer qrybf.Close()
						fireader(fsys, fi, qrybf)
						s.query = qrybf.String()
					} else {
						s.query, _ = iorw.ReaderToString(fi.Reader())
					}
				}
			}

			if dbrows, err = dbcn.QueryContext(ctx, s.query, a...); err == nil {
				s.dbrows = dbrows
				s.rows = &rows{dbrws: dbrows}
				rdr = &reader{stmnt: s, rws: s.rows}
				return
			}
		}
		return
	}
	return
}

func nextstatement(dbcn *sql.DB, driver string, query string, fsys fs.MultiFileSystem) (stmnt Statement) {
	return &statement{dbcn: dbcn, driver: driver, query: query, fsys: fsys}
}
