package sessioning

import (
	"fmt"
	"net/http"

	"github.com/lnksnk/lnksnk/ioext"
)

type SessionsAPI struct {
	ServeHttp func(Sessions, Session, http.ResponseWriter, *http.Request)
}

type Sessions interface {
	Owner() Session
	ioext.IterateMap[string, Session]
	New(string, ...interface{}) Session
	ServeHTTP(http.ResponseWriter, *http.Request)
	API(...func(*SessionsAPI)) *SessionsAPI
	UniqueKey(...string) string
}

type sessions struct {
	owener Session
	api    *SessionsAPI
	ioext.IterateMap[string, Session]
}

// UniqueKey implements Sessions.
func (s *sessions) UniqueKey(prepost ...string) string {
	if s == nil {
		return ""
	}
	if len(prepost) == 1 {
		return fmt.Sprintf("%s%v", prepost[0], nextserial())
	}
	if len(prepost) > 1 {
		return fmt.Sprintf("%s%v%s", prepost[0], nextserial(), prepost[0])
	}
	return fmt.Sprintf("%v", nextserial())
}

// API implements Sessions.
func (s *sessions) API(configapi ...func(*SessionsAPI)) (api *SessionsAPI) {
	if s == nil {
		return
	}
	if api = s.api; api != nil {
		return
	}
	api = &SessionsAPI{}
	s.api = api
	if len(configapi) > 0 && configapi[0] != nil {
		configapi[0](api)
	}
	return
}

// ServeHTTP implements Sessions.
func (s *sessions) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		return
	}
	if api := s.api; api != nil {
		if srvhttp := api.ServeHttp; srvhttp != nil {
			ssn := s.New("", w, r)
			defer func() {
				ssn.Close()
				ssn = nil
			}()
			srvhttp(s, ssn, w, r)
		}
	}
}

// New implements Sessions.
func (s *sessions) New(name string, a ...interface{}) (ssn Session) {
	if s == nil {
		return nil
	}
	if name != "" {
		if ssn, _ = s.Get(name); ssn == nil {
			ssn = NewSession(s.Owner(), a...)
			if ssnref, _ := ssn.(*session); ssnref != nil {
				ssnref.key = name
			}
			s.Set(name, ssn)
		}
		return
	}
	name = s.UniqueKey()
	ssn = NewSession(s.owener, a...)
	if ssnref, _ := ssn.(*session); ssnref != nil {
		ssnref.key = name
	}
	s.Set(name, ssn)
	return
}

// Clear implements Sessions.
// Subtle: this method shadows the method (IterateMap).Clear of sessions.IterateMap.
func (s *sessions) Clear() {
	if s == nil {
		return
	}
	if itr := s.IterateMap; itr != nil {
		itr.Clear()
	}
}

// Close implements Sessions.
// Subtle: this method shadows the method (IterateMap).Close of sessions.IterateMap.
func (s *sessions) Close() {
	if s == nil {
		return
	}
	owner, _ := s.owener.(*session)
	s.owener = nil
	if itr := s.IterateMap; itr != nil {
		s.IterateMap = nil
		go itr.Close()
	}
	if owner != nil {
		owner.sessions = nil
	}
}

// Contains implements Sessions.
// Subtle: this method shadows the method (IterateMap).Contains of sessions.IterateMap.
func (s *sessions) Contains(name string) bool {
	if s == nil || name == "" {
		return false
	}
	if itr := s.IterateMap; itr != nil {
		return itr.Contains(name)
	}
	return false
}

// Delete implements Sessions.
// Subtle: this method shadows the method (IterateMap).Delete of sessions.IterateMap.
func (s *sessions) Delete(name ...string) {
	if s == nil || len(name) == 0 {
		return
	}
	if itr := s.IterateMap; itr != nil {
		itr.Delete(name...)
	}
}

// Events implements Sessions.
// Subtle: this method shadows the method (IterateMap).Events of sessions.IterateMap.
func (s *sessions) Events() ioext.IterateMapEvents[string, Session] {
	if s == nil {
		return nil
	}
	if itr := s.IterateMap; itr != nil {

	}
	return nil
}

// Get implements Sessions.
// Subtle: this method shadows the method (IterateMap).Get of sessions.IterateMap.
func (s *sessions) Get(name string) (value Session, found bool) {
	if s == nil {
		return
	}
	if itr := s.IterateMap; itr != nil {
		value, found = itr.Get(name)
	}
	return
}

// Iterate implements Sessions.
// Subtle: this method shadows the method (IterateMap).Iterate of sessions.IterateMap.
func (s *sessions) Iterate() func(func(string, Session) bool) {
	if s == nil {
		return func(f func(string, Session) bool) {
		}
	}
	if itr := s.IterateMap; itr != nil {
		return itr.Iterate()
	}
	return func(f func(string, Session) bool) {
	}
}

// Owner implements Sessions.
func (s *sessions) Owner() Session {
	if s == nil {
		return nil
	}
	return s.owener
}

// Set implements Sessions.
// Subtle: this method shadows the method (IterateMap).Set of sessions.IterateMap.
func (s *sessions) Set(name string, value Session) {
	if s == nil || name == "" || value == nil {
		return
	}
	if itr := s.IterateMap; itr != nil {
		itr.Set(name, value)
		if ssnref, _ := value.(*session); ssnref != nil {
			ssnref.key = name
		}
	}
}

func NewSessions(a ...interface{}) (ssns Sessions) {
	var ownerssn Session
	for _, d := range a {
		if ssnd, ssndk := d.(Session); ssndk {
			if ownerssn == nil && ssnd != nil {
				ownerssn = ssnd
			}
		}
	}
	return &sessions{IterateMap: ioext.MapIterator[string, Session](), owener: ownerssn}
}
