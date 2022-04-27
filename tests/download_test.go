package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/bearki/beclient"
)

func TestDownload(t *testing.T) {
	urls := "https://apd-19a9e7921853375f63e3d0b7d4156ca2.v.smtcdns.com/moviets.tc.qq.com/AkBigrzeYUAq18DoqUtqiYl294nRhjCHLcOJ6lJDE4FQ/uwMROfz2r55goaQXGdGnC2de64gtX89GT746tcTJVnDpJgsD/svp_50112/nFNRIur563rwzXiv4agVeTlT1pSbqzL-RxK278t3sC5SIVd8aehN-TjZIyeSR2m-e8JOT_zKJjfQeQ0wT8BOFeOzNftoBVI8iH93rg8408xc3sfo_CK6cbut55gu3QPyTKTV1spaQ-kUshcH2cja56TE5WEtr450m9RRVyQjQq9A9S5ZbUJBQQ/gzc_1000102_0b53piaa4aaa7qao2qvjpjq4a6wdbz3aacsa.f321004.ts.m3u8?ver=4"
	err := beclient.New(urls, true).
		// Debug().
		TimeOut(time.Hour*10).
		// DownloadMultiThread(5, 1024*1024).
		Download("qqq.txt", func(currSize, totalSize float64) {
			fmt.Println("已下载：", int64(currSize), int64(totalSize))
		}).
		Get(nil)
	fmt.Println()
	fmt.Println("ERROR:", err)
}
