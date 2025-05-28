package sessioning

import (
	"io"

	"github.com/lnksnk/lnksnk/ioext"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

type SessionVM interface {
	VM() interface{}
	Session() Session
	Set(string, interface{}) error
	SetWriter(io.Writer)
	Writer() io.Writer
	Print(...interface{}) error
	Println(...interface{}) error
}

type sessionvm struct {
	orgvm interface{}
	ssn   Session
	wrtrs []io.Writer
}

// Writer implements SessionVM.
func (s *sessionvm) Writer() io.Writer {
	if s == nil {
		return nil
	}
	if wtrsl := len(s.wrtrs); wtrsl > 0 {
		return s.wrtrs[wtrsl-1]
	}
	return nil
}

// SetWriter implements SessionVM.
func (s *sessionvm) SetWriter(wrtr io.Writer) {
	if s != nil {
		if wrtr != nil {
			s.wrtrs = append(s.wrtrs, wrtr)
			return
		}
		wtrsl := len(s.wrtrs)
		if wtrsl > 0 {
			if wtrsl == 1 {
				s.wrtrs = nil
				return
			}
			s.wrtrs = s.wrtrs[:wtrsl-1]
			return
		}
	}
}

// Print implements SessionVM.
func (s *sessionvm) Print(a ...interface{}) (err error) {
	if s == nil || len(a) == 0 {
		return
	}
	if wtrsl := len(s.wrtrs); wtrsl > 0 {
		if wrtr := s.wrtrs[wtrsl-1]; wrtr != nil {
			if srvwtr, _ := wrtr.(serveio.Writer); srvwtr != nil {
				return srvwtr.Print(a...)
			}
			err = ioext.Fprint(wrtr, a...)
		}
	}
	return
}

// Println implements SessionVM.
func (s *sessionvm) Println(a ...interface{}) (err error) {
	if s == nil || len(a) == 0 {
		return
	}
	if wtrsl := len(s.wrtrs); wtrsl > 0 {
		if wrtr := s.wrtrs[wtrsl-1]; wrtr != nil {
			if srvwtr, _ := wrtr.(serveio.Writer); srvwtr != nil {
				return srvwtr.Println(a...)
			}
			err = ioext.Fprintln(wrtr, a...)
		}
	}
	return
}

// VM implements SessionVM.
func (s *sessionvm) VM() interface{} {
	if s == nil {
		return nil
	}
	return s.orgvm
}

// Session implements SessionVM.
func (s *sessionvm) Session() Session {
	if s == nil {
		return nil
	}
	return s.ssn
}

// Set implements SessionVM.
func (s *sessionvm) Set(name string, value interface{}) (err error) {
	if s == nil || name == "" {
		return
	}
	if vmset, _ := s.orgvm.(interface {
		Set(string, interface{}) error
	}); vmset != nil {
		err = vmset.Set(name, value)
	}
	return
}

func InvokeVM(orgvm interface{}, ssn Session) (ssnvm SessionVM) {
	if ssnref, _ := ssn.(*session); ssnref != nil {
		ssnvm = &sessionvm{orgvm: orgvm, ssn: ssn}
	}
	return
}
