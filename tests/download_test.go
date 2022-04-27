package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/bearki/beclient"
)

func TestDownload(t *testing.T) {
	urls := "http://mirror.aarnet.edu.au/pub/TED-talks/911Mothers_2010W-480p.mp4"
	err := beclient.New(urls, true).
		// Debug().
		TimeOut(time.Hour*10).
		DownloadMultiThread(5, 1024*1024).
		Download("qqq.mp4", func(currSize, totalSize float64) {
			fmt.Println("已下载：", int64(currSize), int64(totalSize))
		}).
		Get(nil)
	fmt.Println()
	fmt.Println("ERROR:", err)
}
