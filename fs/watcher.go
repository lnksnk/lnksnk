package fs

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type watcher struct {
	fswtchr    *fsnotify.Watcher
	ctxcnl     context.CancelFunc
	ctx        context.Context
	EventClose func(wthr *watcher)
	wtchng     *sync.Map
}

func (wtchr *watcher) Add(path string) bool {
	if wtchr == nil || path == "" {
		return false
	}
	if fswtchr := wtchr.fswtchr; fswtchr != nil {
		return fswtchr.Add(path) == nil
	}
	return false
}

func (wtchr *watcher) Close() (err error) {
	if wtchr == nil {
		return
	}
	watchers.CompareAndDelete(wtchr, wtchr)
	fswtchr := wtchr.fswtchr
	wtchr.fswtchr = nil
	ctxcncl := wtchr.ctxcnl
	wtchr.ctxcnl = nil
	ctxcncl()
	if fswtchr != nil {
		fswtchr.Close()
	}
	evtclose := wtchr.EventClose
	wtchr.EventClose = nil
	if evtclose != nil {
		evtclose(wtchr)
	}
	return
}

func invokeWatcher(evtcreate func(string), evtrename func(string), evtremove func(string), evtwrite func(string), evterr func(error)) (wtchr *watcher) {
	if fswtchr, _ := fsnotify.NewWatcher(); fswtchr != nil {
		ctx, ctxcnl := context.WithCancel(context.Background())
		wtchr = &watcher{fswtchr: fswtchr, ctx: ctx, ctxcnl: ctxcnl}
		watchers.Store(wtchr, wtchr)
		wchng := &sync.Map{}
		wtchr.wtchng = wchng
		var waitFor = 100 * time.Millisecond

		var processEvent = func(evt fsnotify.Event) {
			defer func() {
				tmrv, _ := wchng.Load(evt.Name)
				if t, _ := tmrv.(*time.Timer); t != nil {
					t.Stop()
				}
				wchng.Delete(evt.String())
			}()
			if evt.Has(fsnotify.Create) && evtcreate != nil {
				evtcreate(evt.Name)
			}
			if evt.Has(fsnotify.Write) && evtwrite != nil {
				evtwrite(evt.Name)
			}
			if evt.Has(fsnotify.Rename) && evtrename != nil {
				evtrename(evt.Name)
			}
			if evt.Has(fsnotify.Remove) && evtremove != nil {
				evtremove(evt.Name)
			}
		}
		go func() {
			for {
				select {
				case evt, evtok := <-fswtchr.Events:
					evt.Name = strings.Replace(evt.Name, "\\", "/", -1)
					if evt.Has(fsnotify.Create) || evt.Has(fsnotify.Write) {
						var tmr *time.Timer
						tmrv, hstmr := wchng.Load(evt.Name)
						if !hstmr && evtok {
							t := time.AfterFunc(math.MaxInt64, func() { processEvent(evt) })
							t.Stop()
							tmr = t
							wchng.Store(evt.Name, t)
						} else if evtok {
							tmr, _ = tmrv.(*time.Timer)
						}
						if tmr != nil {
							if evtok {
								tmr.Reset(waitFor)
							} else {
								tmr.Stop()
							}
						}
					} else if evtok {
						go processEvent(evt)
					}
				case err := <-fswtchr.Errors:
					if evterr != nil {
						go func(fserr error) {

						}(err)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	return
}

var watchers = &sync.Map{}
