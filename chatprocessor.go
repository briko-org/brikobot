package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/virushuo/brikobot/util"
	"github.com/virushuo/brikobot/spider"
    "github.com/google/uuid"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"regexp"
	"github.com/virushuo/brikobot/database"
    "github.com/vmihailenco/msgpack/v4"
)

const (
        NEED_DATA_INPUT  = 1 << iota
        FETCH_URL  = 1 << iota
        DATA_OK  = 1 << iota
        SEND_TO_API = 1 << iota
        TRANSLATE_OK  = 1 << iota
        EDIT = 1 << iota
        DONE = 1 << iota
)

type InputMessage struct {
	Text    string
	Lang    string
	SourceURL string
}

type OutputMessage struct{
    Chat_id int64
    U_id int
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
    StateData string
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


func makePublishKeyboard(lang_list []string) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	var row []tgbotapi.InlineKeyboardButton
	for _, value := range lang_list {
        button := tgbotapi.NewInlineKeyboardButtonData("Edit "+value, "EDIT_"+value)
	    row = append(row, button)
	}
	keyboard = append(keyboard, row)

	var publishrow []tgbotapi.InlineKeyboardButton
	button := tgbotapi.NewInlineKeyboardButtonData("OK, PUBLISH!", "PUBLISH_MSG")
	publishrow = append(publishrow, button)
	button = tgbotapi.NewInlineKeyboardButtonData("No, Delete it.", "CANCEL_MSG")
	publishrow = append(publishrow, button)
	keyboard = append(keyboard, publishrow)

	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}

}

func readTranslateOutputMessageChannel(c chan OutputMessage , bot *tgbotapi.BotAPI, db *database.Db) {
	for {
		outputmsg := <-c
        lang_content := fmt.Sprintf("[%s]%s %s", outputmsg.Lang, outputmsg.Text, outputmsg.SourceURL)

	    lang_list := []string{}
	    for key, value := range outputmsg.Translation {
			lang_list = append(lang_list, key)
	        if len(lang_content) > 0 {
                lang_content = lang_content + fmt.Sprintf("\n\n[%s]%s", key, value)
	        } else {
                lang_content = lang_content + fmt.Sprintf("[%s]%s", key, value)
	        }
        }

        currentSession := loadSession(outputmsg.U_id, outputmsg.Chat_id, db)
        currentSession.Output = outputmsg
        currentSession.State = TRANSLATE_OK
        currentSession.StateData = ""

        b, err := msgpack.Marshal(&currentSession)
        if err != nil {
            fmt.Println(err)
        } else {
            _, err := db.SetSession(outputmsg.Chat_id, outputmsg.U_id, b)
            if err != nil {
                fmt.Println(err)
            }
		    msg := tgbotapi.MessageConfig{
			    BaseChat: tgbotapi.BaseChat{
			        ChatID: outputmsg.Chat_id,
			        ReplyToMessageID: 0,
			    },
			    Text: fmt.Sprintf("%s\n%s",lang_content, ""),
			    //ParseMode: "Markdown",
			    DisableWebPagePreview: false,
		    }
		    msg.ReplyMarkup = makePublishKeyboard(lang_list)
		    bot.Send(msg)
        }
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
			Text: "Please select the original language",
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
}

func updateSession(input string, session *Session){
    text := strings.TrimSpace(input)

    validURL := util.IsURL(text)
    if validURL == true {
        session.Input.SourceURL=text
    } else {
        //if len(text) == 2 && 
        if len(text) == 2 && inputlangVerify(text) == true{
            session.Input.Lang=text
        } else {
            session.Input.Text=text
        }
    }
}

func isTwitterUrl(url string) bool{
	regex := *regexp.MustCompile(`https://twitter.com/[a-zA-Z0-9]+/status/\d+.*`)
	res := regex.FindIndex([]byte(url))
    if len(res) >=2 && res[0]==0{
        return true
    } else {
        return false
    }
}

func (session *Session) tryFetchUrl(ch chan spider.SpiderResponse , u_id int, chat_id int64) (bool, string){
    url := session.Input.SourceURL

    if isTwitterUrl(url) == true {
        //go fetch

	    s:= &spider.SpiderMessage{
            Chat_id:chat_id,
            U_id:u_id,
            URL: url,
        }
        go s.FetchTweetContent(ch)
        return true, "twitter"
    }else {
        return false,""
    }
}

func inputlangVerify(lang string) bool{
    result := false
    if LANG_CORRELATION[lang] != "" {
        lang = LANG_CORRELATION[lang]
    }
    for _, l := range SUPPORT_LANG_LIST {
        if lang == l {
            result = true
        }
    }
    return result
}

func loadSession(u_id int, chat_id int64, db *database.Db) Session{
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
    return currentSession

}

func ProcessUpdateCmdMessage(bot *tgbotapi.BotAPI, cmd string, query string, ch chan OutputMessage, db *database.Db, message_id int, u_id int, chat_id int64) {

    currentSession := loadSession(u_id, chat_id, db)

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
    } else if cmd =="EDIT"{
        currentSession.State = EDIT
        currentSession.StateData = query
        b, err := msgpack.Marshal(&currentSession)
        if err != nil {
            fmt.Println(err)
        } else {
            _, err := db.SetSession(chat_id, u_id, b)
            if err != nil {
                fmt.Println(err)
            }
            if currentSession.Output.Translation[currentSession.StateData] != ""{
                responsemsg := tgbotapi.NewMessage(chat_id, currentSession.Output.Translation[currentSession.StateData])
	            bot.Send(responsemsg)
                responsemsg = tgbotapi.NewMessage(chat_id, fmt.Sprintf("Edit [%s], Please input your translation:", currentSession.StateData))
	            bot.Send(responsemsg)
            }
        }
    } else if cmd =="SUBMIT"{
        if currentSession.State== DATA_OK {
            currentSession.State = SEND_TO_API
            go requestBriko(BRIKO_API, REQUEST_LANG_LIST , LANG_CORRELATION, message_id, chat_id, u_id, currentSession.Input, ch)
            b, err := msgpack.Marshal(&currentSession)
            if err != nil {
                fmt.Println(err)
            } else {
                _, err := db.SetSession(chat_id, u_id, b)
                if err != nil {
                    fmt.Println(err)
                }
                responsemsg := tgbotapi.NewMessage(chat_id, "Waiting for BRIKO AI translate.")
	            bot.Send(responsemsg)
            }
        } else {
                responsemsg := tgbotapi.NewMessage(chat_id, "Still Waiting for BRIKO AI translate. ")
	            bot.Send(responsemsg)
        }
    } else if cmd =="CANCEL"{
        _, err := db.DelSession(chat_id, u_id)
        if err == nil {
            responsemsg := tgbotapi.NewMessage(chat_id, "Cancelled, please input the content.")
	        bot.Send(responsemsg)
        } else {
            fmt.Println(err)
        }
        //delete session 
    } else if cmd =="PUBLISH"{
        lang_content := fmt.Sprintf("[%s]%s", currentSession.Output.Lang, currentSession.Output.Text)
	    lang_list := []string{}
	    for key, value := range currentSession.Output.Translation {
            lang_list = append(lang_list, key)
	        if len(lang_content) > 0 {
                lang_content = lang_content + fmt.Sprintf("\n\n[%s]%s", key, value)
	        } else {
                lang_content = lang_content + fmt.Sprintf("[%s]%s", key, value)
	        }
        }
        lang_content = lang_content + fmt.Sprintf("\n%s", currentSession.Output.SourceURL)
        publishresult := publishToChat(u_id, CHANNEL_CHAT_ID, lang_content, lang_list, bot, db)
        if publishresult ==true {
            _, err := db.DelSession(chat_id, u_id)
            if err == nil {
                responsemsg := tgbotapi.NewMessage(chat_id, "Publish successed. You can input new text to translate.")
	            bot.Send(responsemsg)
            } else {
                fmt.Println(err)
            }
        }
    } else {
        re_msg := tgbotapi.NewMessage(chat_id, "")
        re_msg.Text = fmt.Sprintf("Unknown Queryback command: %s", cmd)
    }
}

func ProcessUpdateMessageChat(bot *tgbotapi.BotAPI, update *tgbotapi.Update, chspider chan spider.SpiderResponse, db *database.Db,  u_id int, chat_id int64) {
    input := update.Message.Text

    currentSession := loadSession(u_id, chat_id, db)
    if currentSession.State != DONE {
		switch currentSession.State  {
            case NEED_DATA_INPUT:
                updateSession(input, &currentSession)
                canFetch := false
                if currentSession.Input.SourceURL!="" && currentSession.Input.Text ==""{
                    s := ""
                    canFetch,s = currentSession.tryFetchUrl(chspider, u_id, chat_id)
				    msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("I'm trying to fetch content from %s",s))
                    bot.Send(msg)
                }

                if canFetch == false{
                    r, responsemsg := currentSession.Input.verifyData(chat_id)
                    if r == true {
                        currentSession.State = DATA_OK
                    }
				    bot.Send(responsemsg)
                }
            case EDIT:
                lang := currentSession.StateData
                if lang !="" {
                    currentSession.Output.Translation[lang] = input

                    lang_content := fmt.Sprintf("[%s]%s %s", currentSession.Output.Lang, currentSession.Output.Text, currentSession.Output.SourceURL)
	                lang_list := []string{}
	                for key, value := range currentSession.Output.Translation {
                        lang_list = append(lang_list, key)
	                    if len(lang_content) > 0 {
                            lang_content = lang_content + fmt.Sprintf("\n\n[%s]%s", key, value)
	                    } else {
                            lang_content = lang_content + fmt.Sprintf("[%s]%s", key, value)
	                    }
                    }

		            msg := tgbotapi.MessageConfig{
			            BaseChat: tgbotapi.BaseChat{
			                ChatID: currentSession.Output.Chat_id,
			                ReplyToMessageID: 0,
			            },
			            Text: fmt.Sprintf("%s\n%s",lang_content, ""),
			            //ParseMode: "Markdown",
			            DisableWebPagePreview: false,
		            }
		            msg.ReplyMarkup = makePublishKeyboard(lang_list)
		            bot.Send(msg)
                }

			default:
                fmt.Println("unknown state")
        }
    }

    b, err := msgpack.Marshal(&currentSession)
    fmt.Println(err)
    if err == nil {
        _, err := db.SetSession(chat_id, u_id, b)
        if err != nil {
            fmt.Println(err)
        }
    } else {
        fmt.Println(err)
    }
}

func requestBriko(APIURL string, lang_list []string, lang_correlation map[string]string, msgId int,chat_id int64, u_id int, inmsg InputMessage, ch chan OutputMessage) {
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
                U_id: u_id,
                Text:    inmsg.Text,
                Lang:    inmsg.Lang,
                SourceURL: inmsg.SourceURL,
                Translation: rmsg.TranslationResults,
	        }
            ch <- *output
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

