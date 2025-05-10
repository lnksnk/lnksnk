package sessioning

import (
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/lnksnk/lnksnk/dbms"
	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
	"github.com/lnksnk/lnksnk/parameters"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

type Session interface {
	Path() string
	Cache() ioext.IterateMap[string, any]
	Params() parameters.Parameters
	In() serveio.Reader
	Out() serveio.Writer
	Db() dbms.DBMSHandler
	Fsys() fs.MultiFileSystem
	Sessions() Sessions
	API(...func(*SessionAPI)) *SessionAPI
	VM() SessionVM
	Close()
}

type session struct {
	owner    Session
	key      string
	path     string
	cache    ioext.IterateMap[string, any]
	in       serveio.Reader
	params   parameters.Parameters
	dspsewtr bool
	out      serveio.Writer
	db       dbms.DBMSHandler
	dspserdr bool
	fsys     fs.MultiFileSystem
	sessions Sessions
	api      *SessionAPI
	vm       SessionVM
}

// VM implements Session.
func (s *session) VM() (vm SessionVM) {
	if s == nil {
		return
	}
	if vm = s.vm; vm == nil {
		if api := s.api; api != nil {
			if api.InvokeVM != nil {
				s.vm = api.InvokeVM(s)
				return s.vm
			}
		}
	}
	return
}

// Params implements Session.
func (s *session) Params() (params parameters.Parameters) {
	if s == nil {
		return nil
	}
	if params = s.params; params != nil {
		return
	}
	if in := s.in; in != nil {
		s.params = in.Params()
		return s.params
	}
	return
}

// Close implements Session.
func (s *session) Close() {
	if s == nil {
		return
	}
	ssns := s.sessions
	s.api = nil
	in := s.in
	s.in = nil
	if s.dspserdr {
		s.dspserdr = false
		in.Close()
		in = nil
	}
	out := s.out
	s.out = nil
	if s.dspsewtr {
		s.dspsewtr = false
		out.Close()
		out = nil
	}
	s.params = nil
	db := s.db
	s.db = nil
	cache := s.cache
	s.cache = nil
	s.fsys = nil
	owner := s.owner
	s.owner = nil

	s.sessions = nil
	defer func() {
		if db != nil {
			go db.Close()
		}
		if cache != nil {
			cache.Close()
		}
		if ssns != nil {
			ssns.Close()
		}
		if owner != nil {
			if ownerref, _ := owner.(*session); ownerref != nil {
				if ownersessions := ownerref.sessions; ownersessions != nil {
					go ownersessions.Delete(s.key)
				}
			}
		}
	}()

}

// API implements Session.
func (s *session) API(confapi ...func(*SessionAPI)) (api *SessionAPI) {
	if s == nil {
		return nil
	}
	if api = s.api; api != nil {
		return
	}
	s.api = &SessionAPI{}
	if len(confapi) > 0 && confapi[0] != nil {
		confapi[0](s.api)
	}
	return s.api
}

// Sessions implements Session.
func (s *session) Sessions() (sessions Sessions) {
	if s == nil {
		return nil
	}
	sessions = s.sessions
	if sessions == nil {
		s.sessions = NewSessions(s)
		sessions = s.sessions
	}
	return
}

// Fsys implements Session.
func (s *session) Fsys() fs.MultiFileSystem {
	if s == nil {
		return nil
	}
	return s.fsys
}

// Path implements Session.
func (s *session) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Cache implements Session.
func (s *session) Cache() (cache ioext.IterateMap[string, any]) {
	if s == nil {
		return nil
	}
	cache = s.cache
	if cache != nil {
		return
	}
	cache = ioext.MapIterator[string, any]()
	s.cache = cache
	return
}

// Db implements Session.
func (s *session) Db() (db dbms.DBMSHandler) {
	if s == nil {
		return nil
	}
	if db = s.db; db != nil {
		return
	}
	if api := s.api; api != nil && api.InvodeDB != nil {
		s.db = api.InvodeDB()
		return s.db
	}
	return
}

// In implements Session.
func (s *session) In() serveio.Reader {
	if s == nil {
		return nil
	}
	return s.in
}

// Out implements Session.
func (s *session) Out() serveio.Writer {
	if s == nil {
		return nil
	}
	return s.out
}

func NewSession(ownerssn Session, a ...interface{}) (ssn Session) {
	var fsys fs.MultiFileSystem
	var db dbms.DBMSHandler
	var in serveio.Reader
	var out serveio.Writer
	var path = ""
	var rq *http.Request
	var w io.Writer
	var params parameters.Parameters
	if ownerssn != nil {
		db = ownerssn.Db()
		in = ownerssn.In()
		out = ownerssn.Out()
	}
	for _, d := range a {
		if sd, sdk := d.(string); sdk {
			if sd != "" {
				if path == "" {
					path = sd
				}
			}
			continue
		}
		if dbd, dbdk := d.(dbms.DBMSHandler); dbdk {
			if db == nil && dbd != nil {
				db = dbd
			}
			continue
		}
		if ind, indk := d.(serveio.Reader); indk {
			if in == nil && ind != nil {
				in = ind
			}
			continue
		}
		if outd, outdk := d.(serveio.Writer); outdk {
			if out == nil && outd != nil {
				out = outd
			}
			continue
		}
		if fsysd, fsysdk := d.(fs.MultiFileSystem); fsysdk {
			if fsys == nil && fsysd != nil {
				fsys = fsysd
			}
			continue
		}
		if paramsd, paramsdk := d.(parameters.Parameters); paramsdk {
			if params == nil && paramsd != nil {
				params = paramsd
			}
			continue
		}
		if rqd, rqdk := d.(*http.Request); rqdk {
			if rq == nil && rqd != nil {
				rq = rqd
				if in == nil {
					if path == "" {
						path = rq.URL.Path
					}
					in = serveio.NewContextPathReader(rq.Context(), rq, path)
				}
			}
			continue
		}
		if wd, wk := d.(io.Writer); wk {
			if w == nil && wd != nil {
				w = wd
				if out == nil {
					out = serveio.NewWriter(w)
				}
			}
			continue
		}
	}
	if params == nil && in != nil {
		params = in.Params()
	}
	if fsys == nil && ownerssn != nil {
		fsys = ownerssn.Fsys()
	}
	if path == "" && in != nil {
		path = in.Path()
	}
	ssn = &session{path: path, db: db, in: in, dspserdr: rq != nil, out: out, dspsewtr: w != nil, fsys: fsys, owner: ownerssn, params: params}
	return
}

var lastserial int64 = time.Now().UnixNano()

func nextserial() (nxsrl int64) {
	for {
		if nxsrl = time.Now().UnixNano(); atomic.CompareAndSwapInt64(&lastserial, atomic.LoadInt64(&lastserial), nxsrl) {
			break
		}
		time.Sleep(1 * time.Nanosecond)
	}
	return
}

type SessionAPI struct {
	InvodeDB   func() dbms.DBMSHandler
	InvokeVM   func(Session) SessionVM
	RunProgram func(interface{}, io.Writer)
	Eval       func(interface{}, ...map[string]interface{}) error
}
