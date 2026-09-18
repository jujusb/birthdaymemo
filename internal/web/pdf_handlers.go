package web

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mcbill1/birthdaymemo/internal/database"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/models"
	"github.com/mcbill1/birthdaymemo/internal/pdfexport"
)

// pdfRequest PDF 导出请求
// 字体支持两种方式：
//  1. 通过 font_key 引用预设字体（前端不需要传输 base64，体积小）
//  2. 通过 font 字段上传自定义 base64 字体（单次使用，生成后销毁）
//
// 任务5：副标题通过 subtitle_image 字段以浏览器字体渲染为 PNG 上传，不再使用服务器本地字体
// 注：年份不从前端传入，由后端使用 time.Now().Year() 自动填充
type pdfRequest struct {
	Range         pdfexport.Range   `json:"range"`
	Settings      models.PdfSetting `json:"settings"`
	TitleFontKey  string            `json:"title_font_key,omitempty"`
	TableFontKey  string            `json:"table_font_key,omitempty"`
	TitleFont     string            `json:"title_font,omitempty"`     // base64 TTF（自定义上传）
	TableFont     string            `json:"table_font,omitempty"`     // base64 TTF
	BgImage       string            `json:"bg_image,omitempty"`       // base64 图片（PNG/JPEG/WebP/GIF）
	SubtitleImage string            `json:"subtitle_image,omitempty"` // 任务5：base64 PNG，浏览器字体渲染的副标题
}

// validateRange 校验导出范围：仅检查类型与月份范围，生日每年重复，不做年份限制
// 年份由后端使用当前时间自动填充；学年导出进行中的学年（9 月起 12 页）
func validateRange(r pdfexport.Range, lang string) string {
	if r.Type != pdfexport.RangeYear && r.Type != pdfexport.RangeMonth && r.Type != pdfexport.RangeSchoolYear {
		return i18n.T(lang, "validation.invalidRangeType")
	}
	if r.Type == pdfexport.RangeMonth {
		if r.Month < 1 || r.Month > 12 {
			return i18n.T(lang, "validation.invalidMonth")
		}
	}
	return ""
}

// loadEntriesForRange 加载某用户范围内（整年/指定月份）的生日
func (s *Server) loadEntriesForRange(uid uint, r pdfexport.Range) []pdfexport.BirthdayEntry {
	var bds []models.Birthday
	s.db.Where("user_id = ?", uid).Find(&bds)
	return entriesForBirthdays(bds, r)
}

// entriesForBirthdays 将生日记录按导出范围转为 PDF 条目（月份范围在内存中过滤）
func entriesForBirthdays(bds []models.Birthday, r pdfexport.Range) []pdfexport.BirthdayEntry {
	out := make([]pdfexport.BirthdayEntry, 0, len(bds))
	for _, b := range bds {
		if r.Type == pdfexport.RangeMonth && b.BirthMonth != r.Month {
			continue
		}
		out = append(out, pdfexport.BirthdayEntry{
			Name: b.Name, Gender: b.Gender, BirthYear: b.BirthYear,
			BirthMonth: b.BirthMonth, BirthDay: b.BirthDay,
		})
	}
	return out
}

// decodeBase64Bytes 安全解码 base64（去掉 data URL 前缀），失败返回 nil
func decodeBase64Bytes(s string) []byte {
	if s == "" {
		return nil
	}
	// 去掉 data:...;base64, 前缀
	if idx := strings.Index(s, ";base64,"); idx >= 0 {
		s = s[idx+len(";base64,"):]
	} else if idx := strings.Index(s, "base64,"); idx >= 0 {
		s = s[idx+len("base64,"):]
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil
	}
	return data
}

// buildResources 从请求中解析单次使用的资源（字体/背景图/副标题图）
func buildResources(req pdfRequest) pdfexport.Resources {
	return pdfexport.Resources{
		TitleFontKey:  req.TitleFontKey,
		TableFontKey:  req.TableFontKey,
		TitleFont:     decodeBase64Bytes(req.TitleFont),
		TableFont:     decodeBase64Bytes(req.TableFont),
		BgImage:       decodeBase64Bytes(req.BgImage),
		SubtitleImage: decodeBase64Bytes(req.SubtitleImage),
	}
}

// listPresetFonts 返回预设字体列表
func (s *Server) listPresetFonts(w http.ResponseWriter, r *http.Request) {
	OK(w, pdfexport.PresetFonts)
}

func (s *Server) pdfPreview(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r)
	var req pdfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	now := time.Now()
	if msg := validateRange(req.Range, s.cfg.Language); msg != "" {
		Fail(w, CodeBadRequest, msg)
		return
	}
	entries := s.loadEntriesForRange(u.ID, req.Range)
	res := buildResources(req)
	data, err := s.pdfGen.Generate(req.Range, entries, req.Settings, now, res)
	if err != nil {
		Fail(w, CodeInternal, err.Error())
		return
	}
	OK(w, map[string]string{"pdf": "data:application/pdf;base64," + base64.StdEncoding.EncodeToString(data)})
}

func (s *Server) pdfExport(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r)
	var req pdfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	now := time.Now()
	if msg := validateRange(req.Range, s.cfg.Language); msg != "" {
		Fail(w, CodeBadRequest, msg)
		return
	}
	entries := s.loadEntriesForRange(u.ID, req.Range)
	res := buildResources(req)
	data, err := s.pdfGen.Generate(req.Range, entries, req.Settings, now, res)
	if err != nil {
		Fail(w, CodeInternal, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=birthdaymemo.pdf")
	w.Write(data)
}

// ---- 公开分享的 PDF 导出（免登录，仅限分享范围内的数据） ----

// handlePublicPdfFonts 返回预设字体列表（公开的静态元数据，分享页导出窗口需要）
func (s *Server) handlePublicPdfFonts(w http.ResponseWriter, r *http.Request) {
	OK(w, pdfexport.PresetFonts)
}

// handlePublicSharePdfSettings 返回分享所有者的 PDF 设计（只读，作为客人导出窗口的初始值；客人无法保存）
func (s *Server) handlePublicSharePdfSettings(w http.ResponseWriter, r *http.Request) {
	link := s.loadShareLink(chi.URLParam(r, "token"))
	if link == nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	var ps models.PdfSetting
	if err := s.db.Where("user_id = ?", link.OwnerUserID).First(&ps).Error; err != nil {
		database.EnsureUserSettings(s.db, link.OwnerUserID)
		s.db.Where("user_id = ?", link.OwnerUserID).First(&ps)
	}
	OK(w, ps)
}

// handlePublicSharePdfExport 按分享范围生成 PDF（纯生成、不落盘；字体/背景图随请求单次使用，不保存）
func (s *Server) handlePublicSharePdfExport(w http.ResponseWriter, r *http.Request) {
	link := s.loadShareLink(chi.URLParam(r, "token"))
	if link == nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	var req pdfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	if msg := validateRange(req.Range, s.cfg.Language); msg != "" {
		Fail(w, CodeBadRequest, msg)
		return
	}
	bds, _ := s.shareScopeBirthdays(link)
	entries := entriesForBirthdays(bds, req.Range)
	res := buildResources(req)
	data, err := s.pdfGen.Generate(req.Range, entries, req.Settings, time.Now(), res)
	if err != nil {
		Fail(w, CodeInternal, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=birthdaymemo.pdf")
	w.Write(data)
}
