package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/virushuo/brikobot/session"
	"github.com/virushuo/brikobot/util"
	"fmt"
	"strings"
	//"regexp"
	"github.com/virushuo/brikobot/database"
    "github.com/vmihailenco/msgpack/v4"
)

type InputMessage struct {
	Text    string
	Lang    string
	SourceURL string
}

type OutputMessage struct{
	Text    string
	Lang    string
	SourceURL string
    Translation []string
    LangList    []string
}

type Session struct {
    Id int64
    Input InputMessage
    Output OutputMessage
}

func (inmsg *InputMessage) verifyData() (bool,string) {
    if inmsg.Text == ""{
        return false, "please input the content"
    } else if inmsg.Lang == "" {
        return false, "which is the language of input?"
    } else if inmsg.SourceURL == "" {
        return false, "please input the source language"
    }
    return true,""
}

func ProcessUpdateMessageChat(bot *tgbotapi.BotAPI, update *tgbotapi.Update, ch chan session.State, db *database.Db,  u_id int, chat_id int64) string{
    input := update.Message.Text

    // not start with "/"
    sourceURL := "http://"
    text := ""
    lang := ""
    validURL := util.IsURL(strings.TrimSpace(input))
    if validURL == true {
        //fetch?
        sourceURL=strings.TrimSpace(input)
    } else {
        text = strings.TrimSpace(input)
    }
    fmt.Println(input)
    //url / text 
    inmsg := InputMessage{
	    Text: text,
	    Lang: lang,
        SourceURL :sourceURL,
	}
    r,msg := inmsg.verifyData()
    if r==false {
        b, err := msgpack.Marshal(&inmsg)
        fmt.Println(string(b))
        fmt.Println(err)
        fmt.Println(msg)

        commandtag, err := db.SetSession(chat_id, u_id, b)
        fmt.Println(commandtag)
        if err != nil {
            fmt.Println(err)
        }


        var item InputMessage
	    err = msgpack.Unmarshal(b, &item)
        fmt.Println("==========")
        item.Lang="fr"
        fmt.Println(item)
        fmt.Printf("%T\n", item)
        fmt.Println(err)
    }
    //save session and return
    return "ProcessUpdateMessageChat end"
}
