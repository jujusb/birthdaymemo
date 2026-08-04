package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mcbill1/birthdaymemo/internal/audit"
	"github.com/mcbill1/birthdaymemo/internal/auth"
	"github.com/mcbill1/birthdaymemo/internal/config"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/logger"
	"github.com/mcbill1/birthdaymemo/internal/models"
)

// ---- 用户管理 ----

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	s.db.Order("id asc").Find(&users)
	OK(w, users)
}

// createUserRequest 创建用户请求
type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	if err := config.ValidateUsername(req.Username, s.cfg.Language); err != nil {
		Fail(w, CodeBadRequest, err.Error())
		return
	}
	if err := config.ValidatePassword(req.Password, s.cfg.Language); err != nil {
		Fail(w, CodeBadRequest, err.Error())
		return
	}
	// 唯一性
	var count int64
	s.db.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.usernameExists"))
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	admin := userFromContext(r)
	u := models.User{
		Username:           req.Username,
		PasswordHash:       hash,
		Role:               models.RoleUser,
		MustChangePassword: true,
		Language:           s.cfg.Language,
		Theme:              models.ThemeLight,
		TopbarRangeDays:    30,
	}
	if err := s.db.Create(&u).Error; err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	audit.Log(s.db, admin.ID, admin.Username, "create_user", req.Username, ipFromContext(r))
	logger.Info("admin: user '%s' created by '%s' ip=%s", req.Username, admin.Username, ipFromContext(r))
	OK(w, u)
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	admin := userFromContext(r)
	id := chi.URLParam(r, "id")
	idInt, _ := strconv.Atoi(id)
	if uint(idInt) == admin.ID {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "admin.cannotDeleteSelf"))
		return
	}
	var u models.User
	if err := s.db.First(&u, id).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	if u.Role == models.RoleAdmin {
		var adminCount int64
		s.db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&adminCount)
		if adminCount <= 1 {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "admin.cannotDeleteLastAdmin"))
			return
		}
	}
	s.db.Delete(&u)
	auth.DestroyUserSessions(u.ID)
	audit.Log(s.db, admin.ID, admin.Username, "delete_user", u.Username, ipFromContext(r))
	logger.Info("admin: user '%s' deleted by '%s' ip=%s", u.Username, admin.Username, ipFromContext(r))
	OK(w, nil)
}

// resetPasswordRequest 重置密码请求
type resetPasswordRequest struct {
	Password string `json:"password"`
}

func (s *Server) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	admin := userFromContext(r)
	id := chi.URLParam(r, "id")
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	if err := config.ValidatePassword(req.Password, s.cfg.Language); err != nil {
		Fail(w, CodeBadRequest, err.Error())
		return
	}
	var u models.User
	if err := s.db.First(&u, id).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	s.db.Model(&models.User{}).Where("id = ?", u.ID).Updates(map[string]any{
		"password_hash":        hash,
		"must_change_password": true,
	})
	auth.DestroyUserSessions(u.ID)
	audit.Log(s.db, admin.ID, admin.Username, "reset_password", u.Username, ipFromContext(r))
	logger.Info("admin: password reset for '%s' by '%s' ip=%s", u.Username, admin.Username, ipFromContext(r))
	OK(w, nil)
}

// ---- 系统配置 ----

// systemConfigResponse 系统配置响应
type systemConfigResponse struct {
	Language                  string `json:"language"`
	ListenAddress             string `json:"listen_address"`
	ListenPort                int    `json:"listen_port"`
	OperationLogRetentionDays int    `json:"operation_log_retention_days"`
	ExternalURL               string `json:"external_url"`
}

func (s *Server) getSystemConfig(w http.ResponseWriter, r *http.Request) {
	OK(w, systemConfigResponse{
		Language:                  s.cfg.Language,
		ListenAddress:             s.cfg.ListenAddress,
		ListenPort:                s.cfg.ListenPort,
		OperationLogRetentionDays: s.cfg.OperationLogRetentionDays,
		ExternalURL:               s.cfg.ExternalURL,
	})
}

// systemConfigRequest 系统配置请求
type systemConfigRequest struct {
	Language                  string `json:"language"`
	ListenAddress             string `json:"listen_address"`
	ListenPort                int    `json:"listen_port"`
	OperationLogRetentionDays int    `json:"operation_log_retention_days"`
	ExternalURL               string `json:"external_url"`
}

// systemConfigResult 系统配置更新结果
type systemConfigResult struct {
	NeedRestart bool `json:"need_restart"`
}

func (s *Server) updateSystemConfig(w http.ResponseWriter, r *http.Request) {
	admin := userFromContext(r)
	var req systemConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	addr, err := config.ValidateListenAddress(req.ListenAddress, s.cfg.Language)
	if err != nil {
		Fail(w, CodeBadRequest, err.Error())
		return
	}
	if req.ListenPort < 1 || req.ListenPort > 65535 {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "validation.invalidPort"))
		return
	}
	externalURL, err := config.ValidateExternalURL(req.ExternalURL, s.cfg.Language)
	if err != nil {
		Fail(w, CodeBadRequest, err.Error())
		return
	}
	// 语言校验
	langValid := false
	for _, l := range i18n.Available() {
		if l.Code == req.Language {
			langValid = true
			break
		}
	}
	if !langValid {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "settings.language"))
		return
	}

	needRestart := false
	if addr != s.cfg.ListenAddress || req.ListenPort != s.cfg.ListenPort {
		needRestart = true
	}
	// 语言与日志保留实时生效
	s.cfg.Language = req.Language
	s.cfg.OperationLogRetentionDays = req.OperationLogRetentionDays
	// 监听地址/端口写入配置文件，需重启生效
	s.cfg.ListenAddress = addr
	s.cfg.ListenPort = req.ListenPort
	// 外部访问地址实时生效（用于邮件确认链接）
	s.cfg.ExternalURL = externalURL

	path, _ := config.ConfigPath()
	if err := config.Save(path, s.cfg); err != nil {
		Fail(w, CodeInternal, err.Error())
		return
	}
	audit.Log(s.db, admin.ID, admin.Username, "update_system_config", "system config", ipFromContext(r))
	logger.Info("admin: system config updated by '%s' (lang=%s addr=%s:%d retention=%d external_url=%s) ip=%s",
		admin.Username, req.Language, addr, req.ListenPort, req.OperationLogRetentionDays, externalURL, ipFromContext(r))
	OK(w, systemConfigResult{NeedRestart: needRestart})
}

// ---- SMTP ----

// smtpResponse SMTP 响应（不返回密码）
type smtpResponse struct {
	ID          uint   `json:"id"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	From        string `json:"from"`
	Encryption  string `json:"encryption"`
	HasPassword bool   `json:"has_password"`
}

func (s *Server) getSmtp(w http.ResponseWriter, r *http.Request) {
	var smtp models.SmtpSetting
	s.db.First(&smtp, 1)
	OK(w, smtpResponse{
		ID: smtp.ID, Host: smtp.Host, Port: smtp.Port, Username: smtp.Username,
		From: smtp.From, Encryption: smtp.Encryption, HasPassword: smtp.Password != "",
	})
}

// smtpRequest SMTP 更新请求
type smtpRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	From       string `json:"from"`
	Encryption string `json:"encryption"`
}

func (s *Server) updateSmtp(w http.ResponseWriter, r *http.Request) {
	admin := userFromContext(r)
	var req smtpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	var smtp models.SmtpSetting
	s.db.First(&smtp, 1)
	smtp.Host = strings.TrimSpace(req.Host)
	smtp.Port = req.Port
	smtp.Username = strings.TrimSpace(req.Username)
	if req.Password != "" {
		smtp.Password = req.Password
	}
	smtp.From = strings.TrimSpace(req.From)
	switch req.Encryption {
	case models.SMTPEncryptionNone, models.SMTPEncryptionTLS, models.SMTPEncryptionStartTLS:
		smtp.Encryption = req.Encryption
	default:
		smtp.Encryption = models.SMTPEncryptionStartTLS
	}
	smtp.ID = 1
	s.db.Save(&smtp)
	audit.Log(s.db, admin.ID, admin.Username, "update_smtp", "smtp config", ipFromContext(r))
	logger.Info("admin: smtp updated by '%s' (host=%s:%d user=%s from=%s enc=%s) ip=%s",
		admin.Username, smtp.Host, smtp.Port, smtp.Username, smtp.From, smtp.Encryption, ipFromContext(r))
	OK(w, smtpResponse{
		ID: smtp.ID, Host: smtp.Host, Port: smtp.Port, Username: smtp.Username,
		From: smtp.From, Encryption: smtp.Encryption, HasPassword: smtp.Password != "",
	})
}

func (s *Server) testSmtp(w http.ResponseWriter, r *http.Request) {
	admin := userFromContext(r)
	var smtp models.SmtpSetting
	s.db.First(&smtp, 1)
	if smtp.Host == "" || smtp.Username == "" || smtp.Password == "" {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "validation.smtpNotConfigured"))
		return
	}
	if err := s.smtpSender.Test(smtp); err != nil {
		logger.Error("admin: smtp test failed by '%s' host=%s:%d err=%v", admin.Username, smtp.Host, smtp.Port, err)
		Fail(w, CodeBadRequest, err.Error())
		return
	}
	logger.Info("admin: smtp test ok by '%s' host=%s:%d", admin.Username, smtp.Host, smtp.Port)
	OK(w, nil)
}

// ---- 邮件模板 ----

func (s *Server) getEmailTemplate(w http.ResponseWriter, r *http.Request) {
	var tpl models.EmailTemplate
	s.db.First(&tpl, 1)
	OK(w, tpl)
}

// emailTemplateRequest 邮件模板请求
type emailTemplateRequest struct {
	Subject            string `json:"subject"`
	Body               string `json:"body"`
	SelfAdvanceSubject string `json:"self_advance_subject"`
	SelfAdvanceBody    string `json:"self_advance_body"`
	SelfTodaySubject   string `json:"self_today_subject"`
	SelfTodayBody      string `json:"self_today_body"`
}

func (s *Server) updateEmailTemplate(w http.ResponseWriter, r *http.Request) {
	admin := userFromContext(r)
	var req emailTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	if strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Body) == "" {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.invalidInput"))
		return
	}
	// 任务5：本人生日模板校验
	if strings.TrimSpace(req.SelfAdvanceSubject) == "" || strings.TrimSpace(req.SelfAdvanceBody) == "" {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.invalidInput"))
		return
	}
	if strings.TrimSpace(req.SelfTodaySubject) == "" || strings.TrimSpace(req.SelfTodayBody) == "" {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.invalidInput"))
		return
	}
	var tpl models.EmailTemplate
	s.db.First(&tpl, 1)
	tpl.ID = 1
	tpl.Subject = req.Subject
	tpl.Body = req.Body
	tpl.SelfAdvanceSubject = req.SelfAdvanceSubject
	tpl.SelfAdvanceBody = req.SelfAdvanceBody
	tpl.SelfTodaySubject = req.SelfTodaySubject
	tpl.SelfTodayBody = req.SelfTodayBody
	s.db.Save(&tpl)
	audit.Log(s.db, admin.ID, admin.Username, "update_email_template", "email template", ipFromContext(r))
	logger.Info("admin: email template updated by '%s' ip=%s", admin.Username, ipFromContext(r))
	OK(w, tpl)
}

// ---- 语言管理 ----

// languagesResponse 语言管理响应
type languagesResponse struct {
	Available []i18n.Language `json:"available"`
	Errors    []string        `json:"errors"`
}

func (s *Server) adminLanguages(w http.ResponseWriter, r *http.Request) {
	OK(w, languagesResponse{Available: i18n.Available()})
}

func (s *Server) scanLanguages(w http.ResponseWriter, r *http.Request) {
	admin := userFromContext(r)
	errs := i18n.LoadUserDir(s.langDir)
	out := languagesResponse{Available: i18n.Available()}
	for _, e := range errs {
		out.Errors = append(out.Errors, e.Error())
	}
	audit.Log(s.db, admin.ID, admin.Username, "scan_languages", "scan language folder", ipFromContext(r))
	logger.Info("admin: languages scanned by '%s' (available=%d, errors=%d) ip=%s",
		admin.Username, len(out.Available), len(out.Errors), ipFromContext(r))
	OK(w, out)
}

// validateLanguageFileRequest 校验语言文件请求
type validateLanguageFileRequest struct {
	Content string `json:"content"`
	Code    string `json:"code"`
}

func (s *Server) validateLanguageFile(w http.ResponseWriter, r *http.Request) {
	var req validateLanguageFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	if err := i18n.ValidateFile([]byte(req.Content), "en"); err != nil {
		Fail(w, CodeBadRequest, err.Error())
		return
	}
	OK(w, nil)
}

// exportSampleLanguage 导出英文版示例语言文件（参考模板）
func (s *Server) exportSampleLanguage(w http.ResponseWriter, r *http.Request) {
	data := i18n.SampleFile()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="en.json"`)
	w.Write(data)
}

// ---- 操作日志 ----

// listOperationLogs 返回操作日志列表（支持筛选 + 分页）
// 查询参数：
//   - page: 页码，默认 1
//   - size: 每页条数，默认 50
//   - users: 用户名（多选，逗号分隔，精确匹配）
//   - actions: 操作类型（多选，逗号分隔，精确匹配）
//   - detail: 详情关键字（模糊搜索）
//   - ip: IP 地址关键字（模糊搜索）
func (s *Server) listOperationLogs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 200 {
		size = 50
	}
	q := s.db.Model(&models.OperationLog{})
	// 用户名多选筛选
	if users := r.URL.Query().Get("users"); users != "" {
		userList := splitNonEmpty(users, ",")
		if len(userList) > 0 {
			q = q.Where("username IN ?", userList)
		}
	}
	// 操作类型多选筛选
	if actions := r.URL.Query().Get("actions"); actions != "" {
		actionList := splitNonEmpty(actions, ",")
		if len(actionList) > 0 {
			q = q.Where("action IN ?", actionList)
		}
	}
	// 详情关键字（模糊搜索）
	if detail := r.URL.Query().Get("detail"); detail != "" {
		q = q.Where("detail LIKE ?", "%"+detail+"%")
	}
	// IP 地址关键字（模糊搜索）
	if ip := r.URL.Query().Get("ip"); ip != "" {
		q = q.Where("ip LIKE ?", "%"+ip+"%")
	}

	var total int64
	q.Count(&total)

	var logs []models.OperationLog
	q.Order("created_at desc").Limit(size).Offset((page - 1) * size).Find(&logs)
	OK(w, map[string]any{
		"logs":  logs,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// splitNonEmpty 按分隔符切分字符串并去除空白和空项
func splitNonEmpty(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ---- 日志文件查看 ----

// logFileResponse 日志文件列表响应
type logFileResponse struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

func (s *Server) listLogFiles(w http.ResponseWriter, r *http.Request) {
	files := logger.ListLogFiles()
	out := make([]logFileResponse, 0, len(files))
	for _, f := range files {
		out = append(out, logFileResponse{
			Name: f.Name, Size: f.Size, ModTime: f.ModTime,
		})
	}
	OK(w, out)
}

// logFileContentResponse 日志文件内容响应
type logFileContentResponse struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

func (s *Server) getLogFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
	content, err := logger.ReadLogFile(name, tail)
	if err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	OK(w, logFileContentResponse{Name: name, Content: content, Size: int64(len(content))})
}
