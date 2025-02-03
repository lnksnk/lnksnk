package listen

import (
	"net"
	"sync"
)

type listener struct {
	lstnsrs *sync.Map
	hndlrs  *sync.Map
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
