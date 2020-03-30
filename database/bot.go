package database

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/jackc/pgconn"
	"io"
)

func (db *Db) SetChatState(chat_id int64, u_id int, state string, text string) (pgconn.CommandTag, error) {
	keystr := fmt.Sprintf("%d_%d", chat_id, u_id)
	h := md5.New()
	io.WriteString(h, keystr)
	key := fmt.Sprintf("%x", h.Sum(nil))
	r, err := db.pool.Exec(context.Background(), `INSERT INTO botstate (chatkey, chat_id, u_id, state, text) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (chatkey) DO UPDATE SET state=$4, text=$5;`, key, chat_id, u_id, state, text)
	return r, err
}

func (db *Db) GetChatState(chat_id int64, u_id int) (string, string, error) {
	var state, text string
	keystr := fmt.Sprintf("%d_%d", chat_id, u_id)
	h := md5.New()
	io.WriteString(h, keystr)
	key := fmt.Sprintf("%x", h.Sum(nil))

	err := db.pool.QueryRow(context.Background(), "SELECT state, text FROM botstate WHERE chatkey=$1 ", key).Scan(&state, &text)
	if err != nil {
		return "", "", err
	}
	return state, text, nil
}


func (db *Db) SetSession(chat_id int64, u_id int, data []byte) (pgconn.CommandTag, error) {
	keystr := fmt.Sprintf("%d_%d", chat_id, u_id)
	h := md5.New()
	io.WriteString(h, keystr)
	key := fmt.Sprintf("%x", h.Sum(nil))
	r, err := db.pool.Exec(context.Background(), `INSERT INTO session (chatkey, chat_id, u_id, data) VALUES ($1, $2, $3, $4) ON CONFLICT (chatkey) DO UPDATE SET data=$4;`, key, chat_id, u_id, data)
	return r, err
}

func (db *Db) GetSession(chat_id int64, u_id int) ([]byte, error) {
	var data []byte
	keystr := fmt.Sprintf("%d_%d", chat_id, u_id)
	h := md5.New()
	io.WriteString(h, keystr)
	key := fmt.Sprintf("%x", h.Sum(nil))

	err := db.pool.QueryRow(context.Background(), "SELECT data FROM session WHERE chatkey=$1 ", key).Scan(&data)
	if err != nil {
		return []byte(""), err
	}
	return data, nil
}


func (db *Db) DelSession(chat_id int64, u_id int) (pgconn.CommandTag, error) {
	r, err := db.pool.Exec(context.Background(), `DELETE FROM session where chat_id=$1 and u_id=$2;`, chat_id, u_id)
	return r, err
}
