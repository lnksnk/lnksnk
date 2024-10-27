package listen

import (
	"net"
)

type wraplstn struct {
	addr  net.Addr
	ln    net.Listener
	lstnr *listener
}

func (wrplstn *wraplstn) Addr() net.Addr {
	if wrplstn != nil {
		return wrplstn.addr
	}
	return nil
}

func (wrplstn *wraplstn) Accept() (net.Conn, error) {
	if wrplstn == nil {
		return nil, nil
	}
	if lstnr := wrplstn.lstnr; lstnr != nil {
		return lstnr.Accept()
	}
	return nil, nil
}

func (wrplstn *wraplstn) Close() (err error) {
	if wrplstn != nil {
		ln := wrplstn.ln
		wrplstn.ln = nil
		wrplstn.addr = nil
		wrplstn.lstnr = nil
		if ln != nil {
			err = ln.Close()
		}
	}
	return
}

type listener struct {
	accepts   chan net.Conn
	accepteds chan net.Conn
}

func (lstnr *listener) Start() {
	if lstnr != nil {

		go func() {
			for cn := range lstnr.accepts {
				go func(cnn net.Conn) { lstnr.accepteds <- cnn }(cn)
			}
		}()
	}
}

func (lstnr *listener) Listen(ln net.Listener) net.Listener {
	if lstnr != nil {
		go func() {
			for {
				cn, cnerr := ln.Accept()
				if cnerr != nil {
					func() {
						defer cn.Close()
					}()
					continue
				}
				lstnr.accepts <- cn
			}
		}()
		return &wraplstn{lstnr: lstnr, ln: ln, addr: ln.Addr()}
	}
	return nil
}

func (lstnr *listener) Shutdown() {

}

func (lstnr *listener) Close() (err error) {
	if lstnr != nil {

	}
	return
}

func (lstnr *listener) Addr() net.Addr {
	if lstnr != nil {

	}
	return nil
}

func (lstnr *listener) Accept() (conn net.Conn, err error) {
	if lstnr != nil {
		conn = <-lstnr.accepteds
	}
	return
}
