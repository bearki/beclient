package main

import (
	"fmt"
	"time"

	"github.com/bearki/beclient"
)

func main() {
	urls := "https://easydoc.net/mock/u/82910681/test"
	b := make(map[string]interface{})
	fmt.Println(beclient.New(urls).
		TimeOut(time.Hour*10).
		// Debug().
		// Download("qqq.mp4", func(currSize, totalSize float64) {
		// 	fmt.Println("下载进度:", int64((currSize/totalSize)*100), "%")
		// }).
		Delete(&b, beclient.ContentTypeJson),
	)
	fmt.Println(b)
}
