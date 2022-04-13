package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/bearki/beclient"
)

func TestDownload(t *testing.T) {
	urls := "http://viss.sparker.xyz/20220411%2F92019b7bafd041409df795315f6061b0%2F20220411_0_92019b7bafd041409df795315f6061b0_0_0_5c9be60b3097307093e25680ea2a32f3.jpg?e=1649933578&token=nCBdvQS7iqRAhixysVQQC0OxNqYd4vEPGdnWWUNs:WCEz4KDMQZDZk9KlkyUZvLjIcwU="
	b := make(map[string]interface{})
	fmt.Println(beclient.New(urls, true).
		Debug().
		TimeOut(time.Hour*10).
		// Debug().
		Download("qqq.jpg", func(currSize, totalSize float64) {}).
		Get(nil),
	)
	fmt.Println(b)
}
