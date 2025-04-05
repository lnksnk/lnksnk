package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
)

func Open(datasource string) (db *sql.DB, err error) {
	db, err = sql.Open("pgx/v5", datasource)
	return
}

func OpenPool(datasource string) (db *sql.DB, err error) {
	if !strings.Contains(datasource, "pool_max_conn_lifetime=") {
		datasource += " pool_max_conn_lifetime=10s pool_health_check_period=20s"
	}
	if !strings.Contains(datasource, "pool_health_check_period=") {
		datasource += " pool_health_check_period=20s"
	}
	if !strings.Contains(datasource, "sslmode=") {
		datasource += " sslmode=disable"
	}
	pxcnfg, pxerr := pgxpool.ParseConfig(datasource)
	if pxerr != nil {
		err = pxerr
		return
	}
	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, pxcnfg)
	if err != nil {
		return nil, errors.Wrap(err, "create db conn pool")
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.Wrap(err, "create db conn pool")
	}
	db = stdlib.OpenDBFromPool(pool)
	//db, err = sql.Open("pgx/v5", datasource)
	return
}

func ParseSqlParam(totalArgs int) (s string) {
	return "$" + fmt.Sprintf("%d", totalArgs+1)
}

func InvokeDB(datasource string, a ...interface{}) (db *sql.DB, err error) {
	if db, err = OpenPool(datasource); err == nil && db != nil {
		//db.SetMaxOpenConns(1000)
	}
	return
}
