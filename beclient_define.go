package beclient

import (
	"net/http"
	"net/url"
	"sync"
	"time"
)

// MethodType 请求类型
type MethodType string

const (
	// MethodGet GET请求
	MethodGet MethodType = http.MethodGet
	// MethodHead HEAD请求
	MethodHead MethodType = http.MethodHead
	// MethodPost POST请求
	MethodPost MethodType = http.MethodPost
	// MethodPut PUT请求
	MethodPut MethodType = http.MethodPut
	// MethodPatch PATCH请求
	MethodPatch MethodType = http.MethodPatch
	// MethodDelete DELETE请求
	MethodDelete MethodType = http.MethodDelete
	// MethodOptions OPTIONS请求
	MethodOptions MethodType = http.MethodOptions
	// MethodTrace TRACE请求
	MethodTrace MethodType = http.MethodTrace
)

// ContentTypeType 资源类型变量类型
type ContentTypeType string

const (
	// ContentTypeTextXml XML格式
	ContentTypeTextXml ContentTypeType = "text/xml"
	// ContentTypeAppXml XML数据格式
	ContentTypeAppXml ContentTypeType = "application/xml"
	// ContentTypeJson JSON数据格式
	ContentTypeJson ContentTypeType = "application/json"
	// ContentTypeFormURL form表单数据被编码为key/value格式拼接到URL后
	ContentTypeFormURL ContentTypeType = "application/x-www-form-urlencoded"
	// ContentTypeFormBody 需要在表单中进行文件上传时，就需要使用该格式
	ContentTypeFormBody ContentTypeType = "multipart/form-data"
)

// DownloadCallbackFuncType 下载内容回调方法类型
// @params currSize  int64 当前已下载大小
// @params totalSize int64 资源内容总大小
type DownloadCallbackFuncType func(currSize, totalSize float64)

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
	downloadBufferSize   int64                    // 下载缓冲区大小（默认1024*5byte，最小5byte，最大1024*1024byte）
	downloadMaxThread    int64                    // 最大下载线程数量（默认20）
	downloadMaxSize      int64                    // 单个线程最大下载容量，仅在使用线程数低于最大线程数时有效（默认1024*100byte，最小5byte，最大1024*1024*10byte）
	downloadSavePath     string                   // 下载资源保存路径
	downloadCallFunc     DownloadCallbackFuncType // 下载进度回调函数
	timeOut              time.Duration            // 请求及响应的超时时间
	client               *http.Client             // HTTP客户端
	request              *http.Request            // 请求体
	response             *http.Response           // 响应体
	debug                bool                     // 是否Debug输出，（输出为json格式化后的数据）
	errMsg               error                    // 错误信息
}
