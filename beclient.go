package beclient

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ajg/form"
)

// BeClient 客户端控制器
type BeClient struct {
	baseURL              string                   // 基本网址Host Name
	disabledBaseURLParse bool                     // 是否禁用基础地址解析
	pathURL              string                   // 路由地址
	contentType          ContentTypeType          // 资源类型（会根据该类型来格式化请求参数，默认值：application/json）
	headers              sync.Map                 // 请求头
	cookies              sync.Map                 // 请求Cookie
	querys               url.Values               // 路由地址后的追加参数
	method               MethodType               // 请求方法类型
	data                 interface{}              // 请求参数
	isDownloadRequest    bool                     // 是否为下载请求
	downloadSavePath     string                   // 下载资源保存路径
	downloadCallFunc     DownloadCallbackFuncType // 下载进度回调函数
	timeOut              time.Duration            // 请求及响应的超时时间
	client               *http.Client             // HTTP客户端
	request              *http.Request            // 请求体
	response             *http.Response           // 响应体
	debug                bool                     // 是否Debug输出，（输出为json格式化后的数据）
	errMsg               error                    // 错误信息
}

// DownloadCallbackFuncType 下载内容回调方法类型
// @params currSize  int64 当前已下载大小
// @params totalSize int64 资源内容总大小
type DownloadCallbackFuncType func(currSize, totalSize float64)

// New 创建一个基础客户端
// @params baseURL              string    基础访问地址
// @params disabledBaseURLParse ...bool   是否禁用基础地址解析
// @return                      *BeClient 客户端指针
func New(baseURL string, disabledBaseURLParse ...bool) *BeClient {
	// 新建客户端
	c := new(BeClient)
	// 初始化默认超时时间为15秒
	c.TimeOut(time.Second * 15)
	// 初始化默认资源类型
	c.ContentType(ContentTypeJson)
	// 是否禁用地址解析
	if len(disabledBaseURLParse) > 0 && disabledBaseURLParse[0] {
		c.disabledBaseURLParse = true
		c.baseURL = baseURL
		return c
	}
	// 处理基本网址
	u, err := url.Parse(baseURL)
	if err != nil {
		c.errMsg = err
		return c
	}
	// 截取协议，域名或IP，端口号三部分
	c.baseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	// 判断是否需要赋值path
	if len(u.Path) > 0 {
		c.Path(u.Path)
	}
	// 初始化query参数
	c.querys = make(url.Values)
	// 判断是否需要处理URL参数
	query := u.Query()
	if len(query) > 0 {
		for key := range query {
			c.Query(key, query.Get(key))
		}
	}
	// 返回创建的客户端
	return c
}

// Path 路由地址
// @Desc 多次调用会拼接地址
// @params pathURL string    路由地址
// @return         *BeClient 客户端指针
func (c *BeClient) Path(pathURL string) *BeClient {
	// 禁用地址解析后该函数禁用
	if c.disabledBaseURLParse {
		return c
	}
	// 根据第一个问号分割字符串
	index := strings.Index(pathURL, "?")
	if index > -1 {
		// 裁剪字符串
		pathURL = pathURL[:index]
		query := pathURL[index+1:]
		// 解析参数
		queryMap := make(map[string]string)
		err := form.DecodeString(&queryMap, query)
		if err != nil {
			c.errMsg = err
			return c
		}
		index := strings.Index(pathURL, "?")
		if index > -1 {
			// 裁剪字符串
			query := pathURL[index-1:]
			pathURL = pathURL[:index]
			// 解析参数
			queryMap := make(map[string]string)
			err := form.DecodeString(&queryMap, query)
			if err != nil {
				c.errMsg = err
				return c
			}
			// 遍历赋值参数
			for key, val := range queryMap {
				c.Query(key, val)
			}
		}
	} else {
		// 不需要处理参数
		c.pathURL += pathURL
	}
	return c
}

// Header 配置请求头
// @Desc 多次调用相同KEY的值会被覆盖
// @params key string    请求头名称
// @params val string    请求同内容
// @return     *BeClient 客户端指针
func (c *BeClient) Header(key, val string) *BeClient {
	c.headers.Store(key, val)
	return c
}

// Cookie 配置请求Cookie
// @Desc 多次调用相同KEY的值会被覆盖
// @params key string    Cookie名称
// @params val string    Cookie内容
// @return     *BeClient 客户端指针
func (c *BeClient) Cookie(key, val string) *BeClient {
	c.cookies.Store(key, val)
	return c
}

// Query 配置请求URL后参数
// @Desc 多次调用相同KEY的值会被覆盖
// @params key string    参数名称
// @params val string    参数内容
// @return     *BeClient 客户端指针
func (c *BeClient) Query(key, val string) *BeClient {
	c.querys.Set(key, val)
	return c
}

// ContentType 配置资源类型
// @Desc 请求体将会被格式化为该资源类型
// @return *BeClient 客户端指针
func (c *BeClient) ContentType(contentType ContentTypeType) *BeClient {
	c.contentType = ContentTypeType(contentType)
	return c
}

// Debug 开启Debug模式
// @Desc 打印的是JSON格式化后的数据
// @return *BeClient 客户端指针
func (c *BeClient) Debug() *BeClient {
	c.debug = true
	return c
}

// TimeOut 配置请求及响应的超时时间
// @params reqTimeOut time.Duration 请求超时时间
// @return            *BeClient     客户端指针
func (c *BeClient) TimeOut(timeOut time.Duration) *BeClient {
	c.timeOut = timeOut
	return c
}

// Body 配置Body请求参数
// @Desc 会根据Content-Type来格式化数据
// @params data interface{} 请求参数
// @return      *BeClient   客户端指针
func (c *BeClient) Body(reqBody interface{}) *BeClient {
	c.data = reqBody
	return c
}

// Download 标记当前请求为下载类型
// @Desc 使用该接口注册回调可实现下载进度功能
// @params savePath string                   下载资源文件保存路径
// @params callback DownloadCallbackFuncType 下载进度回调函数(请勿在回调函数内处理过多业务，否则可能会造成响应超时)
// @return          *BeClient                客户端指针
func (c *BeClient) Download(savePath string, callback DownloadCallbackFuncType) *BeClient {
	c.isDownloadRequest = true
	c.downloadSavePath = filepath.Join(savePath) // 处理一下传入的路径
	c.downloadCallFunc = callback
	return c
}

// Get GET请求
// @Desc 第二个参数没有时，第一个参数必须为[]byte类型，将不对响应内容做转换处理
// @params resData        interface{}        指定类型的变量指针
// @params resContentType ...ContentTypeType 规定的响应内容资源类型，将会根据该类型对响应内容做转换
// @return                error              错误信息
func (c *BeClient) Get(resData interface{}, resContentType ...ContentTypeType) error {
	// 赋值请求类型
	c.method = MethodGet
	// 发起请求
	return c.send(resData, resContentType...)
}

// Head HEAD请求
// @Desc 第二个参数没有时，第一个参数必须为[]byte类型，将不对响应内容做转换处理
// @params resData        interface{}        指定类型的变量指针
// @params resContentType ...ContentTypeType 规定的响应内容资源类型，将会根据该类型对响应内容做转换
// @return                error              错误信息
func (c *BeClient) Head(resData interface{}, resContentType ...ContentTypeType) error {
	// 赋值请求类型
	c.method = MethodHead
	// 发起请求
	return c.send(resData, resContentType...)
}

// Post POST请求
// @Desc 第二个参数没有时，第一个参数必须为[]byte类型，将不对响应内容做转换处理
// @params resData        interface{}        指定类型的变量指针
// @params resContentType ...ContentTypeType 规定的响应内容资源类型，将会根据该类型对响应内容做转换
// @return                error              错误信息
func (c *BeClient) Post(resData interface{}, resContentType ...ContentTypeType) error {
	// 赋值请求类型
	c.method = MethodPost
	// 发起请求
	return c.send(resData, resContentType...)
}

// Put PUT请求
// @Desc 第二个参数没有时，第一个参数必须为[]byte类型，将不对响应内容做转换处理
// @params resData        interface{}        指定类型的变量指针
// @params resContentType ...ContentTypeType 规定的响应内容资源类型，将会根据该类型对响应内容做转换
// @return                error              错误信息
func (c *BeClient) Put(resData interface{}, resContentType ...ContentTypeType) error {
	// 赋值请求类型
	c.method = MethodPut
	// 发起请求
	return c.send(resData, resContentType...)
}

// Patch PATCH请求
// @Desc 第二个参数没有时，第一个参数必须为[]byte类型，将不对响应内容做转换处理
// @params resData        interface{}        指定类型的变量指针
// @params resContentType ...ContentTypeType 规定的响应内容资源类型，将会根据该类型对响应内容做转换
// @return                error              错误信息
func (c *BeClient) Patch(resData interface{}, resContentType ...ContentTypeType) error {
	// 赋值请求类型
	c.method = MethodPatch
	// 发起请求
	return c.send(resData, resContentType...)
}

// Delete DELETE请求
// @Desc 第二个参数没有时，第一个参数必须为[]byte类型，将不对响应内容做转换处理
// @params resData        interface{}        指定类型的变量指针
// @params resContentType ...ContentTypeType 规定的响应内容资源类型，将会根据该类型对响应内容做转换
// @return                error              错误信息
func (c *BeClient) Delete(resData interface{}, resContentType ...ContentTypeType) error {
	// 赋值请求类型
	c.method = MethodDelete
	// 发起请求
	return c.send(resData, resContentType...)
}

// Options OPTIONS请求
// @Desc 第二个参数没有时，第一个参数必须为[]byte类型，将不对响应内容做转换处理
// @params resData        interface{}        指定类型的变量指针
// @params resContentType ...ContentTypeType 规定的响应内容资源类型，将会根据该类型对响应内容做转换
// @return                error              错误信息
func (c *BeClient) Options(resData interface{}, resContentType ...ContentTypeType) error {
	// 赋值请求类型
	c.method = MethodOptions
	// 发起请求
	return c.send(resData, resContentType...)
}

// Trace Trace请求
// @Desc 第二个参数没有时，第一个参数必须为[]byte类型，将不对响应内容做转换处理
// @params resData        interface{}        定类型的变量指针
// @params resContentType ...ContentTypeType 规定的响应内容资源类型，将会根据该类型对响应内容做转换
// @return                error              错误信息
func (c *BeClient) Trace(resData interface{}, resContentType ...ContentTypeType) error {
	// 赋值请求类型
	c.method = MethodTrace
	// 发起请求
	return c.send(resData, resContentType...)
}

// GetRequest 获取请求体
// @return *http.Request 请求体
func (c *BeClient) GetRequest() *http.Request {
	return c.request
}

// GetResponse 获取响应体
// @return *http.Response 响应体
func (c *BeClient) GetResponse() *http.Response {
	return c.response
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
	// 创建客户端
	c.createClient()
	// 创建请求体
	err := c.createRequest()
	if err != nil {
		return err
	}
	// 结束时释放请求体
	defer c.request.Body.Close()
	// 发起请求
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
	// 判断是否为下载请求
	if c.isDownloadRequest { // 下载请求
		// 判断是否请求成功
		if res.StatusCode != http.StatusOK {
			errBody, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf(string(errBody))
		}
		// 判断是否需要保存到文件中
		if len(c.downloadSavePath) >= 0 { // 需要保存到文件中
			// 判断文件夹部分是否为空
			saveDir := filepath.Dir(c.downloadSavePath)
			if len(saveDir) > 0 {
				// 创建文件夹部分
				err = os.MkdirAll(saveDir, 0755)
				if err != nil {
					return err
				}
			}
			// 打开文件，边下载边写入，防止内存占用高
			file, err := os.OpenFile(c.downloadSavePath, os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0666)
			if err != nil {
				return err
			}
			// 延迟关闭文件
			defer file.Close()
			// 定义已下载的大小
			var currSize int
			// 读取响应
			for {
				// 定义临时变量（每次读5MB）
				temp := make([]byte, 1024*5)
				// 读取响应体
				size, err := res.Body.Read(temp)
				// 追加已下载大小
				currSize += size
				// 判断读取是否正确
				if err != nil {
					// 读取结束，跳出循环
					if err == io.EOF {
						// 写入到文件
						n, err := file.Write(temp[:size])
						// 判断是否写入正确
						if err != nil {
							return err
						}
						if n != size {
							return errors.New("write to file byte length inconsistency")
						}
						// 回调进度
						c.downloadCallFunc(float64(currSize), float64(res.ContentLength))
						// 跳出循环
						break
					}
					// 有错误，直接返回错误
					return err
				}
				// 写入到文件
				n, err := file.Write(temp[:size])
				// 判断是否写入正确
				if err != nil {
					return err
				}
				if n != size {
					return errors.New("write to file byte length inconsistency")
				}
				// 回调进度
				c.downloadCallFunc(float64(currSize), float64(res.ContentLength))
			}
			// 下载成功
			return nil
		}
		// 没有保存路径，直接报错
		return errors.New("download file save path is null")
	}
	// 普通请求
	// 直接获取全部内容
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	// 转换响应内容，结束请求
	return c.responseConverData(resBody, resData, resContentType...)
}

// createClient 创建HTTP客户端
// @return *BeClient 客户端指针
func (c *BeClient) createClient() *BeClient {
	// 初始化客户端
	c.client = &http.Client{
		Transport:     http.DefaultClient.Transport,
		CheckRedirect: http.DefaultClient.CheckRedirect, // 检查重定向
		Jar:           http.DefaultClient.Jar,
		Timeout:       c.timeOut,
	}
	return c
}

// createRequest 创建请求体
// @return error 错误信息
func (c *BeClient) createRequest() error {
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
	c.request, err = http.NewRequest(string(c.method), c.baseURL+c.pathURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	// 配置请求头
	c.headers.Range(func(key, val interface{}) bool {
		c.request.Header.Set(key.(string), val.(string))
		return true
	})
	// 配置Cookie
	c.cookies.Range(func(key, val interface{}) bool {
		c.request.AddCookie(&http.Cookie{
			Name:  key.(string),
			Value: val.(string),
		})
		return true
	})
	// 返回空错误
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
func (c *BeClient) responseConverData(src []byte, dst interface{}, resContentType ...ContentTypeType) error {
	// 判断响应内容是否为空
	if len(src) == 0 {
		return nil
	}

	// 判断是否需要接收响应内容
	if dst == nil { // 不需要接收响应内容
		return nil
	}

	// 判断是否需要对响应内容做转换
	if len(resContentType) == 0 { // 不需要转换
		// 强制断言为byte指针
		dsts := dst.(*[]byte)
		*dsts = src
		return nil
	}

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

	// 默认不转换，直接赋值
	dsts := dst.(*[]byte)
	*dsts = src

	// 结束
	return nil
}
