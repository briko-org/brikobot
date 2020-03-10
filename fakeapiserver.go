package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

//{
//    "msgType": "Translation",
//    "msgID" : "BOT00012234",
//    "sourceLang" : "EN",
//    "requestLang" : ["JA", "ZH"]
//    "sourceContent" : "I have an apple."
//}

//{
//    "msgType": "RespondTranslation",
//    "msgID" : "TRANS00012",
//    "msgIDRespondTo" : "BOT00012234",
//    "msgFlag" : "success",
//    "translationResults" : {
//                "ZH": "我有一个苹果。",
//                "JA": "りんごを持っています。"
//               }
//}

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

func main() {
	http.HandleFunc("/t", TranslateServer)
	http.HandleFunc("/", HelloServer)
	http.ListenAndServe(":8080", nil)
}

//case "POST":
func TranslateServer(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Fprintf(w, "TranslateServer")
	case "POST":
		// Decode the JSON in the body and overwrite 'tom' with it
		d := json.NewDecoder(r.Body)
		m := &requestMsg{}
		err := d.Decode(m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		translation := map[string]string{"ZH": "我有一个苹果。", "JA": "りんごを持っています。"}
		result := &responseMsg{
			MsgType:            m.MsgType,
			MsgID:              "TRANS00012",
			MsgIDRespondTo:     m.MsgID,
			MsgFlag:            "success",
			TranslationResults: translation,
		}
		output, _ := json.Marshal(result)
		fmt.Fprintf(w, "%s", output)
		//tom = p
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Not allowed")
	}
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
	input := r.URL.Query().Get("input")
	fmt.Println("input:" + input)
	time.Sleep(time.Second * 3)
	output := "[EN][CN][JP]:[EN]英文:" + input + "\n[CN]中文:" + input + "\n[JP]日文:" + input
	fmt.Fprintf(w, "%s", output)
}
