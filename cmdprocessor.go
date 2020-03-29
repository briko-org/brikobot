package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/virushuo/brikobot/session"
	"github.com/virushuo/brikobot/util"
	"fmt"
	"strings"
	"regexp"
	"github.com/virushuo/brikobot/database"
)

func verifyCommandMsg(message string) (bool,string){
    if strings.Index(message, "/input") == 0 {
        inputstr := strings.TrimLeft(message[6:], " ")
        if len(inputstr) > 4 + MIN_INPUT_LENGTH {
            match, _:= regexp.Match(`\[([A-Z]{2})\]`, []byte(strings.ToUpper(inputstr[:4])))
            if match == true{
                split_list := strings.Split(inputstr, " ")
                last_str := split_list[len(split_list)-1]
                validURL := util.IsURL(last_str)
                if validURL == false {
                    return false, "The original URL is required. eg: /input [en] this is an apple. https://thisisanapple.com"
                }
                return true,""
            }else {
                return false, "no language tag. eg: /input [en] this is an apple. https://thisisanapple.com"
            }
        } else {
            return false, fmt.Sprintf("minimum input length is %d", 4 + MIN_INPUT_LENGTH)
        }
    }
    return true, ""
}

func ProcessUpdateMessageWithSlash(bot *tgbotapi.BotAPI, update *tgbotapi.Update, ch chan session.State, db *database.Db,  u_id int, chat_id int64) string{
	n, t, err := db.GetChatState(chat_id, u_id)
    var msgtext string
    if update.Message.Text == "/help" || update.Message.Text == "/?" {
        msgtext = HELP_TEXT
    } else if update.Message.Text == "/new" {
    	stat := session.New(u_id, chat_id)
    	stat.Name = "NONE"
    	stat.Text = ""
    	stat_next := session.New(u_id, chat_id)
    	stat_next.Name = "NEW"
    	stat_next.Text = t
    	r, str := stat.NextUpdate(stat_next, db)
    	if r == true {
    		msgtext = str
    	} else {
    		msgtext = "error"
    	}
    } else if update.Message.Text == "/show" {
    	if err != nil && err.Error() == "no rows in result set" {
    		msgtext = "Current state is nil, send /help for help, send /new to start"
    	} else if err != nil {
    		msgtext = "Error: " + err.Error()
    	} else {
    		msgtext = fmt.Sprintf("Show current status:\nState: %s\nText: %s", n, t)
    	}
    } else {
        verifyresult, verifymsg := verifyCommandMsg(update.Message.Text)
        if verifyresult == false {
            return verifymsg
        }
    	if err != nil && err.Error() == "no rows in result set" {
    		stat := session.New(u_id, chat_id)
    		stat.Name = "NONE"
    		stat.Text = ""
    		msgtext = stat.Response(session.New(u_id, chat_id))
    	} else if err == nil {
    		stat := session.New(u_id, chat_id)
    		stat.Name = n
    		stat.Text = t
    
    		stat_next := session.New(u_id, chat_id)
    		idx := strings.Index(update.Message.Text, " ")
    		if idx > 1 {
    			name := update.Message.Text[1:idx]
    			text := update.Message.Text[idx+1:]
    			stat_next.Name = strings.ToUpper(name)
    			stat_next.Text = text
    		} else {
    			stat_next.Name = strings.ToUpper(update.Message.Text[1:])
    			stat_next.Text = ""
    		}
    
    		r, str := stat.NextUpdate(stat_next, db)
    		if stat_next.Name == "INPUT" && r == true {
    			go stat_next.RequestBriko(BRIKO_API, REQUEST_LANG_LIST , LANG_CORRELATION, update.Message.MessageID, ch)
    		}
    		if stat_next.Name == "UPDATE" && r == true {
                r, str = stat.MergeUpdateState(stat_next, LANG_CORRELATION)
                if r == true {
                    stat_next.Text=str
    		        r, str = stat.NextUpdate(stat_next, db)
                }
            }
    
    		if stat_next.Name == "PUBLISH" && r == true {
    			idx := strings.Index(stat.Text, ":")
    			lang_list_str := stat.Text[:idx]
    			to_publish_text := stat.Text[idx+1:]
    			regex := *regexp.MustCompile(`\[([A-Za-z]{2})\]`)
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
                return ""
    		}
    		msgtext = str
    	}
    }
    if len(msgtext)==0 {
    	stat := session.New(u_id, chat_id)
    	stat.Name = n
    	stat.Text = t
        state_list := stat.NextState()
        menuitem := session.MakeMenu(state_list)
        msgtext = "Unknown command\n"
        msgtext += menuitem
    
    }
    return msgtext
}

