package listening

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/lnksnk/lnksnk/ioext"
)

type LISTENING interface {
	Load(...interface{})
	Unload(...interface{})
	Handling() HANDELING
	ioext.IterateMap[string, LISTENER]
	Listen(name string, network string, addr string, handler string, tlsconf ...*tls.Config) error
	Default(network string, addr string, tlsconf ...*tls.Config) error
}

type HANDELING interface {
	ioext.IterateMap[string, http.Handler]
	Register(string, http.Handler)
	Default(http.Handler)
}

type handling struct {
	ioext.IterateMap[string, http.Handler]
}

func (hnldng *handling) Default(handler http.Handler) {
	if hnldng == nil || handler == nil {
		return
	}

	if hnliter := hnldng.IterateMap; hnliter != nil {
		hnliter.Set("default", handler)
	}
}

func (hnldng *handling) Register(name string, handler http.Handler) {
	if hnldng == nil || name == "" {
		return
	}
	if hnliter := hnldng.IterateMap; hnliter != nil {
		hnliter.Set(name, handler)
	}
}

func NewListen() LISTENING {
	return &listening{IterateMap: ioext.MapIterator[string, LISTENER](), hndlng: &handling{IterateMap: ioext.MapIterator[string, http.Handler]()}}
}

type listening struct {
	hndlng HANDELING
	ioext.IterateMap[string, LISTENER]
}

func (lstng *listening) Unload(a ...interface{}) {
	if lstng == nil {
		return
	}
	var lstnconfig []interface{}

	al := len(a)
	if al > 0 {
		var dellstnrs []string
		in, _ := ioext.NewBuffer(a...).Reader(true).Marshal()
		if arrs, arrk := in.([]string); arrk {
			for _, as := range arrs {
				if as != "" {
					lstnconfig = append(lstnconfig, as)
				}
			}
		} else if arri, arrik := in.([]interface{}); arrik {
			if len(arri) > 0 {
				lstnconfig = arri
			}
		}

		if len(lstnconfig) > 0 {
			for _, lstnd := range lstnconfig {
				if as, ask := lstnd.(string); ask && as != "" {
					dellstnrs = append(dellstnrs, as)
				}
			}
		}
		if len(dellstnrs) > 0 {
			events := lstng.Events().(*ioext.MapIterateEvents[string, LISTENER])
			ctx, ctxcnl := context.WithCancel(context.Background())
			events.EventDeleted = func(dltd map[string]LISTENER) {
				for _, lstn := range dltd {
					lstn.Shutdown()
				}
				ctxcnl()
			}
			lstng.Delete(dellstnrs...)
			<-ctx.Done()
		}
		return
	}
	events := lstng.Events().(*ioext.MapIterateEvents[string, LISTENER])
	ctx, ctxcnl := context.WithCancel(context.Background())
	events.EventDisposed = func(dltd map[string]LISTENER) {
		for _, lstn := range dltd {
			lstn.Shutdown()
		}
		ctxcnl()
	}
	lstng.Close()
	<-ctx.Done()
}

func (lstng *listening) Load(a ...interface{}) {
	if lstng == nil || len(a) == 0 {
		return
	}
	var lstnconfig map[string]interface{}
	ai := 0
	if al := len(a); al > 0 {
		for ai < al {
			if confd, confdk := a[ai].(map[string]interface{}); confdk {
				if len(confd) > 0 {
					if lstnconfig == nil {
						lstnconfig = confd
					} else {
						for ck, cv := range confd {
							lstnconfig[ck] = cv
						}
					}
				}
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			ai++
		}
		if al > 0 {
			in, _ := ioext.NewBuffer(a...).Reader(true).Marshal()
			if cnfd, cnfk := in.(map[string]interface{}); cnfk {
				if len(cnfd) > 0 {
					if lstnconfig == nil {
						lstnconfig = cnfd
					} else {
						for ck, cv := range cnfd {
							lstnconfig[ck] = cv
						}
					}
				}
			}
		}
	}
	if len(lstnconfig) == 0 {
		return
	}
	var dfltnetwork = ""
	var dfltaddr = ""
	for lk, lv := range lstnconfig {
		if lk == "default" {
			if dlftmp, _ := lv.(map[string]interface{}); len(dlftmp) > 0 {
				dfltnetwork, _ = dlftmp["network"].(string)
				dfltaddr, _ = dlftmp["addr"].(string)
			}
			continue
		}
	}
	if dfltaddr != "" && dfltnetwork != "" {
		lstng.Default(dfltnetwork, dfltaddr)
	}
}

func (lstng *listening) Default(network string, addr string, tlsconf ...*tls.Config) (err error) {
	if lstng == nil || network == "" || addr == "" {
		return
	}
	lstng.Listen("default", network, addr, "default", tlsconf...)
	return
}

func (lstng *listening) Listen(name string, network string, addr string, handler string, tlsconf ...*tls.Config) (err error) {
	if lstng == nil || network == "" || addr == "" || handler == "" || name == "" {
		return
	}
	if hndlng, lstniter := lstng.hndlng, lstng.IterateMap; hndlng != nil && lstniter != nil {
		if hndlr, hdnlrok := hndlng.Get(handler); hdnlrok {
			lstnr, lstnrk := lstniter.Get(name)
			if lstnrk {
				//TODO reset network and addr
				lstnr.SwapHandler(hndlr)
				return
			}
			if lstnr = nextlistener(network, addr, hndlr); lstnr != nil {
				lstniter.Set(name, lstnr)
			}
		}
	}
	return
}

func (lstng *listening) Handling() HANDELING {
	if lstng == nil {
		return nil
	}
	return lstng.hndlng
}
