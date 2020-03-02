package main

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/spf13/viper"
	"github.com/virushuo/brikobot/database"
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

//var rankingKeyboard = tgbotapi.NewInlineKeyboardMarkup(
//	tgbotapi.NewInlineKeyboardRow(
//		tgbotapi.NewInlineKeyboardButtonData("1", "1"),
//		tgbotapi.NewInlineKeyboardButtonData("2", "2"),
//		tgbotapi.NewInlineKeyboardButtonData("3", "3"),
//		tgbotapi.NewInlineKeyboardButtonData("4", "4"),
//		tgbotapi.NewInlineKeyboardButtonData("5", "5"),
//		tgbotapi.NewInlineKeyboardButtonURL("improve", "https://briko.org"),
//	),
//	tgbotapi.NewInlineKeyboardRow(
//		tgbotapi.NewInlineKeyboardButtonData("1", "1"),
//		tgbotapi.NewInlineKeyboardButtonData("2", "2"),
//		tgbotapi.NewInlineKeyboardButtonData("3", "3"),
//		tgbotapi.NewInlineKeyboardButtonData("4", "4"),
//		tgbotapi.NewInlineKeyboardButtonData("5", "5"),
//		tgbotapi.NewInlineKeyboardButtonURL("improve", "https://briko.org"),
//	),
//)

func makeRankingKeyboard(lang_list []string) tgbotapi.InlineKeyboardMarkup{
    var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, value := range lang_list{
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
			user_ranking, err := strconv.Atoi(update.CallbackQuery.Data)
			if err == nil { // error: ranking value must be a int
				commandtag, err := db.AddRanking(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.From.ID, user_ranking)
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

			//edit := tgbotapi.EditMessageTextConfig{
			//    BaseEdit: tgbotapi.BaseEdit{
			//        ChatID:    update.CallbackQuery.Message.Chat.ID,
			//        MessageID: update.CallbackQuery.Message.MessageID,
			//    },
			//    Text:  fmt.Sprintf("%s\n(%s)",update.CallbackQuery.Message.Text, update.CallbackQuery.Data) ,
			//}
			//_, err = bot.Send(edit)
		}
		if update.Message != nil {
			//update.Message.Chat.ID

			switch update.Message.Text {
			case "open":
				msg := tgbotapi.NewMessage(CHANNEL_CHAT_ID, update.Message.Text)
				msg.Text = "some test text"
				//msg.ReplyMarkup = rankingKeyboard
				bot.Send(msg)
			case "/new":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "new session")
				bot.Send(msg)
			case "/help":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "show help messages")
				bot.Send(msg)
			default:
				for _, value := range WHITELIST_ID_INT {
					if update.Message.From.ID == value {
						msg := tgbotapi.NewMessage(CHANNEL_CHAT_ID, update.Message.Text)
                        lang_list := make([]string, 2) //TOFIX: the lang_list should be created dynamically from user's answer. And we will build a multi-langs ranking keyboard.
                        lang_list[0]= "CN"
                        lang_list[1]= "FR"
                        newkeyboard := makeRankingKeyboard(lang_list)
						//msg.ReplyMarkup = rankingKeyboard
						msg.ReplyMarkup = newkeyboard

						sentmsg, err := bot.Send(msg)
						if err != nil {
							fmt.Fprintf(os.Stderr, "error: %v\n", err)
						}
						commandtag, err := db.AddMessage(sentmsg.Chat.ID, sentmsg.MessageID, update.Message.From.ID, update.Message.Text)
						if err != nil {
							fmt.Fprintf(os.Stderr, "error: %v\n", err)
							fmt.Fprintf(os.Stderr, "commandtag: %v\n", commandtag)
						}
					}
				}
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
