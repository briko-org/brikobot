package session

import (
	"fmt"
	"testing"
)

func TestNewState(t *testing.T) {
	var u_id int = 1001
	var chat_id int64 = 9000001
	stat := New(u_id, chat_id)
	if stat.U_id != u_id {
		t.Errorf("new State failed, expected %v , got %v", u_id, stat.U_id)
	} else {
		t.Logf("new State .U_id success")
	}
	if stat.Chat_id != chat_id {
		t.Errorf("new State failed, expected %v , got %v", chat_id, stat.Chat_id)
	} else {
		t.Logf("new State.Chat_id success")
	}
}

func TestNextState(t *testing.T) {
	var u_id int = 1001
	var chat_id int64 = 9000001
	var name = "NEW"
	var text = ""
	stat := New(u_id, chat_id)
	stat.Name = name
	stat.Text = text

	//var next_command = "INPUT"
	//var next_text= "some test string"
	//next_cmd := New(u_id, chat_id)
	//next_cmd.Command = next_command
	//next_cmd.Text = next_text
	//cmd.NextUpdate(next_cmd)

	response := stat.Response()
	fmt.Printf("===============test, response %v\n", response)
	//t.Logf("====Command %v response: %v\n", cmd.Command, response );
}
