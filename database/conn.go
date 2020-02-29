package database

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Db struct {
	pool *pgxpool.Pool
}

func New(connstr string) (*Db, error) {
	db := new(Db)
	poolConfig, err := pgxpool.ParseConfig(connstr)
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConns = 6
	db.pool, err = pgxpool.ConnectConfig(context.Background(), poolConfig)
	//defer db.pool.Close()
	return db, err
}
