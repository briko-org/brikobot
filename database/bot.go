package database

import (
	"context"
    "crypto/md5"
    "io"
    "fmt"
	"github.com/jackc/pgconn"
)

func (db *Db) SetChatState(chat_id int64, u_id int, state string, text string) (pgconn.CommandTag, error) {
    keystr := fmt.Sprintf("%d_%d", chat_id, u_id );
    h := md5.New()
    io.WriteString(h, keystr)
	key := fmt.Sprintf("%x", h.Sum(nil))
	r, err := db.pool.Exec(context.Background(), `INSERT INTO botstate (chatkey, chat_id, u_id, state, text) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (chatkey) DO UPDATE SET state=$4, text=$5;`, key, chat_id, u_id, state, text)
	return r, err
}

