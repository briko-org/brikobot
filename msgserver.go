package main

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/abadojack/whatlanggo"
	"github.com/spf13/viper"
	"github.com/virushuo/brikobot/database"
	"github.com/virushuo/brikobot/spider"
    "github.com/vmihailenco/msgpack/v4"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var db *database.Db
var (
	PG_URL           string
	BOT_TOKEN        string
	CHANNEL_CHAT_ID  int64
	WHITELIST_ID_INT []int
    MIN_INPUT_LENGTH int
    BRIKO_API   string
    REQUEST_LANG_LIST []string
    SUPPORT_LANG_LIST []string
    HELP_TEXT string
    LANG_CORRELATION map[string]string
)

func makeRankingKeyboard(lang_list []string) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, value := range lang_list {
			var row []tgbotapi.InlineKeyboardButton
			for i := 0; i < 5; i++ {
				label := strconv.Itoa(i + 1)
				if i == 0  { //&& len(lang_list) > 2
					label = value + " " + strconv.Itoa(i+1)
				}
				button := tgbotapi.NewInlineKeyboardButtonData(label, value+","+strconv.Itoa(i+1))
				row = append(row, button)
			}
			keyboard = append(keyboard, row)
	}
	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}

func loadconf() {
	viper.AddConfigPath(filepath.Dir("./config/"))
	viper.AddConfigPath(filepath.Dir("."))
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.ReadInConfig()
	PG_URL = viper.GetString("PG_URL")
	BOT_TOKEN = viper.GetString("BOT_TOKEN")
	CHANNEL_CHAT_ID = viper.GetInt64("CHANNEL_CHAT_ID")
	MIN_INPUT_LENGTH = viper.GetInt("MIN_INPUT_LENGTH")
    BRIKO_API = viper.GetString("BRIKO_API")
	REQUEST_LANG_LIST = viper.GetStringSlice("REQUEST_LANG_LIST")
    SUPPORT_LANG_LIST = viper.GetStringSlice("SUPPORT_LANG_LIST")
    HELP_TEXT = viper.GetString("HELP_TEXT")
    LANG_CORRELATION = viper.GetStringMapString("LANG_CORRELATION")
}

func loadwhitelist() {
	var WHITELIST_ID []string
	viper.AddConfigPath(filepath.Dir("./config/"))
	viper.AddConfigPath(filepath.Dir("."))
	viper.SetConfigName("whitelist")
	viper.SetConfigType("yaml")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error parsing config file: %s \n", err))
	}

	WHITELIST_ID = viper.GetStringSlice("whitelist")

	for _, value := range WHITELIST_ID {
		temp_int, err := strconv.Atoi(value)
		if err != nil {
			panic(fmt.Errorf("Fatal error parsing config file: %s \n", err))
		}
		WHITELIST_ID_INT = append(WHITELIST_ID_INT, temp_int)
	}
}

func publishToChat(from_id int, chat_id int64, text string, lang_list []string, bot *tgbotapi.BotAPI, db *database.Db) bool {
    allow_publish := false
	for _, value := range WHITELIST_ID_INT {
	    if from_id == value {
            allow_publish = true
        }
	}
	if allow_publish == true  {
		//msg := tgbotapi.NewMessage(chat_id, text)
		msg := tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID: chat_id,
				ReplyToMessageID: 0,
			},
			Text: text,
			//ParseMode: "Markdown",
			DisableWebPagePreview: false,
		}

		newkeyboard := makeRankingKeyboard(lang_list)
		msg.ReplyMarkup = newkeyboard
		sentmsg, err := bot.Send(msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		commandtag, err := db.AddMessage(sentmsg.Chat.ID, sentmsg.MessageID, from_id, text)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			fmt.Fprintf(os.Stderr, "commandtag: %v\n", commandtag)
            return false
		}else {
            return true
        }
	}else {
        fmt.Println("userid not in the whitelist %v %v", from_id, chat_id)
        return false
    }
}


func startservice(bot *tgbotapi.BotAPI, db *database.Db) {
	//var ch chan session.State = make(chan session.State)
	//go readTranslateChannel(ch, bot, db)

    var choutput chan OutputMessage = make(chan OutputMessage)
    go readTranslateOutputMessageChannel(choutput, bot, db)

	var chspider chan spider.SpiderResponse= make(chan spider.SpiderResponse)
	go readSpiderChannel(chspider, bot, db) //, bot

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	for update := range updates {
		if update.CallbackQuery != nil {
            callbackcmd :=  strings.Split(update.CallbackQuery.Data, "_")
			if len(callbackcmd) == 2 { //is callback cmd

			    chat_id := int64(update.CallbackQuery.From.ID)
			    u_id := update.CallbackQuery.From.ID
                cmd := callbackcmd[0]
                fmt.Println("==========Query:")
                fmt.Println(update.CallbackQuery.Data)
                if cmd =="SETLANG" || cmd =="SUBMIT" || cmd =="CANCEL" || cmd == "EDIT" || cmd == "PUBLISH"{
				    ProcessUpdateCmdMessage(bot, cmd, callbackcmd[1], choutput, db, update.CallbackQuery.Message.MessageID, u_id , chat_id )
			        bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data))
                }
            } else {
			    callbackdata := strings.Split(update.CallbackQuery.Data, ",")
			    if len(callbackdata) == 2 {
			    	lang := callbackdata[0]
			    	user_ranking, err := strconv.Atoi(callbackdata[1])
			    	if err == nil { // error: ranking value must be a int
			    		commandtag, err := db.AddRanking(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.From.ID, lang, user_ranking)
			    		if err != nil {
			    			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			    			fmt.Fprintf(os.Stderr, "commandtag: %v\n", commandtag)
			    		} else {
			    			re_msg := tgbotapi.NewMessage(int64(update.CallbackQuery.From.ID), "")
			    			re_msg.Text = fmt.Sprintf("Rating %s Message %d has been submitted.", update.CallbackQuery.Data, update.CallbackQuery.Message.MessageID)
			    			bot.Send(re_msg)
			    		}
			    	} else {
			    		fmt.Fprintf(os.Stderr, "rating value strconv error: %s %v\n", update.CallbackQuery.Data, err)
			    	}
			    	bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data))
			    }
            }

		}
		if update.Message != nil {
			chat_id := update.Message.Chat.ID
			u_id := update.Message.From.ID
			n, t, err := db.GetChatState(chat_id, u_id)
			msgtext := "default text"
            fmt.Println(n)
            fmt.Println(t)
            fmt.Println(err)

			switch []byte(update.Message.Text)[0] {
            case 63: //"?"
                msgtext = HELP_TEXT
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgtext)
				bot.Send(msg)
			case 47: //start with "/"
                msgtext := "unknown command. send ? or /help for help."
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgtext)
                if update.Message.Text =="/help" || update.Message.Text =="/start"{
                    msgtext = HELP_TEXT
				    msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgtext)
                } else if update.Message.Text =="/reset" || update.Message.Text =="/del" {
                    db.DelSession(chat_id, u_id)
                    msgtext = "Cleared, please input new content or url."
				    msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgtext)
                }
                bot.Send(msg)
			default:
                ProcessUpdateMessageChat(bot, &update, chspider, db,  u_id , chat_id )
			}
		}
	}
}

func readSpiderChannel(c chan spider.SpiderResponse, bot *tgbotapi.BotAPI, db *database.Db) {
	for {
		spidermsg := <-c
        fmt.Println(spidermsg)
        if spidermsg.Content !=""{
            currentSession := loadSession(spidermsg.U_id, spidermsg.Chat_id, db)
            currentSession.Input.Text = spidermsg.Content
	        lang_info := whatlanggo.Detect(spidermsg.Content)
            input_lang := lang_info.Lang.Iso6391()
            if LANG_CORRELATION[input_lang] != "" {
                input_lang = LANG_CORRELATION[input_lang]
            }

	        for _, value := range SUPPORT_LANG_LIST{
                if value == input_lang {
                    currentSession.Input.Lang = input_lang
                }
            }

            msgtext := fmt.Sprintf("Fetch content from %s\n%s",spidermsg.Url, spidermsg.Content)
            if currentSession.Input.Lang != "" {
                msgtext += fmt.Sprintf("\nSet Language: %s",currentSession.Input.Lang)
            }
            msg := tgbotapi.NewMessage(spidermsg.Chat_id, msgtext)
		    bot.Send(msg)

            currentSession.Input.SourceURL = spidermsg.Url
            r, responsemsg := currentSession.Input.verifyData(spidermsg.Chat_id)
            if r == true {
                currentSession.State = DATA_OK
            }

            b, err := msgpack.Marshal(&currentSession)
            fmt.Println(err)
            if err == nil {
                commandtag, err := db.SetSession(spidermsg.Chat_id, spidermsg.U_id, b)
                fmt.Println(commandtag)
                if err != nil {
                    fmt.Println(err)
                } else {
				    bot.Send(responsemsg)
                }
            } else {
                fmt.Println(err)
            }

        } else {
            msg := tgbotapi.NewMessage(spidermsg.Chat_id, fmt.Sprintf("Can't fetch content from %s , please input the content.",spidermsg.Url))
		    bot.Send(msg)
        }
	}
}


func main() {
	loadconf()
	loadwhitelist()

	db, err := database.New(PG_URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	bot, err := tgbotapi.NewBotAPI(BOT_TOKEN)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)
	startservice(bot, db)
}
