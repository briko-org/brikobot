package spider

import (
	"fmt"
	"golang.org/x/net/html"
	"net/http"
)

func trimContent(content string) string{
    start := 0
    end := len(content)
    if content[0] == 226 {
        start=1
    }
    if content[len(content)-1] == 157 {
        end=len(content)-1
    }
    return string([]rune(content)[start:end])
}



func (spidermsg *SpiderMessage) FetchTweetContent(ch chan SpiderResponse) {
	resp, err := http.Get(spidermsg.URL)
	content := ""
	defer resp.Body.Close()
	if err == nil {
		doc, err := html.Parse(resp.Body)
		if err == nil {
			var f func(*html.Node)
			f = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "meta" {
					for _, p := range n.Attr {
						if p.Key == "property" && p.Val == "og:description" {
							for _, p1 := range n.Attr {
								if p1.Key == "content" {
									content = p1.Val
                                    content = trimContent(content)
									return
								}
							}
						}
					}
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			}
			f(doc)
		}
	}
	response := &SpiderResponse{
		Chat_id: spidermsg.Chat_id,
		U_id:    spidermsg.U_id,
		Content: content,
		Url:     spidermsg.URL,
		Err:     err,
	}
	ch <- *response
}
