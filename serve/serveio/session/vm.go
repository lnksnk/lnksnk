package sessioning

type SessionVM interface {
	VM() interface{}
	Session() Session
	Set(string, interface{}) error
}

type sessionvm struct {
	orgvm interface{}
	ssn   Session
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
