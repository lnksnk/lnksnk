package dbdvl

import (
	"database/sql/driver"
	gocsv "encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/lnksnk/lnksnk/iorw"
)

type gocvsreader struct {
	*gocsv.Reader
	hdrs        bool
	firstrecord []string
	record      []string
	lsterr      error
	cols        []string
	cr          io.Closer
}

// Close implements driver.Rows.
func (g *gocvsreader) Close() (err error) {
	if g == nil {

	}
	cr := g.cr
	g.cr = nil
	g.Reader = nil
	if cr != nil {
		cr.Close()
	}
	return
}

// Columns implements driver.Rows.
func (g *gocvsreader) Columns() (cols []string) {
	if g == nil {
		return
	}
	cols = g.cols
	if len(cols) == 0 {
		Reader, lsterr := g.Reader, g.lsterr
		if lsterr == nil && Reader != nil {
			if !g.hdrs {
				g.firstrecord, lsterr = Reader.Read()
				if lsterr != nil {
					g.lsterr = lsterr
					return
				}
				if frstrcl := len(g.firstrecord); frstrcl > 0 {
					cols = make([]string, frstrcl)
					for cn := range frstrcl {
						cols[cn] = fmt.Sprintf("%s%d", "Column", cn+1)
					}
					g.cols = cols
				}
				return
			}

			cols, lsterr = Reader.Read()
			if lsterr != nil {
				g.lsterr = lsterr
				return
			}
			g.cols = cols
		}
	}
	return
}

// Next implements driver.Rows.
func (g *gocvsreader) Next(dest []driver.Value) (err error) {
	if g == nil {
		return
	}

	cols, record, Reader, firstrecord := g.cols, g.record, g.Reader, g.firstrecord
	clsl := len(cols)
	if clsl == 0 {
		clsl, err = len(g.Columns()), g.lsterr
		if err != nil {
			return
		}
	}
	if Reader != nil {
		if len(firstrecord) == clsl {
			record = firstrecord
			g.firstrecord = nil
		} else {
			if record, err = Reader.Read(); err != nil {
				g.lsterr = err
				return
			}
		}
		g.record = record
		rcrdl := len(record)
		if rcrdl == clsl && len(dest) == rcrdl {
			for rn := range rcrdl {
				for n, r := range record[rn] {
					if iorw.IsSpace(r) {
						if n == rcrdl-1 {
							record[rn] = ""
							break
						}
						continue
					}
					record[rn] = record[rn][n:]
					tl := len(record[rn])
					for tn := range record[rn] {
						if iorw.IsSpace(rune(record[rn][tl-(tn+1)])) {
							if tn == tl-1 {
								record[rn] = ""
								break
							}
							continue
						}
						record[rn] = record[rn][:tl-(tn)]
						dest[rn] = strings.TrimFunc(record[rn], iorw.IsSpace)
						break
					}
					break
				}
			}
			return
		}
		return
	}
	return
}

func newgoreader(r io.Reader, conf map[string]interface{}, close bool) (gordr *gocvsreader) {
	comma := ';'
	headers := true

	for cfk, cfv := range conf {
		if strings.EqualFold(cfk, "coldelim") {
			if cr, crk := cfv.(int32); crk {
				comma = rune(cr)
				continue
			}
			if cs, ck := cfv.(string); ck {
				if cs != "" {
					comma = []rune(cs)[0]
				}
			}
			continue
		}
		if strings.EqualFold(cfk, "headers") {
			if hv, hk := cfv.(bool); hk {
				headers = hv
				continue
			}
			continue
		}
	}
	gordr = &gocvsreader{hdrs: headers, Reader: gocsv.NewReader(r), cr: func() io.Closer {
		if close {
			clr, _ := r.(io.Closer)
			return clr
		}
		return nil
	}()}
	gordr.Comma = comma
	gordr.LazyQuotes = true
	gordr.ReuseRecord = true
	return
}
