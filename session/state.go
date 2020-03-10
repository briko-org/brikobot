package session

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	MsgType            string            `json:"msgType"`
	MsgID              string            `json:"msgID"`
	MsgFlag            string            `json:"msgFlag"`
	MsgIDRespondTo     string            `json:"msgIDRespondTo"`
	TranslationResults map[string]string `json:"translationResults"`
}

func makeMenu(state_list []string) string {
	var menu string
	for _, value := range state_list {
		if value != "TRANSLATE" {
			menu += fmt.Sprintf("/%s\n", strings.ToLower(value))
		}
	}
	return menu
}

func (stat *State) Response(nextstat *State) string {
	var msg string
	stat_list := stat.NextState()
	for _, name := range stat_list {
		if name == "TRANSLATE" {
			msg = "Waiting for BRIKO AI translate"
		}
	}

	if nextstat.Name == stat.Name {
		msg = msg + "\nshow menu" + "\n" + makeMenu(stat_list)
	} else {
		if nextstat.Name == "TRANSLATE" && stat.Name == "INPUT" {
			msg = "Waiting for BRIKO AI translate"
		} else if nextstat.Name == "NEW" {
			msg = "new task,\nmenu" + "\n" + makeMenu(stat_list)
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
		state_list = append(state_list, "PUBLISH")
	case "IMPROVE":
		state_list = append(state_list, "SUBMIT")
	case "SUBMIT":
	case "PUBLISH":
		state_list = append(state_list, "NEW")
	}
	return state_list
}

func (stat *State) RequestBriko(ch chan State) {
	//'{"msgType": "Translation", "msgID" : "BOT00012234", "sourceLang" : "EN", "requestLang" : ["JA", "ZH"], "sourceContent" : "I have an apple."}'
	//Name string
	//Text string
	//U_id int
	//Chat_id int64

	lang_list := [3]string{"EN", "JA", "ZH"}

	data := &requestMsg{
		MsgType:    "Translation",
		MsgID:      "BOT00012234",
		SourceLang: "EN",
	}

	regex := *regexp.MustCompile(`\[([A-Za-z]{2})\]`)
	res := regex.FindStringSubmatch(stat.Text)
	if len(res) > 1 {
		data.SourceLang = strings.ToUpper(res[1])
		data.SourceContent = string(stat.Text[4:])
		requestLang := []string{}
		for _, value := range lang_list {
			if value != data.SourceLang {
				requestLang = append(requestLang, value)
			}
		}
		data.RequestLang = requestLang
		output, _ := json.Marshal(data)
		resp, _ := http.Post("http://localhost:8080/t", "application/json", bytes.NewBuffer(output))

		d := json.NewDecoder(resp.Body)
		rmsg := &responseMsg{}
		err := d.Decode(rmsg)
		if err != nil {
			fmt.Println(err) //TODO: send the error msg to bot
		} else {
			if rmsg.MsgFlag == "success" {
				lang_content := ""
				lang_list_str := fmt.Sprintf("[%s]", data.SourceLang)
				translation := rmsg.TranslationResults
				for key, value := range translation {
					if len(lang_content) > 0 {
						lang_content = lang_content + fmt.Sprintf("\n[%s]%s", key, value)
					} else {
						lang_content = lang_content + fmt.Sprintf("[%s]%s", key, value)
					}
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
