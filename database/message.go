package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
)

func (db *Db) AddMessage(chat_id int64, message_id int, u_id int, text string) (pgconn.CommandTag, error) {
	fmt.Println("====db")
	fmt.Println(db)
	r, err := db.pool.Exec(context.Background(), `insert into message (chat_id, message_id, u_id, text) values ($1, $2, $3, $4);`, chat_id, message_id, u_id, text)
	return r, err
}
