package session

import (
	"bytes"
	"encoding/json"
	"strconv"
	"fmt"
	"io/ioutil"
    "github.com/asaskevich/govalidator"
	"github.com/virushuo/brikobot/database"
	"net/http"
	"regexp"
	"strings"
)

type State struct {
	Name    string
	Text    string
	U_id    int
	Chat_id int64
}

func New(u_id int, chat_id int64) *State {
	stat := new(State)
	stat.U_id = u_id
	stat.Chat_id = chat_id
	stat.Name = "NONE"
	stat.Text = ""
	return stat
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



func menuItemdetail(cmd string) string{
    cmd_detail := cmd
    if cmd == "input" {
        cmd_detail = "input to submit content to BRIKOAI.\nformat: /input [lang] content source_url.\neg: /input [en] this is an apple. https://thisisanapple.com"
    } else if cmd == "update" {
        cmd_detail = "update to edit incorrect translations.\nformat: /update [lang] content.\neg: /update [en] this is a pineapple."
    } else if cmd == "publish" {
        cmd_detail = "publish to publish to the channel."
    } else if cmd == "show" {
        cmd_detail = "show to show the current status."
    } else if cmd == "new" {
        cmd_detail = "new to initialize a new task."
    }
	return fmt.Sprintf("/%s\n", cmd_detail)
}

func MakeMenu(state_list []string) string {
	var menu string
	for _, value := range state_list {
		if value != "TRANSLATE" {
	        menu += menuItemdetail(strings.ToLower(value))
		}
	}
	return menu
}

func (stat *State) Response(nextstat *State) string {
	var msg string
	stat_list := stat.NextState()
    wait_for_briko_ai := false
	for _, name := range stat_list {
		if name == "TRANSLATE" {
			msg = "Waiting for BRIKO AI translate"
            wait_for_briko_ai = true
		}
	}

    if wait_for_briko_ai  == true {
	    msg = "Waiting for BRIKO AI translate, input /show to check current status."
    } else {
	    if nextstat.Name == stat.Name {
            if stat.Name == "UPDATE"{
                msg = msg + "\nUpdate: " + stat.Text
            }
	        msg = msg + "\nYou can send these commands:\n" + MakeMenu(stat_list)
	    }
	    if nextstat.Name == "NEW" {
	        msg = "New task initiated,\nYou can send these commands:" + "\n" + MakeMenu(stat_list)
	    }
    }
	return msg
}

func (stat *State) NextUpdate(nextstat *State, db *database.Db) (bool, string) {
	if nextstat.Chat_id != stat.Chat_id || nextstat.U_id != stat.U_id {
		return false, stat.Response(nextstat)
	}

	stat_list := stat.NextState()
	var if_allowed_transition bool = false
	for _, name := range stat_list {
		if name == nextstat.Name {
			if_allowed_transition = true
		}
	}

	if if_allowed_transition == true {
		//update
        fmt.Println("=============")
        fmt.Println(nextstat.Text)
        fmt.Println(stat.Text)
		commandtag, err := db.SetChatState(nextstat.Chat_id, nextstat.U_id, nextstat.Name, nextstat.Text)
		fmt.Println(commandtag)
		fmt.Println(err)
		return true, nextstat.Response(nextstat)
	}
	return false, stat.Response(nextstat)
}

func (stat *State) NextState() []string {
	var state_list []string
	switch stat.Name {
	case "HELP":
	case "NONE":
		state_list = append(state_list, "ABOUT")
		state_list = append(state_list, "HELP")
		state_list = append(state_list, "NEW")
	case "NEW":
		state_list = append(state_list, "INPUT")
	case "INPUT":
		state_list = append(state_list, "INPUT")
		state_list = append(state_list, "TRANSLATE")
	case "TRANSLATE":
		state_list = append(state_list, "UPDATE")
		state_list = append(state_list, "PUBLISH")
	case "UPDATE":
		state_list = append(state_list, "UPDATE")
		state_list = append(state_list, "PUBLISH")
	case "IMPROVE":
		state_list = append(state_list, "SUBMIT")
	case "SUBMIT":
	case "PUBLISH":
		state_list = append(state_list, "NEW")
	}
	return state_list
}

func (stat *State) RequestBriko(APIURL string, lang_list []string, msgId int, ch chan State) {
	data := &requestMsg{
		MsgType:    "Translation",
		MsgID:      strconv.Itoa(msgId),
		SourceLang: "en",
	}

	regex := *regexp.MustCompile(`\[([A-Za-z]{2})\]`)
	res := regex.FindStringSubmatch(stat.Text)
	if len(res) > 1 {
		data.SourceLang = strings.ToLower(res[1])
        split_list := strings.Split(stat.Text, " ")
        last_str := split_list[len(split_list)-1]
        validURL := govalidator.IsURL(last_str)
        end_pos := len(last_str)-1
        if validURL == true {
            end_pos = len(stat.Text) - len(last_str)
        }

		requestLang := []string{}
		for _, value := range lang_list {
			if value != data.SourceLang {
				requestLang = append(requestLang, value)
			}
		}

        data.SourceContent = string(stat.Text[4:end_pos])
        SourceURL := string(stat.Text[end_pos:])

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
            fmt.Println("===rmsg err");
			fmt.Println(err) //TODO: send the error msg to bot
		} else {
			if rmsg.MsgFlag == "success" {
                lang_content := fmt.Sprintf("[%s] %s %s", data.SourceLang, data.SourceContent, SourceURL)
				lang_list_str := fmt.Sprintf("[%s]", data.SourceLang)
				translation := rmsg.TranslationResults
				for key, value := range translation {
					lang_content = lang_content + fmt.Sprintf("\n\n[%s] %s", key, value)
					lang_list_str = lang_list_str + fmt.Sprintf("[%s]", key)
				}
				ch <- State{"TRANSLATE", fmt.Sprintf("%s:%s", lang_list_str, lang_content), stat.U_id, stat.Chat_id}
			} else {
				fmt.Println(rmsg.MsgFlag) //TODO: send the error msg to bot
			}
		}
	} else {
		fmt.Println("no language tag")
	}
}


func (stat *State) MergeUpdateState(next_stat *State) (bool, string){
    if next_stat.Name == "UPDATE" && (stat.Name == "TRANSLATE" || stat.Name == "UPDATE") {

        if len(strings.TrimSpace(next_stat.Text))<=4 {
            return false, "update text format error"
        }


        idx := strings.Index(stat.Text, ":")
        if idx <= 0 {
            return false, "update text format error"
        }
        lang_list_str := stat.Text[:idx]
        to_publish_text := stat.Text[idx+1:]
        regex := *regexp.MustCompile(`\[([A-Za-z]{2})\]`) //match language tags
        res := regex.FindAllStringSubmatch(lang_list_str, -1)
        lang_list := make([]string, len(res))
        lang_text_pos := make([]int, len(res))
        if len(res) > 1 {
            for i, value := range res {
                if len(value) == 2 {
                    lang_list[i] = value[0]
					pos := strings.Index(to_publish_text, value[0])
					lang_text_pos[i] = pos

                }
            }
        } else {
            fmt.Println("no language tag")
        }

        input_lang_tag := strings.TrimSpace(next_stat.Text)[:4]

        output_text :=""
	    for i, pos := range lang_text_pos {
            split_text := ""
            if i+1 == len(lang_text_pos){
                split_text = to_publish_text[pos:]
            } else {
                split_text= to_publish_text[pos : lang_text_pos[i+1]]
            }
            if lang_list[i]==input_lang_tag {
                split_text= strings.TrimSpace(next_stat.Text)
            }
            if i==0 {
                output_text = strings.Trim(split_text, "\n")
            }else {
                output_text = output_text + fmt.Sprintf("\n\n%s", strings.Trim(split_text, "\n"))
            }

            //lang_content = lang_content + fmt.Sprintf("\n\n[%s] %s", key, value)

        }
        publish_str := fmt.Sprintf("%s:%s", lang_list_str,output_text)
        return true, publish_str
    } else {
        return false, fmt.Sprintf("wrong state. state: %s , next state: %s", stat.Name, next_stat.Name)
    }
}
