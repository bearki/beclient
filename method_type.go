package beclient

import "net/http"

type MethodType string

var (
	// GET 请求
	MethodGet MethodType = http.MethodGet
	// HEAD 请求
	MethodHead MethodType = http.MethodHead
	// POST 请求
	MethodPost MethodType = http.MethodPost
	// PUT 请求
	MethodPut MethodType = http.MethodPut
	// PATCH 请求
	MethodPatch MethodType = http.MethodPatch
	// DELETE 请求
	MethodDelete MethodType = http.MethodDelete
	// OPTIONS 请求
	MethodOptions MethodType = http.MethodOptions
	// TRACE 请求
	MethodTrace MethodType = http.MethodTrace
)
