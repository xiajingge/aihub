package dependencies

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/xiajignge/aihub/config"
	"github.com/xiajignge/aihub/internal/ent"
	"github.com/xiajignge/aihub/internal/ent/migrate"
	_ "github.com/xiajignge/aihub/internal/ent/runtime"

	_ "github.com/go-sql-driver/mysql"
)

func NewEntClient(cfg config.DBConfig) *ent.Client {
	var opts []ent.Option
	if cfg.Debug {
		opts = append(opts, ent.Debug())
	}

	var (
		sqlDB     *sql.DB
		dbDialect string
		err       error
	)

	switch cfg.Dialect {
	case "postgres", "pgx", "postgresdb", "pg", "postgresql":
		sqlDB, err = sql.Open("pgx", cfg.DSN)
		if err != nil {
			panic(err)
		}

		dbDialect = dialect.Postgres
	case "sqlite3", "sqlite":
		sqlDB, err = sql.Open("sqlite3", cfg.DSN)
		if err != nil {
			panic(err)
		}

		dbDialect = dialect.SQLite
	case "mysql", "mariadb":
		sqlDB, err = sql.Open("mysql", cfg.DSN)
		if err != nil {
			panic(err)
		}

		dbDialect = dialect.MySQL
	default:
		panic(fmt.Errorf("invalid dialect: %s", cfg.Dialect))
	}

	// 设置连接池
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	drv := entsql.OpenDB(dbDialect, sqlDB)
	opts = append(opts, ent.Driver(drv))
	client := ent.NewClient(opts...)

	err = client.Schema.Create(
		context.Background(),
		migrate.WithGlobalUniqueID(false),
		migrate.WithForeignKeys(false),
	)
	if err != nil {
		panic(err)
	}

	return client
}
