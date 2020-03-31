package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/virushuo/brikobot/session"
	"github.com/virushuo/brikobot/util"
    "github.com/google/uuid"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"bytes"
	"fmt"
	"strconv"
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
    Chat_id int64
	Text    string
	Lang    string
	SourceURL string
    Translation map[string]string
    //Translation []string
    //LangList    []string
}


type Session struct {
    Id uuid.UUID
    State int
    Input InputMessage
    Output OutputMessage
}

type requestMsg struct {
	MsgType       string   `json:"msgType"`
	MsgID         string   `json:"msgID"`
	SourceLang    string   `json:"sourceLang"`
	RequestLang   []string `json:"requestLang"`
	SourceContent string   `json:"sourceContent"`
}

type responseMsg struct {
	MsgFlag            string            `json:"msgFlag"`
	MsgID              string            `json:"msgID"`
	MsgIDRespondTo     string            `json:"msgIDRespondTo"`
	MsgType            string            `json:"msgType"`
	TranslationResults map[string]string `json:"translationResults"`
}

func makeReplyKeyboard(lang_list []string, submit bool) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	var row []tgbotapi.InlineKeyboardButton
	for _, value := range lang_list {
	    button := tgbotapi.NewInlineKeyboardButtonData(value, "SETLANG_"+value)
	    row = append(row, button)
	}
	keyboard = append(keyboard, row)

    if submit ==true {
	    var submitrow []tgbotapi.InlineKeyboardButton
	    button := tgbotapi.NewInlineKeyboardButtonData("OK, translate!", "SUBMIT_MSG")
	    submitrow = append(submitrow, button)
	    button = tgbotapi.NewInlineKeyboardButtonData("No, Cancel.", "CANCEL_MSG")
	    submitrow = append(submitrow, button)
	    keyboard = append(keyboard, submitrow)
    }

	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}


func (inmsg *InputMessage) verifyData(chat_id int64) (bool, tgbotapi.MessageConfig) {
    lang_list := []string {"zh", "en", "fr", "jp"}
    if inmsg.Text == ""{
        return false, tgbotapi.NewMessage(chat_id, "please input the content")
    } else if inmsg.Lang == "" {
		responseMsg := tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID: chat_id,
				ReplyToMessageID: 0,
			},
			Text: "please select the language",
			//ParseMode: "Markdown",
			DisableWebPagePreview: false,
		}
		responseMsg.ReplyMarkup = makeReplyKeyboard(lang_list, false)
        return false, responseMsg
    } else if inmsg.SourceURL == "" {
        return false, tgbotapi.NewMessage(chat_id, "please input the source url")
    }

	responseMsg := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID: chat_id,
			ReplyToMessageID: 0,
		},
        Text: fmt.Sprintf("Input: [%s]%s source:%s", inmsg.Lang, inmsg.Text, inmsg.SourceURL) ,
		//ParseMode: "Markdown",
		DisableWebPagePreview: false,
	}
	responseMsg.ReplyMarkup = makeReplyKeyboard(lang_list, true)
    return true, responseMsg

    //return true, tgbotapi.NewMessage(chat_id, fmt.Sprintf("date is ok to submit to the briko AI: %s %s %s", inmsg.Lang, inmsg.Text, inmsg.SourceURL))
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

func ProcessTranslationResult(outputmsg *OutputMessage){
}

func ProcessUpdateCmdMessage(bot *tgbotapi.BotAPI, cmd string, query string, ch chan OutputMessage, db *database.Db, message_id int, u_id int, chat_id int64) string{
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
    } else if cmd =="SUBMIT"{

        fmt.Println("=======SUBMIT:")
        fmt.Println(BRIKO_API)
        //requestBriko(APIURL string, lang_list []string, lang_correlation map[string]string, msgId int,inmsg InputMessage, ch chan OutputMessage)
        //var ch chan OutputMessage = make(chan OutputMessage)
        go requestBriko(BRIKO_API, REQUEST_LANG_LIST , LANG_CORRELATION, message_id, chat_id, currentSession.Input, ch)
        //call api
    } else if cmd =="CANCEL"{
        _, err := db.DelSession(chat_id, u_id)
        if err == nil {
            responsemsg := tgbotapi.NewMessage(chat_id, "Cancelled, please input the content.")
	        bot.Send(responsemsg)
        } else {
            fmt.Println(err)
        }
        //delete session 
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
    if err == nil {
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

func requestBriko(APIURL string, lang_list []string, lang_correlation map[string]string, msgId int,chat_id int64, inmsg InputMessage, ch chan OutputMessage) {
	data := &requestMsg{
		MsgType:    "Translation",
		MsgID:      strconv.Itoa(msgId),
		SourceLang: inmsg.Lang,
	}

    if lang_correlation[data.SourceLang] !=""{
        data.SourceLang = lang_correlation[data.SourceLang]
    }

	requestLang := []string{}
	for _, value := range lang_list {
		if value != data.SourceLang {
			requestLang = append(requestLang, value)
		}
	}
    data.SourceContent = inmsg.Text

	data.RequestLang = requestLang
	output, _ := json.Marshal(data)
    fmt.Println(string(output))
	resp, _ := http.Post(APIURL, "application/json", bytes.NewBuffer(output))
    bodyBytes, err1 := ioutil.ReadAll(resp.Body)
    if err1 != nil {
        fmt.Println(err1)
        //TODO: send the error msg to bot
    }
    bodyString := string(bodyBytes)
    d := json.NewDecoder(strings.NewReader(bodyString))
	rmsg := &responseMsg{}
	err := d.Decode(rmsg)

	if err != nil {
	} else {
	    if rmsg.MsgFlag == "success" {
	        output := &OutputMessage{
                Chat_id: chat_id,
                Text:    inmsg.Text,
                Lang:    inmsg.Lang,
                SourceURL: inmsg.SourceURL,
                Translation: rmsg.TranslationResults,
	        }
            ch <- *output
            //type OutputMessage struct{
            //    Text    string
            //    Lang    string
            //    SourceURL string
            //    Translation []string
            //    LangList    []string
            //}

            //lang_content := fmt.Sprintf("[%s] %s %s", data.SourceLang, data.SourceContent, SourceURL)
	    	//lang_list_str := fmt.Sprintf("[%s]", data.SourceLang)
	    	//translation := rmsg.TranslationResults
	    	//for key, value := range translation {
	    	//	lang_content = lang_content + fmt.Sprintf("\n\n[%s] %s", key, value)
	    	//	lang_list_str = lang_list_str + fmt.Sprintf("[%s]", key)
	    	//}
	    	//ch <- State{"TRANSLATE", fmt.Sprintf("%s:%s", lang_list_str, lang_content), stat.U_id, stat.Chat_id}
	    } else {
            fmt.Println(rmsg.MsgFlag) //TODO: send the error msg to bot
	    }
    }
    //fmt.Println(err)
    //fmt.Println(rmsg)
	//regex := *regexp.MustCompile(`\[([A-Za-z]{2})\]`)
	//res := regex.FindStringSubmatch(stat.Text)
	//if len(res) > 1 {
	//	data.SourceLang = strings.ToLower(res[1])


    //    split_list := strings.Split(stat.Text, " ")
    //    last_str := split_list[len(split_list)-1]
    //    validURL := util.IsURL(last_str)
    //    end_pos := len(last_str)-1
    //    if validURL == true {
    //        end_pos = len(stat.Text) - len(last_str)
    //    }




    //    bodyBytes, err1 := ioutil.ReadAll(resp.Body)
    //    if err1 != nil {
    //        fmt.Println(err1)
    //        //TODO: send the error msg to bot
    //    }
    //    bodyString := string(bodyBytes)
    //    fmt.Println("======bodyString")
    //    fmt.Println(bodyString)
    //    fmt.Println(err1)
    //    d := json.NewDecoder(strings.NewReader(bodyString))
	//	rmsg := &responseMsg{}
	//	err := d.Decode(rmsg)
	//	if err != nil {
    //        fmt.Println("===responseMsg:");
	//		fmt.Println(err) //TODO: send the error msg to bot
	//	} else {
	//		if rmsg.MsgFlag == "success" {
    //            lang_content := fmt.Sprintf("[%s] %s %s", data.SourceLang, data.SourceContent, SourceURL)
	//			lang_list_str := fmt.Sprintf("[%s]", data.SourceLang)
	//			translation := rmsg.TranslationResults
	//			for key, value := range translation {
	//				lang_content = lang_content + fmt.Sprintf("\n\n[%s] %s", key, value)
	//				lang_list_str = lang_list_str + fmt.Sprintf("[%s]", key)
	//			}
	//			ch <- State{"TRANSLATE", fmt.Sprintf("%s:%s", lang_list_str, lang_content), stat.U_id, stat.Chat_id}
	//		} else {
	//			fmt.Println(rmsg.MsgFlag) //TODO: send the error msg to bot
	//		}
	//	}
	//} else {
	//	fmt.Println("no language tag")
	//}
}

