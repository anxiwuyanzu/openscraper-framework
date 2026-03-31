package dot

import (
	"context"
	"database/sql"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/dnscache"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"net"
	"strings"
	//_ "github.com/lib/pq"
)

func NewSqlClient(addr string) *sql.DB {
	return NewSqlClientWithConfig(addr, nil)
}

func NewSqlClientWithConfig(addr string, cfg *SqlConfig) *sql.DB {
	if cfg == nil {
		cfg = &Conf().Sql
	}
	if u, ok := cfg.Others[addr]; ok {
		addr = u
	}

	driverName := getDriverName(addr)
	if driverName == "mysql" {
		dialer := dnscache.DialFunc(nil)
		mysql.RegisterDialContext("tcp", func(ctx context.Context, addr string) (net.Conn, error) {
			return dialer(ctx, "tcp", addr)
		})
	}

	conn, err := sql.Open(driverName, addr)
	if err != nil {
		Logger().WithError(err).Panicf("Connect to mysql fail")
		return nil
	}

	if cfg.MaxOpenConns > 0 {
		conn.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	err = conn.Ping()
	if err != nil {
		Logger().WithError(err).Panicf("Ping to mysql fail")
		return nil
	}
	return conn
}

func NewSqlxClient(addr string) *sqlx.DB {
	return NewSqlxClientWithConfig(addr, nil)
}

func NewSqlxClientWithConfig(addr string, cfg *SqlConfig) *sqlx.DB {
	if cfg == nil {
		cfg = &Conf().Sql
	}
	if u, ok := cfg.Others[addr]; ok {
		addr = u
	}

	driverName := getDriverName(addr)
	if driverName == "mysql" {
		dialer := dnscache.DialFunc(nil)
		mysql.RegisterDialContext("tcp", func(ctx context.Context, addr string) (net.Conn, error) {
			return dialer(ctx, "tcp", addr)
		})
	}

	conn, err := sql.Open(driverName, addr)
	if err != nil {
		Logger().WithError(err).Panicf("Connect to mysql fail")
		return nil
	}

	if cfg.MaxOpenConns > 0 {
		conn.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	err = conn.Ping()
	if err != nil {
		Logger().WithError(err).Panicf("Ping to mysql fail")
		return nil
	}
	return sqlx.NewDb(conn, "mysql")
}

func getDriverName(dsn string) string {
	pos := strings.Index(dsn, "://")
	if pos <= 0 {
		return "mysql"
	}
	return dsn[:pos]
}
