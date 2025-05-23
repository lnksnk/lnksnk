package exec

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/lnksnk/lnksnk/database"
	"github.com/lnksnk/lnksnk/database/dbserve"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

var aliascmdexec dbserve.AliasCommandFunc = func(alias, path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	var qryarr []interface{} = nil
	var errorsfnd []interface{} = nil
	if httpr := r.HttpR(); httpr != nil {
		if cnttype := httpr.Header.Get("Content-Type"); strings.Contains(cnttype, "application/json") {
			if bdy := httpr.Body; bdy != nil {
				var qryref interface{} = nil

				if err = json.NewDecoder(bdy).Decode(&qryref); err == nil {
					if qryarrd, _ := qryref.([]interface{}); len(qryarrd) > 0 {
						qryarr = append(qryarr, qryarrd...)
						dbhnl.Query(alias, qryarr...)
					}
					if qrymp, _ := qryref.(map[string]interface{}); len(qrymp) > 0 {
						var qryarr []interface{} = nil
						for qryk, qryv := range qrymp {
							if qryk == "query" {
								qryarr = append(qryarr, qryv)
								delete(qrymp, qryk)
								continue
							}
						}
						if len(qrymp) > 0 {
							qryarr = append(qryarr, qrymp)
						}
					}
				}
			}
		} else {
			if params := dbhnl.Params(); params != nil {
				if (params.Exist("qry") && params.Type("qry") == "std") || (params.Exist("query") && params.Type("query") == "std") {
					for _, qry := range append(params.Get("qry"), params.Get("query")...) {
						qryarr = append(qryarr, qry)
					}
				}
			}
		}
		if path != "" && path[len(path)-1] != '/' && len(qryarr) == 0 {
			pathext := filepath.Ext(path)
			if pathext != "" {
				if pathext != ".sql" {
					path = path[:len(path)-len(pathext)] + ".sql"
				}
			} else {
				path = path + ".sql"
			}
			qryarr = append(qryarr, path)
		}
	}
	if len(qryarr) > 0 {
		var errfound error = nil
		qryarr = append(qryarr, map[string]interface{}{
			"error": func(err error) {
				errfound = err
			},
		})
		errfound = dbhnl.Execute(alias, qryarr...)
		if errfound == nil {
			err = w.Print("{}")
			return
		}

		if errfound != nil {
			errorsfnd = append(errorsfnd, errfound.Error())
			enc := json.NewEncoder(w)
			err = enc.Encode(map[string]interface{}{"err": errorsfnd})
			return
		}
	}
	return
}

func init() {
	dbserve.HandleCommand("exec", aliascmdexec)
}
