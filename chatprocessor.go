package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/abadojack/whatlanggo"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/virushuo/brikobot/database"
	"github.com/virushuo/brikobot/spider"
	"github.com/virushuo/brikobot/util"
	"github.com/vmihailenco/msgpack/v4"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	NEED_DATA_INPUT = 1 << iota
	FETCH_URL       = 1 << iota
	DATA_OK         = 1 << iota
	SEND_TO_API     = 1 << iota
	TRANSLATE_OK    = 1 << iota
	EDIT            = 1 << iota
	DONE            = 1 << iota
)

type InputMessage struct {
	Text      string
	Lang      string
	SourceURL string
}

type OutputMessage struct {
	Error       error
	Chat_id     int64
	U_id        int
	Text        string
	Lang        string
	SourceURL   string
	Translation map[string]string
	//Translation []string
	//LangList    []string
}

type Session struct {
	Id        uuid.UUID
	State     int
	StateData string
	Input     InputMessage
	Output    OutputMessage
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

	if submit == true {
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

func readTranslateOutputMessageChannel(c chan OutputMessage, bot *tgbotapi.BotAPI, db *database.Db) {
	for {
		outputmsg := <-c
		if outputmsg.Error == nil {
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
				glog.Errorf("readTranslateOutputMessageChannel msgpack Marshal error: %v\n", err)
			} else {
				_, err := db.SetSession(outputmsg.Chat_id, outputmsg.U_id, b)
				if err != nil {
					glog.Errorf("readTranslateOutputMessageChannel db.SetSession  error: %v\n", err)
				}
				msg := tgbotapi.MessageConfig{
					BaseChat: tgbotapi.BaseChat{
						ChatID:           outputmsg.Chat_id,
						ReplyToMessageID: 0,
					},
					Text: fmt.Sprintf("%s\n%s", lang_content, ""),
					//ParseMode: "Markdown",
					DisableWebPagePreview: true,
				}
				msg.ReplyMarkup = makePublishKeyboard(lang_list)
				bot.Send(msg)
			}
		} else {
			//outputmsg.Error
			currentSession := loadSession(outputmsg.U_id, outputmsg.Chat_id, db)
			currentSession.State = DATA_OK
			b, err := msgpack.Marshal(&currentSession)
			if err != nil {
				glog.Errorf("readTranslateOutputMessageChannel msgpack Marshal error: %v\n", err)
			} else {
				_, err := db.SetSession(outputmsg.Chat_id, outputmsg.U_id, b)
				if err != nil {
					glog.Errorf("readTranslateOutputMessageChannel db.SetSession  error: %v\n", err)
				}

				_, responsemsg := currentSession.Input.verifyData(outputmsg.Chat_id)
				bot.Send(responsemsg)

				errormsg := tgbotapi.NewMessage(outputmsg.Chat_id, fmt.Sprintf("BRIKO API Error: %s You can retry later.", outputmsg.Error.Error()))
				bot.Send(errormsg)
			}
		}
	}
}

func (inmsg *InputMessage) verifyData(chat_id int64) (bool, tgbotapi.MessageConfig) {
	glog.V(3).Infof("verifyData: %v", inmsg)
	if inmsg.Text == "" {
		return false, tgbotapi.NewMessage(chat_id, "please input the content")
	} else if inmsg.Lang == "" {
		responseMsg := tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID:           chat_id,
				ReplyToMessageID: 0,
			},
			Text: "Please select the original language",
			//ParseMode: "Markdown",
			DisableWebPagePreview: true,
		}
		responseMsg.ReplyMarkup = makeReplyKeyboard(SUPPORT_LANG_LIST, false)
		return false, responseMsg
	} else if inmsg.SourceURL == "" {
		return false, tgbotapi.NewMessage(chat_id, "please input the source url")
	}

	responseMsg := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:           chat_id,
			ReplyToMessageID: 0,
		},
		Text: fmt.Sprintf("Input: [%s]%s\n\nSource link:%s\n\nOriginal language is [%s], you can set original language with buttons below.", inmsg.Lang, inmsg.Text, inmsg.SourceURL, inmsg.Lang),
		//ParseMode: "Markdown",
		DisableWebPagePreview: true,
	}
	responseMsg.ReplyMarkup = makeReplyKeyboard(SUPPORT_LANG_LIST, true)
	return true, responseMsg
}

func updateSession(input string, session *Session) {
	text := strings.TrimSpace(input)

	validURL := util.IsURL(text)
	if validURL == true {
		session.Input.SourceURL = text
	} else {
		//if len(text) == 2 &&
		if len(text) == 2 && inputlangVerify(text) == true {
			session.Input.Lang = text
		} else {
			session.Input.Text = text
			lang_info := whatlanggo.Detect(text)
			input_lang := lang_info.Lang.Iso6391()
			if LANG_CORRELATION[input_lang] != "" {
				input_lang = LANG_CORRELATION[input_lang]
			}
			session.Input.Lang = input_lang
		}
	}
}

func isTwitterUrl(url string) bool {
	regex := *regexp.MustCompile(`https://twitter.com/[^/]+/status/\d+.*`)
	res := regex.FindIndex([]byte(url))
	if len(res) >= 2 && res[0] == 0 {
		return true
	} else {
		return false
	}
}

func (session *Session) tryFetchUrl(ch chan spider.SpiderResponse, u_id int, chat_id int64) (bool, string) {
	url := session.Input.SourceURL

	if isTwitterUrl(url) == true {
		//go fetch

		s := &spider.SpiderMessage{
			Chat_id: chat_id,
			U_id:    u_id,
			URL:     url,
		}
		go s.FetchTweetContent(ch)
		return true, "twitter"
	} else {
		return false, ""
	}
}

func inputlangVerify(lang string) bool {
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

func loadSession(u_id int, chat_id int64, db *database.Db) Session {
	var currentSession Session
	data, err := db.GetSession(chat_id, u_id)
	if len(data) == 0 {
		currentSession = Session{
			Id:     uuid.New(),
			State:  NEED_DATA_INPUT,
			Input:  InputMessage{},
			Output: OutputMessage{},
		}
	} else {
		if err == nil {
			err = msgpack.Unmarshal(data, &currentSession)
			if err != nil {
				glog.Errorf("load session Unmarshal error: %v\n", err)
			}
		} else {
			glog.Errorf("db.GetSession error: %v\n", err)
		}
	}
	return currentSession

}

func ProcessUpdateCmdMessage(bot *tgbotapi.BotAPI, cmd string, query string, ch chan OutputMessage, db *database.Db, message_id int, u_id int, chat_id int64, username string) {

	currentSession := loadSession(u_id, chat_id, db)

	if cmd == "SETLANG" {
		currentSession.Input.Lang = query
		r, responsemsg := currentSession.Input.verifyData(chat_id)
		if r == true {
			currentSession.State = DATA_OK
		}

		b, err := msgpack.Marshal(&currentSession)
		if err != nil {
			glog.Errorf("SETLANG Marshal error: %v\n", err)
		} else {
			_, err := db.SetSession(chat_id, u_id, b)
			if err != nil {
				glog.Errorf("SETLANG SetSession error: %v\n", err)
			}
			bot.Send(responsemsg)
		}
	} else if cmd == "EDIT" {
		currentSession.State = EDIT
		currentSession.StateData = query
		b, err := msgpack.Marshal(&currentSession)
		if err != nil {
			glog.Errorf("EDIT Marshal error: %v\n", err)
		} else {
			_, err := db.SetSession(chat_id, u_id, b)
			if err != nil {
				glog.Errorf("EDIT SetSession error: %v\n", err)
			}
			if currentSession.Output.Translation[currentSession.StateData] != "" {
				responsemsg := tgbotapi.NewMessage(chat_id, currentSession.Output.Translation[currentSession.StateData])
				bot.Send(responsemsg)
				responsemsg = tgbotapi.NewMessage(chat_id, fmt.Sprintf("Edit [%s], Please input your translation:", currentSession.StateData))
				bot.Send(responsemsg)
			}
		}
	} else if cmd == "SUBMIT" {
		if currentSession.State == DATA_OK {
			currentSession.State = SEND_TO_API
			go requestBriko(BRIKO_API, REQUEST_LANG_LIST, LANG_CORRELATION, message_id, chat_id, u_id, currentSession, ch)
			b, err := msgpack.Marshal(&currentSession)
			if err != nil {
				glog.Errorf("SUBMIT Marshal error: %v\n", err)
			} else {
				_, err := db.SetSession(chat_id, u_id, b)
				if err != nil {
					glog.Errorf("SUBMIT SetSession error: %v\n", err)
				}
				responsemsg := tgbotapi.NewMessage(chat_id, "Waiting for BRIKO AI translate.")
				bot.Send(responsemsg)
			}
		} else {
			responsemsg := tgbotapi.NewMessage(chat_id, "Still Waiting for BRIKO AI translate. ")
			bot.Send(responsemsg)
		}
	} else if cmd == "CANCEL" {
		_, err := db.DelSession(chat_id, u_id)
		if err == nil {
			responsemsg := tgbotapi.NewMessage(chat_id, "Cancelled, please input the content.")
			bot.Send(responsemsg)
		} else {
			glog.Errorf("CANCEL DelSession error: %v\n", err)
		}
		//delete session
	} else if cmd == "PUBLISH" {
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
		publishresult := publishToChat(u_id, username, CHANNEL_CHAT_ID, lang_content, lang_list, bot, db)
		if publishresult == true {
			msgtext := "Publish successed. You can input new text to translate."
			_, err := db.DelSession(chat_id, u_id)
			if err != nil {
				glog.Errorf("CANCEL DelSession error: %v\n", err)
				msgtext += "\ninput /reset to start the next task."
			}
			responsemsg := tgbotapi.NewMessage(chat_id, msgtext)
			bot.Send(responsemsg)
		}
	} else {
		re_msg := tgbotapi.NewMessage(chat_id, "")
		re_msg.Text = fmt.Sprintf("Unknown Queryback command: %s", cmd)
	}
}

func ProcessUpdateMessageChat(bot *tgbotapi.BotAPI, update *tgbotapi.Update, chspider chan spider.SpiderResponse, db *database.Db, u_id int, chat_id int64) {
	input := update.Message.Text

	currentSession := loadSession(u_id, chat_id, db)
	if currentSession.State != DONE {
		switch currentSession.State {
		case NEED_DATA_INPUT:
			updateSession(input, &currentSession)
			canFetch := false
			if currentSession.Input.SourceURL != "" && currentSession.Input.Text == "" {
				s := ""
				canFetch, s = currentSession.tryFetchUrl(chspider, u_id, chat_id)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Sorry, I can't fetch this link %s", s))
				if canFetch == true {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("I'm trying to fetch content from %s", s))
				}
				bot.Send(msg)
			}

			if canFetch == false {
				r, responsemsg := currentSession.Input.verifyData(chat_id)
				if r == true {
					currentSession.State = DATA_OK
				}
				bot.Send(responsemsg)
			}
		case EDIT:
			lang := currentSession.StateData
			if lang != "" {
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
						ChatID:           currentSession.Output.Chat_id,
						ReplyToMessageID: 0,
					},
					Text: fmt.Sprintf("%s\n%s", lang_content, ""),
					//ParseMode: "Markdown",
					DisableWebPagePreview: true,
				}
				msg.ReplyMarkup = makePublishKeyboard(lang_list)
				bot.Send(msg)
			}
		default:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "input /reset to start the new task.")
			bot.Send(msg)
		}
	}

	b, err := msgpack.Marshal(&currentSession)
	if err == nil {
		_, err := db.SetSession(chat_id, u_id, b)
		if err != nil {
			glog.Errorf("ProcessUpdateMessageChat db.SetSession error: %v\n", err)
		}
	} else {
		glog.Errorf("SUBMIT Marshal error: %v\n", err)
	}
}

func requestBriko(APIURL string, lang_list []string, lang_correlation map[string]string, msgId int, chat_id int64, u_id int, currentSession Session, ch chan OutputMessage) {
	inmsg := currentSession.Input
	data := &requestMsg{
		MsgType:    "Translation",
		MsgID:      strconv.Itoa(msgId),
		SourceLang: inmsg.Lang,
	}

	if lang_correlation[data.SourceLang] != "" {
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
	resp, httperr := http.Post(APIURL, "application/json", bytes.NewBuffer(output))
	if httperr != nil {
		glog.Errorf("BRIKO API connect error: %w\n", httperr)
		apierr := fmt.Errorf("BRIKO API connect error: %w\n", httperr)
		output := &OutputMessage{
			Error:   apierr,
			Chat_id: chat_id,
			U_id:    u_id,
		}
		ch <- *output
		return
	} else {
		bodyBytes, err1 := ioutil.ReadAll(resp.Body)
		if err1 != nil {
			glog.Errorf("BRIKO API response error: %w\n", httperr)
			apierr := fmt.Errorf("BRIKO API response error: %w\n", httperr)

			output := &OutputMessage{
				Error:   apierr,
				Chat_id: chat_id,
				U_id:    u_id,
			}
			ch <- *output
			return
		}
		bodyString := string(bodyBytes)
		d := json.NewDecoder(strings.NewReader(bodyString))
		rmsg := &responseMsg{}
		err := d.Decode(rmsg)

		if err != nil {
		} else {
			if rmsg.MsgFlag == "success" {
				output := &OutputMessage{
					Error:       nil,
					Chat_id:     chat_id,
					U_id:        u_id,
					Text:        inmsg.Text,
					Lang:        inmsg.Lang,
					SourceURL:   inmsg.SourceURL,
					Translation: rmsg.TranslationResults,
				}
				ch <- *output
			} else {
				fmt.Println(rmsg.MsgFlag) //TODO: send the error msg to bot
			}
		}
	}
}
