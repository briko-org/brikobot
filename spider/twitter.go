package spider

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
)

func collectText(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, buf)
	}
}

func trimContent(content string) string {
	start := 0
	end := len(content)
	content_runes := []rune(content)
	if content_runes[0] == []rune("“")[0] {
		start = 1
	}
	if content_runes[len(content_runes)-1] == []rune("”")[0] {
		end = len(content_runes) - 1
	}
	return string(content_runes[start:end])
}

func (spidermsg *SpiderMessage) FetchTweetContent(ch chan SpiderResponse) {
	u, err := url.Parse(spidermsg.URL)
	content := ""
	if err == nil {
		fetchUrl := fmt.Sprintf("%s://mobile.twitter.com%s", u.Scheme, u.Path)
		resp, err := http.Get(fetchUrl)
		defer resp.Body.Close()
		if err == nil {
			doc, err := html.Parse(resp.Body)
			if err == nil {
				var f func(*html.Node)
				f = func(n *html.Node) {
					if n.Type == html.ElementNode && n.Data == "div" {
						for _, p := range n.Attr {
							if p.Key == "dir" && p.Val == "ltr" {
								text := &bytes.Buffer{}
								collectText(n, text)
								content = trimContent(text.String())
								return
							}
						}
					}
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if len(content) == 0 {
							f(c)
						}
					}
				}
				f(doc)
			}
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
