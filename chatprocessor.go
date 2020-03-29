package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/virushuo/brikobot/session"
	"github.com/virushuo/brikobot/util"
    "github.com/google/uuid"
	"fmt"
	"strings"
	//"regexp"
	"github.com/virushuo/brikobot/database"
    "github.com/vmihailenco/msgpack/v4"
)

const (
        NEED_DATA_INPUT  = 1 << iota
        DATA_OK  = 1 << iota
        SEND_TO_API = 1 << iota
        DONE = 1 << iota
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
    Id uuid.UUID
    State int
    Input InputMessage
    Output OutputMessage
}

func makeReplyKeyboard(lang_list []string) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	var row []tgbotapi.InlineKeyboardButton
	for _, value := range lang_list {
	    button := tgbotapi.NewInlineKeyboardButtonData(value, "SETLANG_"+value)
	    row = append(row, button)
	}
	keyboard = append(keyboard, row)

	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}


func (inmsg *InputMessage) verifyData(chat_id int64) (bool, tgbotapi.MessageConfig) {
    if inmsg.Text == ""{
        return false, tgbotapi.NewMessage(chat_id, "please input the content")
    } else if inmsg.Lang == "" {
        lang_list := []string {"zh", "en", "fr", "jp"}
        //responseMsg := tgbotapi.NewMessage(chat_id, "please input the content")
		//responseMsg.ReplyMarkup = makeReplyKeyboard(lang_list)
		responseMsg := tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID: chat_id,
				ReplyToMessageID: 0,
			},
			Text: "please select the language",
			//ParseMode: "Markdown",
			DisableWebPagePreview: false,
		}
		responseMsg.ReplyMarkup = makeReplyKeyboard(lang_list)
        return false, responseMsg
    } else if inmsg.SourceURL == "" {
        return false, tgbotapi.NewMessage(chat_id, "please input the source url")
    }
    return true, tgbotapi.NewMessage(chat_id, fmt.Sprintf("date is ok to submit to the briko AI: %s %s %s", inmsg.Lang, inmsg.Text, inmsg.SourceURL))
}

func updateSession(input string, session *Session){
    text := strings.TrimSpace(input)

    validURL := util.IsURL(text)
    if validURL == true {
        //fetch?
        //sourceURL=strings.TrimSpace(input)
        session.Input.SourceURL=text
        //return true, ""
    } else {
        //if len(text) == 2 && 
        if len(text) == 2 && inputlangVerify(text) == true{
            session.Input.Lang=text
        } else {
            session.Input.Text=text
        }
        //return true, ""
        //text = strings.TrimSpace(input)
    }
    //return true, ""
}

func inputlangVerify(lang string) bool{
    lang_list := []string {"zh", "en", "fr", "jp"}
    var LANG_CORRELATION map[string]string 
    LANG_CORRELATION = make(map[string]string)
    LANG_CORRELATION["ja"]="jp"
    LANG_CORRELATION["cn"]="zh"
    result := false
    if LANG_CORRELATION[lang] != "" {
        lang = LANG_CORRELATION[lang]
    }
    for _, l := range lang_list {
        if lang == l {
            result = true
        }
    }
    return result
}

func ProcessUpdateCmdMessage(bot *tgbotapi.BotAPI, cmd string, query string, ch chan session.State, db *database.Db,  u_id int, chat_id int64) string{
    var currentSession Session
	data, err := db.GetSession(chat_id, u_id)
    if len(data) == 0 {
        currentSession = Session {
            Id: uuid.New(),
            State: NEED_DATA_INPUT,
            Input: InputMessage {},
            Output: OutputMessage {},
        }
    } else {
        if err == nil {
	        err = msgpack.Unmarshal(data, &currentSession)
        } else {
            fmt.Println(err)
        }
    }

    if cmd =="SETLANG" {
        currentSession.Input.Lang=query
        r, responsemsg :=currentSession.Input.verifyData(chat_id)
        if r == true {
            currentSession.State = DATA_OK
        }

        b, err := msgpack.Marshal(&currentSession)
        if err != nil {
            fmt.Println(err)
        } else {
            _, err := db.SetSession(chat_id, u_id, b)
            if err != nil {
                fmt.Println(err)
            }
	        bot.Send(responsemsg)
        }
    } else {
        re_msg := tgbotapi.NewMessage(chat_id, "")
        re_msg.Text = fmt.Sprintf("Unknown Queryback command: %s", cmd)
    }

    return "ProcessUpdateCmdMessage"
}

func ProcessUpdateMessageChat(bot *tgbotapi.BotAPI, update *tgbotapi.Update, ch chan session.State, db *database.Db,  u_id int, chat_id int64) string{
    input := update.Message.Text

    var currentSession Session
	data, err := db.GetSession(chat_id, u_id)
    if len(data) == 0 {
        currentSession = Session {
            Id: uuid.New(),
            State: NEED_DATA_INPUT,
            Input: InputMessage {},
            Output: OutputMessage {},
        }
    } else {
        if err == nil {
	    err = msgpack.Unmarshal(data, &currentSession)
        } else {
            fmt.Println(err)
        }
    }


    updateSession(input, &currentSession)

    if currentSession.State != DONE {
		switch currentSession.State  {
            case NEED_DATA_INPUT:
                r, responsemsg :=currentSession.Input.verifyData(chat_id)
                if r == true {
                    currentSession.State = DATA_OK
				    //bot.Send(responsemsg)
                }
				bot.Send(responsemsg)

			default:
                fmt.Println("unknown state")
        }
    }

    b, err := msgpack.Marshal(&currentSession)
    fmt.Println(err)
    if err != nil {
        commandtag, err := db.SetSession(chat_id, u_id, b)
        fmt.Println(commandtag)
        if err != nil {
            fmt.Println(err)
        }
    } else {
        fmt.Println(err)
    }

    return "ProcessUpdateMessageChat end"
}
