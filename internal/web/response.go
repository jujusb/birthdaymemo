package web

import (
	"encoding/json"
	"net/http"
)

// API 统一响应码
const (
	CodeOK           = 0
	CodeBadRequest   = 1
	CodeUnauthorized = 2
	CodeForbidden    = 3
	CodeNotFound     = 4
	CodeInternal     = 5
)

// Response 统一响应结构
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK 成功响应
func OK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, Response{Code: CodeOK, Message: "", Data: data})
}

// OKMsg 带消息的成功响应
func OKMsg(w http.ResponseWriter, msg string, data any) {
	writeJSON(w, http.StatusOK, Response{Code: CodeOK, Message: msg, Data: data})
}

// Fail 失败响应
func Fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, http.StatusOK, Response{Code: code, Message: msg, Data: nil})
}

// FailWithData 失败响应并携带附加数据（如 needs_captcha 标志）
func FailWithData(w http.ResponseWriter, code int, msg string, data any) {
	writeJSON(w, http.StatusOK, Response{Code: code, Message: msg, Data: data})
}

// FailStatus 失败响应并指定 HTTP 状态码
func FailStatus(w http.ResponseWriter, httpStatus, code int, msg string) {
	writeJSON(w, httpStatus, Response{Code: code, Message: msg, Data: nil})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
