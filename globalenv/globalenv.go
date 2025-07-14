package globalenv

import (
	"io"
	"net/http"

	"github.com/lnksnk/lnksnk/globaldbms"
	"github.com/lnksnk/lnksnk/globalfs"
	"github.com/lnksnk/lnksnk/globallisten"
	"github.com/lnksnk/lnksnk/globalsession"
	"github.com/lnksnk/lnksnk/ioext"
)

func LoadEnv(a ...interface{}) {
	var config map[string]interface{}
	ai := 0
	if al := len(a); al > 0 {
		for ai < al {
			if confd, confdk := a[ai].(map[string]interface{}); confdk {
				if len(confd) > 0 {
					if config == nil {
						config = confd
					} else {
						for ck, cv := range confd {
							config[ck] = cv
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
					if config == nil {
						config = cnfd
					} else {
						for ck, cv := range cnfd {
							config[ck] = cv
						}
					}
				}
			}
		}
	}
	if len(config) == 0 {
		return
	}
	var dbcnf map[string]interface{}
	var fsyscnf map[string]interface{}
	var lstncnf map[string]interface{}
	for k, v := range config {
		if k == "dbms" {
			if dbcnfd, dbcnfk := v.(map[string]interface{}); dbcnfk {
				if len(dbcnfd) > 0 {
					if len(dbcnf) == 0 {
						dbcnf = dbcnfd
						continue
					}
					for dk, dv := range dbcnfd {
						dbcnf[dk] = dv
					}
					continue
				}
			}
			continue
		}
		if k == "filesys" {
			if fscnfd, fscnfk := v.(map[string]interface{}); fscnfk {
				if len(fscnfd) > 0 {
					if len(fsyscnf) == 0 {
						fsyscnf = fscnfd
						continue
					}
					for fk, fv := range fscnfd {
						fsyscnf[fk] = fv
					}
					continue
				}
			}
			continue
		}
		if k == "listen" {
			if lstncnfd, lstncnfk := v.(map[string]interface{}); lstncnfk {
				if len(lstncnfd) > 0 {
					if len(lstncnf) == 0 {
						lstncnf = lstncnfd
						continue
					}
					for lk, lv := range lstncnfd {
						lstncnf[lk] = lv
					}
					continue
				}
			}
			continue
		}
	}
	if len(dbcnf) > 0 {
		globaldbms.DBMS.Load(dbcnf)
	}
	if len(fsyscnf) > 0 {
		globalfs.FSYS.Load(fsyscnf)
	}
	if len(lstncnf) > 0 {
		globallisten.LISTEN.Load(lstncnf)
	}
}

func UnloadEnv(a ...interface{}) {

	var unldconf map[string]interface{}
	if al := len(a); al > 0 {
		ai := 0
		for ai < al {
			if mpd, mpk := a[ai].(map[string]interface{}); mpk {
				if len(mpd) > 0 {
					if unldconf == nil {
						unldconf = mpd
					} else {
						for k, v := range mpd {
							unldconf[k] = v
						}
					}
				}
				a = append(a[:ai], a[ai+1:]...)
				continue
			}
			ai++
		}
		if al > 0 {
			in, _ := ioext.NewBuffer(a...).Reader(true).Marshal()
			if mpd, mpk := in.(map[string]interface{}); mpk {
				if len(mpd) > 0 {
					if unldconf == nil {
						unldconf = mpd
					} else {
						for k, v := range mpd {
							unldconf[k] = v
						}
					}
				}
			}
		}
	}
	var dbmsunldcnf map[string]interface{}
	var lstnunldcnf []interface{}
	var fsunldcnf []interface{}
	for k, v := range unldconf {
		if k == "dbms" || k == "listen" || k == "filesys" {
			if mpd, mpk := v.(map[string]interface{}); mpk {
				if len(mpd) > 0 {
					if k == "dbms" {
						if len(dbmsunldcnf) == 0 {
							dbmsunldcnf = mpd
							continue
						}
						for uk, uv := range mpd {
							dbmsunldcnf[uk] = uv
						}
					}
				}
				continue
			}
			if arrs, arrk := v.([]string); arrk {
				if arrk {
					for _, as := range arrs {
						if as != "" {
							if k == "listen" {
								if len(lstnunldcnf) == 0 {
									lstnunldcnf = append(lstnunldcnf, []interface{}{})
								}
								lstnunldcnf[0] = append(lstnunldcnf[0].([]interface{}), as)
								continue
							}
							if k == "filesys" {
								if len(fsunldcnf) == 0 {
									fsunldcnf = append(fsunldcnf, []interface{}{})
								}
								fsunldcnf[0] = append(fsunldcnf[0].([]interface{}), as)
								continue
							}
							break
						}
					}
				}
				continue
			}
			if arri, arrik := v.([]interface{}); arrik {
				if arrik {
					if len(arri) > 0 {
						if k == "listen" {
							if len(lstnunldcnf) == 0 {
								lstnunldcnf = append(lstnunldcnf, []interface{}{})
							}
							lstnunldcnf[0] = append(lstnunldcnf[0].([]interface{}), arri...)
							continue
						}
						if k == "filesys" {
							if len(fsunldcnf) == 0 {
								fsunldcnf = append(fsunldcnf, []interface{}{})
							}
							fsunldcnf[0] = append(fsunldcnf[0].([]interface{}), arri...)
						}
					}
				}
				continue
			}
			continue
		}
	}
	globallisten.LISTEN.Unload(lstnunldcnf...)
	globaldbms.DBMS.Unload(dbmsunldcnf)
	globalfs.FSYS.Unload(fsunldcnf...)
}

func EnvServe(path string, in io.Reader, out io.Writer) {
	globalsession.IOSessionHandler(path, in, out)
}

func EnvServeHTTP(w http.ResponseWriter, r *http.Request) {
	globalsession.HTTPSessionHandler(w, r)
}
