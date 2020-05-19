package plugins

import (
	gotwitter "github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

type Twitterclient struct {
	Client *gotwitter.Client
}

func NewTwitter(CONSUMER_KEY string, CONSUMER_SECRET string, ACCESSTOKEN_TOKEN string, ACCESSTOKEN_SECRET string) *Twitterclient {
	config := oauth1.NewConfig(CONSUMER_KEY, CONSUMER_SECRET)
	token := oauth1.NewToken(ACCESSTOKEN_TOKEN, ACCESSTOKEN_SECRET)
	httpClient := config.Client(oauth1.NoContext, token)
	tc := new(Twitterclient)
	tc.Client = gotwitter.NewClient(httpClient)
	return tc
}
