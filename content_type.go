/**
 * @Title资源类型定义
 * @Author Bearki
 * @DateTime 2021/11/27
 */

package beclient

// 资源类型变量类型
type ContentTypeType string

var (
	// XML格式
	ContentTypeTextXml ContentTypeType = "text/xml"
	// XML数据格式
	ContentTypeAppXml ContentTypeType = "application/xml"
	// JSON数据格式
	ContentTypeJson ContentTypeType = "application/json"
	// form表单数据被编码为key/value格式拼接到URL后
	ContentTypeFormURL ContentTypeType = "application/x-www-form-urlencoded"
	// 需要在表单中进行文件上传时，就需要使用该格式
	ContentTypeFormBody ContentTypeType = "multipart/form-data"
)
