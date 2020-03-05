package database

import (
	"context"
	"github.com/jackc/pgconn"
)

func (db *Db) AddRanking(chat_id int64, message_id int, u_id int, lang string, ranking int) (pgconn.CommandTag, error) {
	r, err := db.pool.Exec(context.Background(), `insert into ranking (chat_id, message_id, u_id, lang, ranking) values ($1, $2, $3, $4, $5);`, chat_id, message_id, u_id, lang, ranking)
	return r, err
}
