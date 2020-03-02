package session
import ("fmt"
"strings"
)

type State struct {
	Command string
    Text string
	U_id int
    Chat_id int64
}

func New(u_id int, chat_id int64) *State{
    stat := new(State)
	stat.U_id = u_id
	stat.Chat_id = chat_id
    stat.Command = "NEW"
    stat.Text = ""
    return stat
}

func makeMenu(cmd_list []string) string{
    var menu string
    for _, value := range cmd_list {
        menu += fmt.Sprintf("/%s\n",strings.ToLower(value))
    }
    return menu
}

func (stat *State) Response() string {
    cmd_list := stat.NextCommands()
    if len(cmd_list) == 0 {
        return "no next cmd, reset to new"
    } else if len(cmd_list) >1 {
        return makeMenu(cmd_list)
    } else {
        return cmd_list[0]
    }
}

func (stat *State) NextUpdate(nextstat *State) bool{
    fmt.Printf("next update cmd : %v current cmd: %v \n", nextstat.Command, stat.Command)
    stat.NextCommands()
	return true;
}

func (stat *State) NextCommands() []string{
    var cmd_list []string
	switch stat.Command {
	    case "NEW":
	        cmd_list = append(cmd_list, "ABOUT")
	        cmd_list = append(cmd_list, "HELP")
	        cmd_list = append(cmd_list, "INPUT")
		case "INPUT":
	        cmd_list = append(cmd_list, "SETLANG")
		case "IMPROVE":
	        cmd_list = append(cmd_list, "SUBMIT")
    }
	cmd_list = append(cmd_list, "NEW")
    fmt.Printf("find the next command of stat.Command: %v\n", stat.Command)
    fmt.Printf("Command list: %v\n", cmd_list)
    return cmd_list
}
