package spider

type SpiderMessage struct {
	Chat_id int64
	U_id    int
	URL     string
}

type SpiderResponse struct {
	Chat_id int64
	U_id    int
	Content string
	Url     string
	Err     error
}
