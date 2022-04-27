package beclient

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// New 创建一个基础客户端
// @params baseURL              string    基础访问地址
// @params disabledBaseURLParse ...bool   是否禁用基础地址解析
// @return                      *BeClient 客户端指针
func New(baseURL string, disabledBaseURLParse ...bool) *BeClient {
	// 新建客户端
	c := new(BeClient)
	// 初始化query参数
	c.querys = make(url.Values)
	// 初始化默认下载缓冲区
	c.DownloadBufferSize(1024 * 1024 * 5)
	// 配置多线程下载参数
	c.DownloadMultiThread(20, 1024*1024*100)
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
	// 解析基本网址
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
	// 判断是否需要处理URL参数
	for key := range u.Query() {
		c.Query(key, u.Query().Get(key))
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
		c.errMsg = errors.New("disabled base url parse is open, Path function disabled")
		return c
	}
	// 根据第一个问号分割字符串
	index := strings.Index(pathURL, "?")
	if index > -1 {
		// 截取路径部分
		c.pathURL += pathURL[:index]
		// 判断是否需要处理参数部分
		if len(pathURL[index:]) > 1 {
			// 截取参数部分
			query := pathURL[index+1:]
			// 解析参数
			values, err := url.ParseQuery(query)
			if err != nil {
				c.errMsg = err
				return c
			}
			// 遍历赋值参数
			for key := range values {
				c.Query(key, values.Get(key))
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
	c.contentType = contentType
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
	c.downloadSavePath = filepath.Join(strings.TrimSpace(savePath))
	c.downloadCallFunc = callback
	return c
}

// DownloadBufferSize 配置下载缓冲区大小
// @params size int64     下载缓冲区大小（单位：byte）（默认1024*1024*5byte[5MB]，最小5byte[5B]，最大1024*1024*1024byte[1GB]）
// @return      *BeClient 客户端指针
func (c *BeClient) DownloadBufferSize(size int64) *BeClient {
	// 最低不能低于5byte
	if size < 5 || size > 1024*1024*1024 {
		return c
	}
	// 赋值全局
	c.downloadBufferSize = size
	return c
}

// DownloadMultiThread 配置多线程下载参数
// @params maxThreadNum    int64     最大下载线程数量（默认20）
// @params maxDownloadSize int64     单个线程最大下载容量，仅在使用线程数低于最大线程数时有效（默认1024*1024*100byte[100MB]，最小5byte[5B]，最大1024*1024*1024*10byte[10GB]）
// @return                 *BeClient 客户端指针
func (c *BeClient) DownloadMultiThread(maxThreadNum, maxDownloadSize int64) *BeClient {
	// 至少要一个线程
	if maxThreadNum < 1 {
		maxThreadNum = c.downloadMaxThread
	}
	// 最低不能低于5byte并且不能大于10GB
	if maxDownloadSize < 5 || maxDownloadSize > 1024*1024*1024*10 {
		// 默认值
		maxDownloadSize = c.downloadMaxSize
	}
	// 赋值到全局
	c.downloadMaxThread = maxThreadNum
	c.downloadMaxSize = maxDownloadSize
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

// GetHttpClient 获取HTTP客户端
// @return *http.Client HTTP客户端
// @return error        错误信息
func (c *BeClient) GetHttpClient() (*http.Client, error) {
	// 判断是否已经构建
	if c.client != nil {
		return c.client, nil
	}
	// 执行构建
	err := c.build()
	if err != nil {
		return nil, err
	}
	// 响应
	return c.client, nil
}

// GetRequest 获取HTTP请求体
// @return *http.Request HTTP请求体
// @return error         错误信息
func (c *BeClient) GetRequest() (*http.Request, error) {
	// 判断是否已经构建
	if c.request != nil {
		return c.request, nil
	}
	// 执行构建
	err := c.build()
	if err != nil {
		return nil, err
	}
	// 响应
	return c.request, nil
}

// GetResponse 获取HTTP响应体
// @return *http.Response HTTP响应体
// @return error          错误信息
func (c *BeClient) GetResponse() (*http.Response, error) {
	// 判断响应体是否为空
	if c.response == nil {
		return nil, errors.New("please send an HTTP request first")
	}
	// 响应
	return c.response, nil
}
