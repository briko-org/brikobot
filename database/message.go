package database

import (
	"context"
    "bytes"
	"github.com/jackc/pgconn"
)

func (db *Db) AddMessage(chat_id int64, message_id int, u_id int, text string) (pgconn.CommandTag, error) {
    textbytes := bytes.Trim([]byte(text),  "\x00")
    textbytes = bytes.Trim(textbytes,  "\u0000")
	r, err := db.pool.Exec(context.Background(), `insert into message (chat_id, message_id, u_id, text) values ($1, $2, $3, $4);`, chat_id, message_id, u_id, string(textbytes))
	return r, err
}
