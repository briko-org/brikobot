package main

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/spf13/viper"
	"github.com/virushuo/brikobot/database"
	"github.com/virushuo/brikobot/session"
    //"database/sql"
    //"errors"
    "strings"
    "regexp"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

var db *database.Db
var (
	PG_URL             string
	BOT_TOKEN          string
	CHANNEL_CHAT_ID    int64
	WHITELIST_ID_INT []int
)

func makeRankingKeyboard(lang_list []string) tgbotapi.InlineKeyboardMarkup{
    var keyboard [][]tgbotapi.InlineKeyboardButton
	for idx, value := range lang_list{
        if idx >0 {
            var row []tgbotapi.InlineKeyboardButton
            for i := 0; i < 5; i++ {
                label := strconv.Itoa(i+1)
                if i==0 {
                    label = value+" "+strconv.Itoa(i+1)
                }
                button := tgbotapi.NewInlineKeyboardButtonData(label, value+","+strconv.Itoa(i+1))
	            row = append(row, button)
            }
            keyboard = append(keyboard, row)
        }
    }
    return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}

func loadconf() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(filepath.Dir("."))
	viper.ReadInConfig()
	PG_URL = viper.GetString("PG_URL")
	BOT_TOKEN = viper.GetString("BOT_TOKEN")
	CHANNEL_CHAT_ID = viper.GetInt64("CHANNEL_CHAT_ID")
}

func loadwhitelist() {
	var WHITELIST_ID []string

	viper.SetConfigName("whitelist")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Dir("."))

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

func publishToChat(from_id int, chat_id int64, text string, lang_list []string,bot *tgbotapi.BotAPI, db *database.Db) {
    for _, value := range WHITELIST_ID_INT {
        if  from_id == value {
            msg := tgbotapi.NewMessage(chat_id, text)
            newkeyboard := makeRankingKeyboard(lang_list)
            msg.ReplyMarkup = newkeyboard
            sentmsg, err := bot.Send(msg)
            if err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
            }
            fmt.Printf("====addMessage%v %v %v %v\n",sentmsg.Chat.ID, sentmsg.MessageID, from_id, text)
            commandtag, err := db.AddMessage(sentmsg.Chat.ID, sentmsg.MessageID, from_id, text)
            if err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                fmt.Fprintf(os.Stderr, "commandtag: %v\n", commandtag)
            }
        }
    }
}

func startservice(bot *tgbotapi.BotAPI, db *database.Db) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	for update := range updates {
        fmt.Println("===output update")
        fmt.Println(update)
		if update.CallbackQuery != nil {
            callbackdata := strings.Split(update.CallbackQuery.Data, ",")
            if len(callbackdata)==2 {
                lang := callbackdata[0]
			    user_ranking, err := strconv.Atoi(callbackdata[1])
			    if err == nil { // error: ranking value must be a int
			    	commandtag, err := db.AddRanking(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.From.ID, lang, user_ranking)
			    	if err != nil {
			    		fmt.Fprintf(os.Stderr, "error: %v\n", err)
			    		fmt.Fprintf(os.Stderr, "commandtag: %v\n", commandtag)
			    	} else {
			    		re_msg := tgbotapi.NewMessage(int64(update.CallbackQuery.From.ID), "")
			    		re_msg.Text = fmt.Sprintf("Ranking %s Message %d has been submitted.", update.CallbackQuery.Data, update.CallbackQuery.Message.MessageID)
			    		bot.Send(re_msg)
			    	}
			    } else {
			    	fmt.Fprintf(os.Stderr, "ranking value strconv error: %s %v\n", update.CallbackQuery.Data, err)
			    }
			    bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data))
            }
		}
		if update.Message != nil {

            chat_id := update.Message.Chat.ID
            u_id := update.Message.From.ID
            n,t,err := db.GetChatState(chat_id, u_id)
            msgtext := "default text"

			switch []byte(update.Message.Text)[0] {
			case 47: //start with "/"
                if update.Message.Text == "/new" {
                    stat:= session.New(u_id, chat_id)
                    stat.Name ="NONE"
                    stat.Text = ""

                    stat_next:= session.New(u_id, chat_id)
                    stat_next.Name = "NEW"
                    stat_next.Text = t
                    r, str := stat.NextUpdate(stat_next, db)
                    if r == true{
                        msgtext = str
                    }else {
                        msgtext = "error"
                    }
                } else if update.Message.Text == "/show" {
                    if err!=nil && err.Error() == "no rows in result set" {
                        msgtext = "Current state is nil, send /help for help, send /new to start"
                    } else if err!=nil {
                        msgtext = "Error: " + err.Error()
                    } else {
                        msgtext = fmt.Sprintf("Show current state:\nState: %s\nText: %s",n,t)
                    }
                } else {
                    if err!=nil && err.Error() == "no rows in result set" {
                        stat:= session.New(u_id, chat_id)
                        stat.Name = "NONE"
                        stat.Text = ""
                        msgtext = stat.Response(session.New(u_id, chat_id))
                    }else if err == nil {
                        stat:= session.New(u_id, chat_id)
                        stat.Name = n
                        stat.Text = t

                        stat_next:= session.New(u_id, chat_id)
                        idx := strings.Index(update.Message.Text, " ")
                        if idx > 1 {
                            name := update.Message.Text[1:idx]
                            text := update.Message.Text[idx+1:]
                            stat_next.Name = strings.ToUpper(name)
                            stat_next.Text = text
                        }else{
                            stat_next.Name = strings.ToUpper(update.Message.Text[1:])
                            stat_next.Text = ""
                        }

                        r, str := stat.NextUpdate(stat_next, db)

                        if stat_next.Name == "PUBLISH" && r == true {
                            idx := strings.Index(stat.Text, ":")
                            lang_list_str := stat.Text[:idx]
                            to_publish_text := stat.Text[idx+1:]
                            regex := *regexp.MustCompile(`\[([A-Z]{2})\]`)
                            res := regex.FindAllStringSubmatch(lang_list_str, -1)
                            lang_list := make([]string, len(res))
                            if len(res) > 1 {
                                for i, value := range res {
                                    if len(value) == 2 {
                                        lang_list[i] = value[1]
                                    }
                                }
                            } else {
                                fmt.Println("no language tag")
                            }

                            publishToChat(update.Message.From.ID, CHANNEL_CHAT_ID, to_publish_text, lang_list, bot, db)
                        }
                        msgtext = str
                        fmt.Println("DEBUG: stat")
                        fmt.Println(stat)
                        fmt.Println(stat_next)
                        fmt.Println(r)
                        fmt.Println(str)
                    }
                }
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgtext )
				bot.Send(msg)
			default:
                if err!=nil && err.Error() == "no rows in result set" {
                    msgtext = "Current state is nil, send /help for help, send /new to start"
                } else if err!=nil {
                    msgtext = "Error: " + err.Error()
                } else {
                    msgtext = fmt.Sprintf("Show current state:\nState: %s\nText: %s",n,t)
                }
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgtext )
				bot.Send(msg)
			}
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
