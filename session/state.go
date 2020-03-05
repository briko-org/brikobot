package session
import ("fmt"
    "strings"
    "io/ioutil"
    "net/http"
    "net/url"
	"github.com/virushuo/brikobot/database"
)

type State struct {
	Name string
    Text string
	U_id int
    Chat_id int64
}

func New(u_id int, chat_id int64) *State{
    stat := new(State)
	stat.U_id = u_id
	stat.Chat_id = chat_id
    stat.Name = "NONE"
    stat.Text = ""
    return stat
}

func makeMenu(state_list []string) string{
    var menu string
    for _, value := range state_list {
        if value != "TRANSLATE" {
            menu += fmt.Sprintf("/%s\n",strings.ToLower(value))
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

    if nextstat.Name == stat.Name  {
        msg = msg + "\nshow menu"+ "\n" + makeMenu(stat_list)
    } else {
        if nextstat.Name == "TRANSLATE" && stat.Name == "INPUT" {
            msg = "Waiting for BRIKO AI translate"
        }else if nextstat.Name == "NEW" {
            msg = "new task,\nmenu"+ "\n" + makeMenu(stat_list)
        }
    }


    return msg
}

func (stat *State) NextUpdate(nextstat *State, db *database.Db) (bool, string){
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

	if if_allowed_transition  == true {
		//update
        commandtag, err := db.SetChatState(nextstat.Chat_id, nextstat.U_id , nextstat.Name, nextstat.Text)
        fmt.Println(commandtag)
        fmt.Println(err)
		return true, nextstat.Response(nextstat)
	}
	return false, stat.Response(nextstat)
}

func (stat *State) NextState() []string{
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

  Url, err := url.Parse("http://localhost:8080/t")
  if err != nil {
      panic("error url")
  }
  parameters := url.Values{}
  parameters.Add("input", stat.Text)
  Url.RawQuery = parameters.Encode()

  resp, _ := http.Get(Url.String())
  body, _ := ioutil.ReadAll(resp.Body)

  ch <- State{"TRANSLATE", string(body), stat.U_id, stat.Chat_id}
}
