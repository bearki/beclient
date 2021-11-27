package main

import (
	"fmt"
	"time"

	"github.com/bearki/beclient"
)

func main() {
	urls := "http://speedtest.dallas.linode.com/100MB-dallas.bin"
	b := make(map[string]interface{})
	fmt.Println(beclient.New(urls).
		TimeOut(time.Hour*10).
		// Debug().
		Download("qqq.bin", func(currSize, totalSize float64) {
			fmt.Println("下载进度:", int64((currSize/totalSize)*100), "%")
		}).
		Get(nil),
	)
	fmt.Println(b)
}
