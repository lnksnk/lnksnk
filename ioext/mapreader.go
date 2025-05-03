package ioext

import (
	"bufio"
	"io"
	"slices"
	"strings"
	"unicode/utf8"
)

type MapReader interface {
	Map() map[string]interface{}
	ReadRune() (rune, int, error)
	String() string
	Runes() []rune
	Close() error
}

type IString interface {
	String() string
}

type StringFunc func() string

func (strngfnc StringFunc) String() string {
	return strngfnc()
}

type IRunes interface {
	Runes() []rune
}

type RunesFunc func() []rune

func (rnsfnc RunesFunc) Runes() []rune {
	return rnsfnc()
}

type IMAP interface {
	Map() map[string]interface{}
}

type MAPFunc func() map[string]interface{}

func (mpfnc MAPFunc) Map() map[string]interface{} {
	return mpfnc()
}

type CloserFunc func() error

func (c CloserFunc) Close() error {
	return c()
}

var (
	emptyMapReader = &struct {
		IMAP
		io.RuneReader
		IString
		IRunes
		io.Closer
	}{IMAP: MAPFunc(func() map[string]interface{} {
		return nil
	}), RuneReader: ReadRuneFunc(func() (rune, int, error) {
		return 0, 0, io.EOF
	}), IString: StringFunc(func() string {
		return ""
	}), IRunes: RunesFunc(func() []rune {
		return nil
	}), Closer: CloserFunc(func() error {
		return nil
	})}
)

func MapReplaceReader(in interface{}, mp map[string]interface{}, flshunmtchd func(string) bool, a ...interface{}) (mprdr MapReader) {
	var readrune, _ = in.(func() (rune, int, error))
	if readrune == nil {
		if r, rk := in.(io.Reader); rk {
			rdr, _ := r.(io.RuneReader)
			if rdr == nil {
				rdr = bufio.NewReaderSize(r, 1)
			}
			readrune = rdr.ReadRune
		} else if s, sk := in.(string); sk {
			if s == "" {
				return emptyMapReader
			}
			readrune = strings.NewReader(s).ReadRune
		} else if int32arr, intk := in.([]int32); intk {
			if len(int32arr) == 0 {
				return emptyMapReader
			}
			rns := make([]rune, len(int32arr))
			copy(rns, int32arr)
			readrune = ReadRuneFunc(func() (r rune, size int, err error) {
				if len(rns) > 0 {
					r = rns[0]
					rns = rns[1:]
					size = utf8.RuneLen(r)
					return
				}
				return 0, 0, io.EOF
			})
		} else if bf, bfk := in.(*Buffer); bfk {
			if bf.Empty() {
				return emptyMapReader
			} else {
				readrune = bf.Reader().ReadRune
			}
		}
	}
	mxklen := 0
	mnklen := 0
	keys := [][]rune{}
	//var skeys []string
	for k := range mp {
		keys = append(keys, []rune(k))
		if mnklen == 0 || mnklen > len(keys[len(keys)-1]) {
			mnklen = len(keys[len(keys)-1])
		}
		if mxklen == 0 || mxklen < len(keys[len(keys)-1]) {
			mxklen = len(keys[len(keys)-1])
		}
	}
	slices.SortFunc(keys, func(a []rune, b []rune) int {
		al, bl := len(a), len(b)
		if al <= bl {
			for an, ar := range a {
				if ar < b[an] {
					return -1
				}
				if ar > b[an] {
					return 1
				}
			}
			return 0
		}
		for bn, br := range b {
			if a[bn] < br {
				return -1
			}
			if a[bn] > br {
				return 1
			}
		}
		return 0
	})
	var validatnamerune func(prvr, r rune) bool
	var prepost []string
	for _, d := range a {
		if vdlrd, vdlrk := d.(func(rune, rune) bool); vdlrk {
			if validatnamerune == nil && vdlrd != nil {
				validatnamerune = vdlrd
			}
			continue
		}
		if prs, prsk := d.(string); prsk {
			if prs == "" {
				continue
			}
			if len(prepost) < 2 {
				prepost = append(prepost, prs)
			}
		}
	}
	if validatnamerune == nil {
		validatnamerune = func(prvr, r rune) bool {
			return true
		}
	}
	if len(prepost) == 0 {
		return &mapreader{mp: mp, keys: keys, orgreadrne: readrune, vldchr: validatnamerune}
	}
	if len(prepost) > 0 && prepost[0] != "" && prepost[1] != "" {
		if flshunmtchd == nil {
			flshunmtchd = func(s string) bool {
				return false
			}
		}
		return &mapprepostreader{mapreader: &mapreader{mp: mp, keys: keys, orgreadrne: readrune, vldchr: validatnamerune}, keys: keys, mnklen: mnklen, mxklen: mxklen, pre: []rune(prepost[0]), prel: len([]rune(prepost[0])), post: []rune(prepost[1]), postl: len([]rune(prepost[1])), flshunmtchd: flshunmtchd}
	}
	return
}

type mapreader struct {
	orgreadrne func() (rune, int, error)
	orgrnes    []rune
	bfrnes     []rune
	bfrdr      io.RuneReader
	mp         map[string]interface{}
	keys       [][]rune
	vldchr     func(rune, rune) bool
}

// Close implements MapReader.
func (m *mapreader) Close() error {
	if m != nil {
		m.bfrdr = nil
		m.bfrnes = nil
		m.keys = nil
		m.orgreadrne = nil
		m.orgrnes = nil
		m.mp = nil
		m.vldchr = nil
	}
	return nil
}

// Map implements MapReader.
func (m *mapreader) Map() map[string]interface{} {
	if m == nil {
		return nil
	}
	return m.mp
}

func (m *mapreader) OrgRune() (r rune, size int, err error) {
	if m != nil {
		if len(m.orgrnes) > 0 {
			r = m.orgrnes[0]
			m.orgrnes = m.orgrnes[1:]
			size = utf8.RuneLen(r)
			return r, size, nil
		}
		if m.orgreadrne != nil {
			if r, size, err = m.orgreadrne(); size > 0 {
				return r, size, err
			}
			if err != nil {
				m.orgreadrne = nil
			}
			return r, size, err
		}
	}
	return 0, 0, io.EOF
}

func (m *mapreader) capture(key string) bool {
	if m == nil || len(m.mp) == 0 {
		return false
	}
	if v, vk := m.mp[key]; vk {
		return tobuf(m, v)
	}
	return false
}

func (m *mapreader) readKey() (r rune, size int, err error, cptrdkey bool) {
rdnxt:
	if r, size, err = m.OrgRune(); size > 0 {
		if len(m.keys) == 0 {
			return
		}
		var keys [][]rune
		for kn, k := range m.keys {
			if k[0] == r {
				keys = append(keys, k)
			}
			if ksl, ksfndl := len(m.keys), len(keys); kn == ksl-1 && ksfndl > 0 {
				if ksfndl == 1 {
					if len(keys[0]) == 1 {
						if m.capture(string(keys[0])) {
							return 0, 0, nil, true
						}
						goto rdnxt
					}
				}
				ki := 1
				kn = 0
				kl := len(keys[kn])
				var lstkey []rune
			rdrnxtr:
				rk, rs, re := m.OrgRune()
				if re != nil || rs == 0 {
					if re != nil && re != io.EOF {
						return 0, 0, re, false
					}
					if len(lstkey) > 0 {
						if m.capture(string(lstkey)) {
							return 0, 0, nil, true
						}
					}
					return 0, 0, io.EOF, false
				}
			chknxtk:
				if kl-1 >= ki && keys[kn][ki] == rk {
					if kl-1 == ki {
						lstkey = keys[kn]
						if ksfndl == 1 {
							if m.capture(string(lstkey)) {
								return 0, 0, nil, true
							}
							goto rdnxt
						}
					}
					if kn++; kn < ksfndl {
						kl = len(keys[kn])
						goto chknxtk
					}
					if kn == ksfndl {
						kn = 0
						ki++
						for kn < ksfndl {
							if kl = len(keys[kn]); kl <= ki {
								keys = append(keys[:kn], keys[kn+1:]...)
								ksfndl--
								continue
							}
							kn++
						}
						kn = 0
						goto rdrnxtr
					}
				}
				if ksfndl == 1 {
					if len(lstkey) > 0 {
						m.orgrnes = append(m.orgrnes, rk)
						if m.capture(string(lstkey)) {
							return 0, 0, nil, true
						}
						goto rdnxt
					}
					m.orgrnes = append(m.orgrnes, keys[kn][1:kl]...)
					m.orgrnes = append(m.orgrnes, rk)
					goto rdnxt
				}
				keys = append(keys[:kn], keys[kn+1:]...)
				ksfndl--
				goto chknxtk
			}
		}
		return r, size, nil, false
	}
	return
}

// ReadRune implements MapReader.
func (m *mapreader) ReadRune() (r rune, size int, err error) {
	if m != nil {
	rdbf:
		if m.bfrdr != nil {
			r, size, err = m.bfrdr.ReadRune()
			if size > 0 {
				return
			}
			if err != nil {
				if err != io.EOF {
					m.Close()
					return
				}
				m.bfrdr = nil
				err = nil
			}
		}
		if len(m.bfrnes) > 0 {
			r = m.bfrnes[0]
			m.bfrnes = m.bfrnes[1:]
			size = utf8.RuneLen(r)
			return
		}

		cptrdk := false
		for {
			if r, size, err, cptrdk = m.readKey(); cptrdk {
				goto rdbf
			}
			if size > 0 {
				return r, size, nil
			}
			m.Close()
			return 0, 0, err
		}
	}
	return 0, 0, err
}

// Runes implements MapReader.
func (m *mapreader) Runes() (rns []rune) {
	for m != nil {
		r, s, _ := m.ReadRune()
		if s > 0 {
			rns = append(rns, r)
			continue
		}
		return
	}
	return
}

// String implements MapReader.
func (m *mapreader) String() string {
	return string(m.Runes())
}

type mapprepostreader struct {
	*mapreader
	pre         []rune
	prei        int
	pvr         rune
	prel        int
	post        []rune
	posti       int
	postl       int
	tstkeyrns   []rune
	flshunmtchd func(string) bool
	//unmatchd    func(interface{}) interface{}
	keys   [][]rune
	mnklen int
	mxklen int
}

func (mppr *mapprepostreader) Close() (err error) {
	if mppr != nil {
		m := mppr.mapreader
		mppr.mapreader = nil
		if m != nil {
			m.Close()
		}
		mppr.tstkeyrns = nil
		mppr.posti = 0
		mppr.prei = 0
		mppr.pre = nil
		mppr.post = nil
		mppr.pvr = 0
		mppr.keys = nil
	}
	return
}

func (mppr *mapprepostreader) Runes() (rns []rune) {
	for {
		r, s, _ := mppr.ReadRune()
		if s > 0 {
			rns = append(rns, r)
			continue
		}
		break
	}
	return
}

func (mppr *mapprepostreader) String() string {
	return string(mppr.Runes())
}

func (mppr *mapprepostreader) Map() map[string]interface{} {
	if mppr == nil {
		return nil
	}
	return mppr.mp
}

func (mppr *mapprepostreader) OrgRune() (r rune, size int, err error) {
	if mppr != nil {
		if len(mppr.orgrnes) > 0 {
			r = mppr.orgrnes[0]
			mppr.orgrnes = mppr.orgrnes[1:]
			size = utf8.RuneLen(r)
			return r, size, nil
		}
		if mppr.orgreadrne != nil {
			if r, size, err = mppr.orgreadrne(); size > 0 {
				return r, size, err
			}
			if err != nil {
				mppr.orgreadrne = nil
			}
			return r, size, err
		}
	}
	return 0, 0, io.EOF
}

func tobuf(m *mapreader, v interface{}) bool {
	if m == nil {
		return false
	}
	if s, sk := v.(string); sk {
		if s == "" {
			return false
		}
		m.bfrnes = append(m.bfrnes, []rune(s)...)
		return true
	}
	if int32r, int32k := v.([]int32); int32k {
		if len(int32r) == 0 {
			return false
		}
		m.bfrnes = append(m.bfrnes, int32r...)
		return true
	}
	if b, bk := v.(*Buffer); bk {
		if b.Empty() {
			return false
		}
		m.bfrdr = b.Reader()
		return true
	}
	if rdr, rdrk := v.(io.RuneReader); rdrk {
		m.bfrdr = rdr
		return true
	}
	if r, rk := v.(io.Reader); rk {
		m.bfrdr = bufio.NewReader(r)
		return true
	}
	return false
}

func (mppr *mapprepostreader) ReadRune() (r rune, size int, err error) {
	if mppr != nil {
	rdbf:
		if mppr.bfrdr != nil {
			r, size, err = mppr.bfrdr.ReadRune()
			if size > 0 {
				return
			}
			if err != nil {
				if err != io.EOF {
					mppr.Close()
					return
				}
				mppr.bfrdr = nil
				err = nil
			}
		}
		if len(mppr.bfrnes) > 0 {
			r = mppr.bfrnes[0]
			mppr.bfrnes = mppr.bfrnes[1:]
			size = utf8.RuneLen(r)
			return
		}
	rdnxt:
		r, size, err = mppr.OrgRune()
		if size > 0 {
			if mppr.posti == 0 && mppr.prei < mppr.prel {
				if mppr.prei > 0 && mppr.pre[mppr.prei-1] == mppr.pvr && mppr.pre[mppr.prei] != r {
					mppr.bfrnes = append(mppr.bfrnes, mppr.pre[:mppr.prei]...)
					mppr.prei = 0
					mppr.pvr = 0
					mppr.orgrnes = append(mppr.orgrnes, r)
					goto rdbf
				}
				if mppr.pre[mppr.prei] == r {
					if mppr.prei++; mppr.prei == mppr.prel {

						goto rdnxt
					}
					mppr.pvr = r
					goto rdnxt
				}
				if mppr.prei > 0 {
					mppr.bfrnes = append(mppr.bfrnes, mppr.pre[:mppr.prei]...)
					mppr.prei = 0
					mppr.pvr = 0
					mppr.orgrnes = append(mppr.orgrnes, r)
					goto rdbf
				}
				mppr.pvr = r
				return r, size, nil
			}
			if mppr.prei == mppr.prel && mppr.posti < mppr.prel {
				if mppr.post[mppr.posti] == r {
					if mppr.posti++; mppr.posti == mppr.postl {
						mppr.posti = 0
						mppr.prei = 0
						mppr.pvr = 0
						if len(mppr.tstkeyrns) > 0 {
							if mppr.capture(string(mppr.tstkeyrns)) {
								mppr.tstkeyrns = nil
								goto rdbf
							}
							if flshunmtchd := mppr.flshunmtchd; flshunmtchd != nil {
								if flshunmtchd(string(mppr.tstkeyrns)) {
									mppr.bfrnes = append(mppr.bfrnes, mppr.pre...)
									mppr.bfrnes = append(mppr.bfrnes, mppr.tstkeyrns...)
									mppr.tstkeyrns = nil
									mppr.bfrnes = append(mppr.bfrnes, mppr.post...)
									goto rdbf
								}
							}
							mppr.tstkeyrns = nil
						}
						goto rdnxt
					}
					goto rdnxt
				}
				if invld := !mppr.vldchr(0, r); invld || mppr.posti > 0 {
					mppr.bfrnes = append(mppr.bfrnes, mppr.pre...)
					mppr.prei = 0
					mppr.bfrnes = append(mppr.bfrnes, mppr.tstkeyrns...)
					mppr.tstkeyrns = nil
					mppr.bfrnes = append(mppr.bfrnes, mppr.post[:mppr.posti]...)
					mppr.posti = 0
					mppr.orgrnes = append(mppr.orgrnes, r)
					goto rdbf
				}
				mppr.tstkeyrns = append(mppr.tstkeyrns, r)
			}
			goto rdnxt
		}
		if err != nil {
			if err != io.EOF {
				return
			}
		}
		if mppr.prei > 0 {
			mppr.bfrnes = append(mppr.bfrnes, mppr.pre[:mppr.prei]...)
			mppr.prei = 0
		}
		if len(mppr.tstkeyrns) > 0 {
			mppr.bfrnes = append(mppr.bfrnes, mppr.tstkeyrns...)
			mppr.tstkeyrns = nil
		}
		if mppr.posti > 0 {
			mppr.bfrnes = append(mppr.bfrnes, mppr.post[:mppr.posti]...)
			mppr.posti = 0
		}
		if len(mppr.bfrnes) > 0 {
			goto rdbf
		}
		mppr.Close()
		return 0, 0, err
	}
	if size == 0 {
		mppr.Close()
		return 0, 0, err
	}
	return
}

func init() {

	/*for {
		tst := MapReplaceReader("{@dat@}{@dat@} {@dat@}{@dat@} {@dat@}", map[string]interface{}{
			"{@": "${", "@}": "}", "@}{@": "}${"})
		if nxts := tst.String(); nxts != "${dat}${dat} ${dat}${dat} ${dat}" {
			fmt.Println(nxts)
		}
	}*/

}
