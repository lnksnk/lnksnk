package template

import (
	"bufio"
	"io"
	"strings"

	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
	"github.com/lnksnk/lnksnk/iorw"
)

type markuptemplate struct {
	fsys        fs.MultiFileSystem
	fi          fs.FileInfo
	cntntprsngs map[int]*contentparsing
	prsix       int
	invalidElem map[string]bool
	validElem   map[string]fs.FileInfo
	cntntbf     *iorw.Buffer
	cdebf       *iorw.Buffer
}

func (m *markuptemplate) Code() *iorw.Buffer {
	if m == nil {
		return nil
	}
	return m.cdebf
}

func (m *markuptemplate) Content() *iorw.Buffer {
	if m == nil {
		return nil
	}
	return m.cntntbf
}

func (m *markuptemplate) InvalidElements() map[string]bool {
	if m == nil {
		return nil
	}
	return m.invalidElem
}

func (m *markuptemplate) ValidElements() map[string]fs.FileInfo {
	if m == nil {
		return nil
	}
	return m.validElem
}

func (m *markuptemplate) Parse(in interface{}) {
	if m == nil {
		return
	}
	var nxtrdr = func(inrd io.RuneReader) io.RuneReader {
		if m.prsix == 0 {
			if c, ck := m.cntntprsngs[m.prsix]; ck {
				if c != nil {
					return ioext.MapReplaceReader(inrd, map[string]interface{}{
						"p-root":   c.fi.Root(),
						"p-base":   c.fi.Base(),
						"p-e-root": c.elmroot,
						"p-e-base": c.elmbase}, "[#", "#]")
				}
			}
		}
		return inrd
	}
	if r, rok := in.(io.Reader); rok {
		rdr, _ := r.(io.RuneReader)
		if rdr == nil {
			rdr = bufio.NewReaderSize(r, 1)
		}
		parseReader(nxtrdr(rdr), m)
		return
	}
	if rdr, rdrk := in.(io.RuneReader); rdrk {
		parseReader(nxtrdr(rdr), m)
		return
	}
	if s, sk := in.(string); sk {
		parseReader(nxtrdr(strings.NewReader(s)), m)
		return
	}
	if bf, bfk := in.(*iorw.Buffer); bfk {
		parseReader(nxtrdr(bf.Reader()), m)
	}
}

func (m *markuptemplate) Wrapup() {
	if m == nil {
		return
	}
	if len(m.cntntprsngs) == 1 && m.prsix == 0 {
		m.cntntprsngs[m.prsix].Close()
	}
}

func parseReader(rdr io.RuneReader, m *markuptemplate) error {
	for {
		r, size, err := rdr.ReadRune()
		if size > 0 {
			parseRune(r, m)
			continue
		}
		if size == 0 && err == nil {
			err = nil
		}
		return err
	}
}

func parseRune(r rune, m *markuptemplate) {
	ctntp, ck := m.cntntprsngs[m.prsix]
	if ck {
		ctntp.parse(r)
	}
}

func MarkupTemplate(a ...interface{}) (m *markuptemplate) {
	var fsys fs.MultiFileSystem
	var fi fs.FileInfo
	for _, d := range a {
		if fsysd, fsysk := d.(fs.MultiFileSystem); fsysk {
			if fsys == nil {
				fsys = fsysd
			}
			continue
		}
		if fid, fik := d.(fs.FileInfo); fik {
			if fi == nil {
				fi = fid
			}
			continue
		}
	}
	m = &markuptemplate{fsys: fsys, fi: fi}
	m.cntntprsngs = map[int]*contentparsing{len(m.cntntprsngs): nextContentParsing(nil, m, fsys, fi)}
	return
}
