package database
import (
    "github.com/jackc/pgconn"
    "context"
)

func (db *Db) AddRanking(chat_id int64, message_id int, u_id int, ranking int) (pgconn.CommandTag, error)  {
    r, err := db.pool.Exec(context.Background(), `insert into ranking (chat_id, message_id, u_id, ranking) values ($1, $2, $3, $4);`, chat_id, message_id, u_id, ranking);
    return r, err
}
