package beclient

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ajg/form"
)

// build 构建HTTP客户端和HTTP请求体
func (c *BeClient) build() error {
	// 初始化客户端
	client := &http.Client{
		Transport:     http.DefaultClient.Transport,
		CheckRedirect: http.DefaultClient.CheckRedirect, // 检查重定向
		Jar:           http.DefaultClient.Jar,
		Timeout:       c.timeOut,
	}
	// 转换请求参数
	reqBody, err := c.requestConvertData()
	if err != nil {
		return err
	}
	// 先拼接URL参数
	if len(c.querys) > 0 {
		// 判断是否禁用了地址解析
		if c.disabledBaseURLParse {
			if strings.Contains(c.baseURL, "?") {
				c.baseURL += fmt.Sprintf("&%s", c.querys.Encode())
			} else {
				c.baseURL += fmt.Sprintf("?%s", c.querys.Encode())
			}
		} else {
			if strings.Contains(c.pathURL, "?") || strings.Contains(c.baseURL, "?") {
				c.pathURL += fmt.Sprintf("&%s", c.querys.Encode())
			} else {
				c.pathURL += fmt.Sprintf("?%s", c.querys.Encode())
			}
		}
	}
	// 创建请求体
	request, err := http.NewRequest(string(c.method), c.baseURL+c.pathURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	// 配置请求头
	c.headers.Range(func(key, val interface{}) bool {
		request.Header.Set(key.(string), val.(string))
		return true
	})
	// 配置Cookie
	c.cookies.Range(func(key, val interface{}) bool {
		request.AddCookie(&http.Cookie{
			Name:  key.(string),
			Value: val.(string),
		})
		return true
	})
	// 标记已经构建完成
	c.client = client
	c.request = request
	// 返回空错误
	return nil
}

// send 发送请求
// @Desc 任何请求均会通过该接口发出请求
// @return []byte 响应体Body内容
// @return error  错误信息
func (c *BeClient) send(resData interface{}, resContentType ...ContentTypeType) error {
	// 延迟判断是否需要Debug日志
	defer func() {
		// 是否需要打印Debug
		if c.debug {
			fmt.Printf("\n\nBeClient: %+v\n", *c)
			fmt.Printf("\nRequest: %+v\n", *c.request)
			fmt.Printf("\nResponse: %+v\n\n", *c.response)
		}
	}()
	// 是否有全局异常
	if c.errMsg != nil {
		return c.errMsg
	}
	// 判断是否已经构建
	if c.client == nil || c.request == nil {
		// 执行构建
		if err := c.build(); err != nil {
			return err
		}
	}
	// 结束时释放请求体
	defer c.request.Body.Close()
	// 判断是否为下载请求
	if c.isDownloadRequest {
		// 直接走下载请求接口
		return c.download()
	}
	// 普通请求,发起请求
	res, err := c.client.Do(c.request)
	// 判断是否请求错误
	if err != nil {
		return err
	}
	if res == nil {
		return errors.New("http.Response is nil pointer")
	}
	// 结束时释放响应体
	defer res.Body.Close()
	// 将响应体赋值到全局
	c.response = res
	// 直接获取全部内容
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	// 转换响应内容，结束请求
	return c.responseConvertData(resBody, resData, resContentType...)
}

// download 下载文件
func (c *BeClient) download() error {
	// 是否存在下载地址
	if len(c.downloadSavePath) == 0 {
		// 没有保存路径，直接报错
		return errors.New("download file save path is null")
	}
	// 判断文件夹部分是否为空
	saveDir := filepath.Dir(c.downloadSavePath)
	if len(saveDir) == 0 {
		return errors.New("download file save path dir is nil")
	}
	// 创建文件夹部分
	err := os.MkdirAll(saveDir, 0755)
	if err != nil {
		return err
	}
	// 缓存请求类型
	method := c.request.Method
	// 发送HEAD请求
	c.request.Method = string(MethodHead)
	headRes, err := c.client.Do(c.request)
	// 还原请求类型
	c.request.Method = method
	// 判断是否请求成功
	if err != nil {
		// 直接走单线程下载
		return c.singleThreadDownload(headRes)
	}
	if headRes == nil {
		// 直接走单线程下载
		return c.singleThreadDownload(headRes)
	}
	// 结束时释放
	defer headRes.Body.Close()
	// 判断是否请求成功
	if headRes.StatusCode != http.StatusOK {
		// 直接走单线程下载
		return c.singleThreadDownload(headRes)
	}
	// 判断大小是否小于缓冲区
	if headRes.ContentLength <= c.downloadBufferSize {
		// 直接走单线程下载
		return c.singleThreadDownload(headRes)
	}
	// 判断是否支持分片下载
	if !strings.Contains(headRes.Header.Get("Accept-Ranges"), "bytes") {
		// 直接走单线程下载
		return c.singleThreadDownload(headRes)
	}
	// 走多线程下载
	return c.multiThreadDownload(headRes)
}

// singleThreadDownload 单线程下载
func (c *BeClient) singleThreadDownload(headRes *http.Response) error {
	// 发送请求
	res, err := c.client.Do(c.request)
	if err != nil {
		return err
	}
	// 判断响应是否为空
	if res == nil {
		return errors.New("response is nil pointer address")
	}
	// 结束时释放
	defer res.Body.Close()
	// 赋值response
	c.response = res
	// 判断是否请求成功
	if res.StatusCode != http.StatusOK {
		// 将返回的错误信息读出
		errBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return errors.New(string(errBody))
	}

	// 打开文件，边下载边写入，防止内存占用高(os.O_TRUNC覆盖式写入)
	file, err := os.OpenFile(c.downloadSavePath, os.O_CREATE|os.O_WRONLY|os.O_SYNC|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	// 延迟关闭文件
	defer file.Close()

	// 定义下载缓冲区
	downBuffer := make([]byte, c.downloadBufferSize)
	// 已下载的大小
	var currSize int64
	// 读取响应
	for currSize < headRes.ContentLength {
		// 读取响应体
		size, err := res.Body.Read(downBuffer)
		// 追加已下载大小
		currSize += int64(size)
		// 判断是否发生错误
		if err != nil && err != io.EOF {
			// 有错误，直接返回错误
			return err
		}
		// 写入到文件
		n, err := file.Write(downBuffer[:size])
		// 判断是否写入正确
		if err != nil {
			return err
		}
		if n != size {
			return errors.New("write to file byte length inconsistency")
		}
		// 判断是否需要回调
		if c.downloadCallFunc != nil {
			// 回调进度
			c.downloadCallFunc(float64(currSize), float64(res.ContentLength))
		}
	}
	// 下载成功
	return nil
}

// multiThreadDownload 多线程下载
func (c *BeClient) multiThreadDownload(headRes *http.Response) error {
	// 根据大小计算需要的线程数
	var downloadThreadNum = headRes.ContentLength / c.downloadMaxSize
	// 有余数时需要加一个线程
	if headRes.ContentLength%c.downloadMaxSize != 0 {
		downloadThreadNum++
	}
	// 是否超过最大线程限制
	if downloadThreadNum > c.downloadMaxThread {
		// 重置为最大线程数
		downloadThreadNum = c.downloadMaxThread
	}
	// 根据计算出的下载线程数量计算每个线程的下载容量
	var singleDownloadSize = headRes.ContentLength / downloadThreadNum
	// 计算最后一个线程的下载量
	var lastThreadDownloadContent = singleDownloadSize + (headRes.ContentLength % downloadThreadNum)

	// 打开文件，边下载边写入，防止内存占用高(os.O_TRUNC覆盖式写入)
	file, err := os.OpenFile(c.downloadSavePath, os.O_CREATE|os.O_WRONLY|os.O_SYNC|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	// 延迟关闭文件
	defer file.Close()

	// 初始化可取消上下文
	globalCtx, globalCancel := context.WithTimeout(context.Background(), c.timeOut)
	// 初始化等待组
	var wg sync.WaitGroup

	// 已下载总量
	var downloadedSize int64

	// 遍历线程数
	var i int64
	for i = 0; i < downloadThreadNum; i++ {
		// 计算下载偏移量Range为闭区间
		offsetStart := i * singleDownloadSize
		offsetEnd := (i+1)*singleDownloadSize - 1
		// 是否为最后一个线程
		if i+1 == downloadThreadNum {
			offsetEnd = (offsetStart + lastThreadDownloadContent) - 1
		}
		// 开启协程下载
		wg.Add(1)
		go func(start, end int64) {
			defer wg.Done()
			// 拷贝request
			request := c.request.Clone(context.Background())
			// 请求Body需要单独拷贝
			body, err := c.requestConvertData()
			if err != nil {
				c.errMsg = err // 错误赋值到全局
				globalCancel() // 取消全部线程的下载
				return
			}
			request.Body = ioutil.NopCloser(bytes.NewReader(body))
			// 配置分段区域
			request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
			// 发送请求
			res, err := c.client.Do(request)
			if err != nil {
				c.errMsg = err // 错误赋值到全局
				globalCancel() // 取消全部线程的下载
				return
			}
			// 判断响应是否为空
			if res == nil {
				c.errMsg = errors.New("response is nil pointer address") // 错误赋值到全局
				globalCancel()                                           // 取消全部线程的下载
				return
			}
			// 结束时释放
			defer res.Body.Close()
			// 判断是否需要赋值response(至于赋值第几个response并不需要关心)
			if c.response == nil {
				// 赋值response
				c.response = res
			}
			// 判断是否请求成功
			if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusPartialContent {
				// 将返回的错误信息读出
				errBody, err := ioutil.ReadAll(res.Body)
				if err != nil {
					c.errMsg = err // 错误赋值到全局
				} else {
					c.errMsg = errors.New(string(errBody)) // 错误赋值到全局
				}
				globalCancel() // 取消全部线程的下载
				return
			}

			// 定义下载缓冲区
			downBuffer := make([]byte, c.downloadBufferSize)
			// 读取响应
			for start < end {
				select {
				case <-globalCtx.Done():
					return

				default:
					// 读取响应体
					size, err := res.Body.Read(downBuffer)
					// 判断是否发生错误
					if err != nil && err != io.EOF {
						c.errMsg = err // 错误赋值到全局
						globalCancel() // 取消全部线程的下载
						return
					}
					// 写入到文件
					n, err := file.WriteAt(downBuffer[:size], start)
					// 追加已下载大小
					start += int64(size)
					// 赋值到全局总量
					atomic.AddInt64(&downloadedSize, int64(size))
					// 判断是否写入正确
					if err != nil {
						c.errMsg = err // 错误赋值到全局
						globalCancel() // 取消全部线程的下载
						return
					}
					if n != size {
						c.errMsg = errors.New("write to file byte length inconsistency") // 错误赋值到全局
						globalCancel()                                                   // 取消全部线程的下载
						return
					}
					// 判断是否需要回调
					if c.downloadCallFunc != nil {
						// 回调进度
						c.downloadCallFunc(float64(atomic.LoadInt64(&downloadedSize)), float64(headRes.ContentLength))
					}
				}
			}
		}(offsetStart, offsetEnd)
	}
	// 等待全部线程下载完成
	wg.Wait()
	// 上下文结束
	globalCancel()
	// 判断是否有错误信息
	if c.errMsg != nil {
		// 下载失败
		return c.errMsg
	}
	if globalCtx.Err() != nil && globalCtx.Err() != context.Canceled {
		return globalCtx.Err()
	}
	// 下载成功
	return nil
}

// requestConvertData 根据请求资源类型转换请求数据
// @params data        interface{}     请求参数
// @params contentType ContentTypeType 资源类型
// @return reqBody     []byte          处理后的请求参数
// @return err         error           错误信息
func (c *BeClient) requestConvertData() (reqBody []byte, err error) {
	// 判断请求参数是否为空
	if c.data == nil {
		return nil, nil
	}

	// 根据资源类型处理请求参数
	switch c.contentType {

	// JSON数据
	case ContentTypeJson:
		return json.Marshal(c.data)

	// XML类型数据
	case ContentTypeTextXml, ContentTypeAppXml:
		return xml.Marshal(c.data)

	// URL后追加参数
	case ContentTypeFormURL:
		// 解析参数
		values, err := form.EncodeToValues(c.data)
		if err != nil {
			return nil, err
		}
		// 追加到全局Query参数
		for key := range values {
			c.Query(key, values.Get(key))
		}
		return nil, nil

	// 表单数据
	case ContentTypeFormBody:
		// 解析参数
		values, err := form.EncodeToValues(c.data)
		if err != nil {
			return nil, err
		}
		return []byte(values.Encode()), nil

	}

	return nil, errors.New("unsupported resource type")
}

// requestConvertData 根据响应资源类型转换响应数据
// @params src         []byte          响应内容
// @params dst         interface{}    用户自定义变量地址
// @params contentType ContentTypeType 资源类型
// @return err         error           错误信息
func (c *BeClient) responseConvertData(src []byte, dst interface{}, resContentType ...ContentTypeType) error {
	// 判断响应内容是否为空
	if len(src) == 0 {
		return nil
	}

	// 判断是否需要接收响应内容
	if dst == nil { // 不需要接收响应内容
		return nil
	}

	// 判断是否需要对响应内容做转换
	if len(resContentType) > 0 { // 需要转换
		// 取出资源类型
		contentType := resContentType[0]
		// 根据资源类型处理响应内容
		switch contentType {
		// JSON数据
		case ContentTypeJson:
			return json.Unmarshal(src, dst)
		// XML类型数据
		case ContentTypeTextXml, ContentTypeAppXml:
			return xml.Unmarshal(src, dst)
		// 表单数据
		case ContentTypeFormBody:
			// 解析参数
			return form.DecodeString(dst, string(src))
		}
	}
	// 默认不转换，直接赋值
	dstValue, ok := dst.(*[]byte)
	if ok {
		*dstValue = src
		return nil
	}
	// 结束
	return errors.New("dst is not *[]byte")
}
