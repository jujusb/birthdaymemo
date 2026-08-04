package web

import (
	"net/http"

	"github.com/mcbill1/birthdaymemo/internal/i18n"
)

// languagesData 公共语言列表响应
type languagesData struct {
	Available []i18n.Language `json:"available"`
	Default   string          `json:"default"`
}

// handleLanguages 返回可用语言列表与默认语言
func (s *Server) handleLanguages(w http.ResponseWriter, r *http.Request) {
	OK(w, languagesData{Available: i18n.Available(), Default: s.cfg.Language})
}

// i18nData 翻译响应
type i18nData struct {
	Lang    string            `json:"lang"`
	Strings map[string]string `json:"strings"`
}

// handleI18n 返回指定语言的全部翻译字符串，供前端使用
func (s *Server) handleI18n(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = s.cfg.Language
	}
	OK(w, i18nData{Lang: lang, Strings: i18n.Strings(lang)})
}
