package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lnksnk/lnksnk/dbms"
	"github.com/lnksnk/lnksnk/dbms/dlv"
	"github.com/lnksnk/lnksnk/dbms/mssql"
	"github.com/lnksnk/lnksnk/dbms/ora"
	"github.com/lnksnk/lnksnk/dbms/postgres"
	"github.com/lnksnk/lnksnk/dbms/sqlite"
	"github.com/lnksnk/lnksnk/es"
	"github.com/lnksnk/lnksnk/fonts"
	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ui"

	"github.com/lnksnk/lnksnk/fs/active"
	"github.com/lnksnk/lnksnk/ioext"
	"github.com/lnksnk/lnksnk/listen"
	"github.com/lnksnk/lnksnk/mimes"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

func main() {
	chn := make(chan bool, 1)
	var mltyfsys = active.AciveFileSystem()
	mltyfsys.CacheExtensions(".html", ".js", ".css", ".svg", ".woff2", ".woff", ".ttf", ".eot", ".sql")
	mltyfsys.DefaultExtensions(".html", ".js", ".json", ".css")
	mltyfsys.ActiveExtensions(".html", ".js", ".svg", ".json", ".xml", ".sql")
	mltyfsys.Map("/embedding")
	mltyfsys.Map("/", "C:/GitHub/lnksnk.github.io", true)
	mltyfsys.Map("/etl", "C:/GitHub/lnketl", true)
	mltyfsys.Map("/media", "C:/movies", true)
	mltyfsys.Set("/embedding/embed.html", `<h2><@print("embed");@></h2>`)
	mltyfsys.Map("/datafiles", "C:/projects/datafiles", true)
	glbldbms := dbms.NewDBMS(mltyfsys)
	dbdrivers := map[string][]interface{}{}
	dbdrivers["postgres"] = []interface{}{dbms.InvokeDBFunc(postgres.InvokeDB), dbms.ParseSqlArgFunc(postgres.ParseSqlParam)}
	dbdrivers["mssql"] = []interface{}{dbms.InvokeDBFunc(mssql.InvokeDB), dbms.ParseSqlArgFunc(mssql.ParseSqlParam)}
	dbdrivers["sqlserver"] = []interface{}{dbms.InvokeDBFunc(mssql.InvokeDB), dbms.ParseSqlArgFunc(mssql.ParseSqlParam)}
	dbdrivers["azuresql"] = []interface{}{dbms.InvokeDBFunc(mssql.InvokeDBAzure), dbms.ParseSqlArgFunc(mssql.ParseSqlParam)}
	dbdrivers["oracle"] = []interface{}{dbms.InvokeDBFunc(ora.InvokeDB), dbms.ParseSqlArgFunc(ora.ParseSqlParam)}
	dbdrivers["sqlite"] = []interface{}{dbms.InvokeDBFunc(sqlite.InvokeDB), dbms.ParseSqlArgFunc(sqlite.ParseSqlParam)}
	dbdrivers["csv"] = []interface{}{dbms.InvokeDBFunc(dlv.InvokeCSVDB), dbms.ParseSqlArgFunc(dlv.ParseSqlParam)}
	dbdrivers["dlv"] = []interface{}{dbms.InvokeDBFunc(dlv.InvokeDLVDB), dbms.ParseSqlArgFunc(dlv.ParseSqlParam)}

	glbldbms.Drivers().DefaultInvokable(func(driver string) (InvokeDB dbms.InvokeDBFunc, ParseSqlParam dbms.ParseSqlArgFunc) {
		if dbdvrapi, dbdrvapik := dbdrivers[driver]; dbdrvapik {
			InvokeDB, _ = dbdvrapi[0].(dbms.InvokeDBFunc)
			ParseSqlParam, _ = dbdvrapi[1].(dbms.ParseSqlArgFunc)
		}
		return
	})
	glbldbms.Drivers().Register("postgres")
	glbldbms.Drivers().Register("mssql")
	glbldbms.Drivers().Register("sqlserver")
	glbldbms.Drivers().Register("azuresql")
	glbldbms.Drivers().Register("oracle")
	glbldbms.Drivers().Register("sqlite")
	glbldbms.Drivers().Register("csv", mltyfsys)
	if mltyfsys.Exist("/embedding/embed.html") {
		glbldbms.Connections().Register("datafiles", "csv", "/datafiles")
		fmt.Println(time.Now())
		if rdr, rdrerr := glbldbms.Query("datafiles", "OMNI Data- RMasterfile_DAT CREDIT 07-08-2022.txt", map[string]interface{}{"ColDelim": "\t", "Trim": true}); rdrerr == nil {
			defer rdr.Close()
			for rc := range rdr.Records() {
				if rc.First() {
					fmt.Println(rc.Columns())
				}
				if rc.Last() {
					fmt.Println(rc.Data())
					fmt.Println(rc.RowNR())
				}
			}
		}
		fmt.Println(time.Now())
	}
	glbldbms.Connections().Register("lnksnk_etl", "postgres", "user=lnksnk_etl password=6@N61ng0 host=localhost port=7654 database=lnksnk_etl")
	fonts.EmbedFonts(mltyfsys)
	ui.EmbedUiJS(mltyfsys)
	var hndlr http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var inout = serveio.NewReaderWriter(serveio.NewReader(r), serveio.NewWriter(w))
		defer inout.Close()
		in, out := inout.Reader(), inout.Writer()
		path := in.Path()

		rqfi := mltyfsys.StatContext(in.Context(), path)
		if rqfi == nil {
			return
		}
		if cls, _ := rqfi.(io.Closer); cls != nil {
			defer cls.Close()
		}
		if rqsize := rqfi.Size(); rqsize > 0 {
			mimetype, texttype, media := mimes.FindMimeType(rqfi.Ext())
			if texttype {
				out.Header().Set("Expires", time.Now().Format(http.TimeFormat))
			}
			if texttype || strings.Contains(mimetype, "text/plain") {
				mimetype += "; charset=utf-8"
			}
			out.Header().Set("Content-type", mimetype)
			actv := texttype
			if !actv && rqfi.Active() {
				actv = true
			}
			if actv {
				var vm = es.New()
				var runvm = func(prgm interface{}, prgout io.Writer) {
					if eprg, _ := prgm.(*es.Program); eprg != nil {
						vm.Set("print", func(a ...interface{}) { ioext.Fprint(prgout, a...) })
						vm.Set("println", func(a ...interface{}) { ioext.Fprintln(prgout, a...) })
						rslt, err := vm.RunProgram(eprg)
						if err != nil {
							out.Print("err:" + err.Error())
						} else {
							if rslt != nil {
								if exp := rslt.Export(); exp != nil {
									out.Print(exp)
								}
							}
						}
					}
				}
				var dbhndl = glbldbms.Handler(in.Params(), func(dbfsys fs.MultiFileSystem, dbfi fs.FileInfo, qryout io.Writer) {
					active.ProcessActiveFile(
						dbfsys,
						dbfi,
						qryout,
						nil,
						runvm)
					vm.Set("print", out.Print)
					vm.Set("print", out.Println)
				})
				defer dbhndl.Close()
				var session = map[string]interface{}{"db": dbhndl, "fs": mltyfsys}
				vm.SetFieldNameMapper(es.NewFieldMapper(es.UncapFieldNameMapper()))
				vm.Set("$", session)
				defer func() {
					vm = nil
				}()
				if rqfi = active.ProcessActiveFile(
					mltyfsys,
					rqfi,
					out,
					nil,
					runvm); rqfi != nil {
					out.Print(rqfi.Reader(in.Context()))
				}
				return
			}

			if !actv || (media && rqfi.Media()) {
				rdr := rqfi.Reader()
				if rdrsk, rangeOffset, rangeType := rdr.(io.ReadSeeker), in.RangeOffset(), in.RangeType(); rdrsk != nil && rangeOffset > -1 && rangeType == "bytes" {
					rdrsk.Seek(rangeOffset, 0)
					maxoffset := int64(0)
					maxlen := int64(0)
					if maxoffset = rangeOffset + (rqsize - rangeOffset); maxoffset > 0 {
						maxlen = maxoffset - rangeOffset
						maxoffset--
					}

					if maxoffset < rangeOffset {
						maxoffset = rangeOffset
						maxlen = 0
					}
					if maxlen > 1024*1024 {
						maxlen = 1024 * 1024
						maxoffset = rangeOffset + (maxlen - 1)
					}
					contentrange := fmt.Sprintf("%s %d-%d/%d", in.RangeType(), rangeOffset, maxoffset, rqsize)
					if out != nil {
						out.Header().Set("Content-Range", contentrange)
						out.Header().Set("Content-Length", fmt.Sprintf("%d", maxlen))
					}
					rdr = io.LimitReader(rdr, maxlen)
					out.MaxWriteSize(maxlen)
					if out != nil {
						out.WriteHeader(206)
					}
				} else {
					if out != nil {
						out.Header().Set("Accept-Ranges", "bytes")
						out.Header().Set("Content-Length", fmt.Sprintf("%d", rqsize))
					}
					out.MaxWriteSize(rqsize)
				}
				out.BPrint(rdr)
				return
			}
		}
	})

	listen.Serve("tcp", ":1090", hndlr)
	//http.Serve(ln, h2c.NewHandler(hndlr, &http2.Server{}))
	<-chn
}
