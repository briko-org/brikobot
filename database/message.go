package database
import (
    "github.com/jackc/pgconn"
    "context"
)

func (db *Db) AddMessage(chat_id int64, message_id int, u_id int, text string) (pgconn.CommandTag, error)  {
    r, err := db.pool.Exec(context.Background(), `insert into message (chat_id, message_id, u_id, text) values ($1, $2, $3, $4);`, chat_id, message_id, u_id, text);
    return r, err
}
