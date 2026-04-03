package client

import (
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/XSAM/otelsql"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	_ "modernc.org/sqlite"

	"arktie.org/ent"
	"arktie.org/internal/data"
)

type Database struct {
	Ent *ent.Client
	RDB *redis.Client
}

func NewDatabase(cfg *data.Config) (database *Database, err error) {
	database = &Database{}

	if err = database.initSQLClient(cfg); err != nil {
		return
	}

	if err = database.initRedisClient(cfg); err != nil {
		return
	}

	return
}

func (c *Database) initSQLClient(cfg *data.Config) error {
	db, err := otelsql.Open(
		cfg.Service.Database.SQL.Driver, cfg.Service.Database.SQL.DSN,
		otelsql.WithAttributes(semconv.DBSystemNameKey.String(cfg.Service.Database.SQL.Driver)),
	)
	if err != nil {
		return err
	}

	c.Ent = ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	return db.Ping()
}

func (c *Database) initRedisClient(cfg *data.Config) (err error) {
	c.RDB = redis.NewClient(&redis.Options{
		Addr:     cfg.Service.Database.Redis.Addr,
		Password: cfg.Service.Database.Redis.Password,
	})

	return redisotel.InstrumentTracing(c.RDB)
}
