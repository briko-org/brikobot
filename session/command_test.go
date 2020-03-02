package session

import ("testing"
"fmt"
)

func TestNewCommand(t *testing.T) {
    var u_id int = 1001
    var chat_id int64 = 9000001
    cmd := New(u_id, chat_id)
    if cmd.U_id != u_id {
        t.Errorf("new Command failed, expected %v , got %v", u_id, cmd.U_id)
    } else {
        t.Logf("new Command.U_id success");
    }
    if cmd.Chat_id != chat_id {
        t.Errorf("new Command failed, expected %v , got %v", chat_id, cmd.Chat_id)
    } else {
        t.Logf("new Command.Chat_id success");
    }
}

func TestNextCommand(t *testing.T) {
    var u_id int = 1001
    var chat_id int64 = 9000001
    var command = "NEW"
    var text= ""
    cmd := New(u_id, chat_id)
    cmd.Command = command
    cmd.Text = text


    //var next_command = "INPUT"
    //var next_text= "some test string"
    //next_cmd := New(u_id, chat_id)
    //next_cmd.Command = next_command
    //next_cmd.Text = next_text
    //cmd.NextUpdate(next_cmd)

    response := cmd.Response()
    fmt.Printf("===============test, response %v\n",response)
    //t.Logf("====Command %v response: %v\n", cmd.Command, response );
}
