package main

import (
    "fmt"
    "time"
    "net/http"
)

func main() {
    http.HandleFunc("/", HelloServer)
    http.ListenAndServe(":8080", nil)
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
    input := r.URL.Query().Get("input")
    fmt.Println("input:"+input)
    time.Sleep(time.Second * 3)
    output := "[EN][CN][JP]:[EN]英文:"+input+"\n[CN]中文:"+input+"\n[JP]日文:"+input
    fmt.Fprintf(w, "%s", output)
}
