package spider

import (
	"golang.org/x/net/html"
    "strings"
	"net/http"
)

func trimContent(content string) string{
    start := 0
    end := len(content)
    content_runes := []rune(content)
    if content_runes[0] == []rune("“")[0]{
        start=1
    }
    if content_runes[len(content_runes)-1] == []rune("”")[0]{
        end=len(content_runes)-1
    }
    return string(content_runes[start:end])
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
                                    content = strings.Replace(content, "&amp;", "&", -1)
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
