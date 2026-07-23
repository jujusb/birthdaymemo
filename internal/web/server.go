package web

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/mcbill1/birthdaymemo/internal/config"
	"github.com/mcbill1/birthdaymemo/internal/email"
	"github.com/mcbill1/birthdaymemo/internal/pdfexport"
	"gorm.io/gorm"
)

// Server 持有所有依赖与路由
type Server struct {
	db         *gorm.DB
	cfg        *config.Config
	langDir    string
	dataDir    string
	fs         http.FileSystem
	smtpSender *email.Sender
	pdfGen     *pdfexport.Generator
}

// NewServer 创建服务
func NewServer(db *gorm.DB, cfg *config.Config, langDir, dataDir string, fs http.FileSystem) *Server {
	return &Server{
		db:         db,
		cfg:        cfg,
		langDir:    langDir,
		dataDir:    dataDir,
		fs:         fs,
		smtpSender: email.NewSender(),
		pdfGen:     pdfexport.NewGenerator(),
	}
}

// Routes 构建路由
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	// 全局安全响应头中间件
	r.Use(SecurityHeaders)

	r.Route("/api", func(r chi.Router) {
		r.Use(RequestLog)
		// 公共接口（无需认证）
		r.Post("/login", s.handleLogin)
		r.Get("/captcha", s.handleCaptcha)
		r.Get("/languages", s.handleLanguages)
		r.Get("/i18n", s.handleI18n)
		r.Get("/email/confirm", s.confirmEmail) // 邮箱确认链接（公开）

		// 需认证接口
		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(s.db, s.cfg.Language))
			r.Post("/logout", s.handleLogout)
			r.Get("/me", s.handleMe)
			r.Post("/change-password", s.handleChangePassword)
			r.Get("/upcoming", s.handleUpcoming)
			r.Get("/calendar", s.handleCalendar)
			r.Get("/birthdays", s.listBirthdays)
			r.Post("/birthdays", s.createBirthday)
			r.Put("/birthdays/{id}", s.updateBirthday)
			r.Delete("/birthdays/{id}", s.deleteBirthday)
			r.Get("/tags", s.listTags)
			r.Post("/tags", s.createTag)
			r.Put("/tags/{id}", s.updateTag)
			r.Delete("/tags/{id}", s.deleteTag)
			r.Get("/settings", s.getSettings)
			r.Put("/settings", s.updateSettings)
			r.Get("/settings/email/status", s.getEmailStatus)
			r.Post("/settings/email/confirm", s.requestEmailConfirmation)
			r.Get("/pdf-settings", s.getPdfSettings)
			r.Put("/pdf-settings", s.updatePdfSettings)
			r.Get("/pdf/fonts", s.listPresetFonts)
			r.Post("/pdf/preview", s.pdfPreview)
			r.Post("/pdf/export", s.pdfExport)

			// 管理员接口
			r.Group(func(r chi.Router) {
				r.Use(RequireAdmin(s.cfg.Language))
				r.Get("/admin/users", s.listUsers)
				r.Post("/admin/users", s.createUser)
				r.Delete("/admin/users/{id}", s.deleteUser)
				r.Post("/admin/users/{id}/reset-password", s.resetUserPassword)
				r.Get("/admin/system-config", s.getSystemConfig)
				r.Put("/admin/system-config", s.updateSystemConfig)
				r.Get("/admin/smtp", s.getSmtp)
				r.Put("/admin/smtp", s.updateSmtp)
				r.Post("/admin/smtp/test", s.testSmtp)
				r.Get("/admin/email-template", s.getEmailTemplate)
				r.Put("/admin/email-template", s.updateEmailTemplate)
				r.Get("/admin/languages", s.adminLanguages)
				r.Post("/admin/languages/scan", s.scanLanguages)
				r.Post("/admin/languages/validate", s.validateLanguageFile)
				r.Get("/admin/languages/sample", s.exportSampleLanguage)
				r.Get("/admin/operation-logs", s.listOperationLogs)
				r.Get("/admin/log-files", s.listLogFiles)
				r.Get("/admin/log-files/{name}", s.getLogFile)
			})
		})
	})

	// 静态资源（嵌入的前端），SPA 回退
	// 不使用 http.FileServer，因为它对 index.html 会产生 301 重定向
	if s.fs != nil {
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "/" {
				path = "/index.html"
			}
			// 尝试打开请求的静态文件（favicon.ico / logo512.png 等已嵌入 frontend/dist）
			f, err := s.fs.Open(path)
			if err == nil {
				stat, _ := f.Stat()
				if !stat.IsDir() {
					defer f.Close()
					http.ServeContent(w, r, path, stat.ModTime(), f)
					return
				}
				f.Close()
			}
			// 文件不存在或为目录，回退到 index.html（SPA 路由）
			f, err = s.fs.Open("/index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			stat, _ := f.Stat()
			http.ServeContent(w, r, "/index.html", stat.ModTime(), f)
		})
	}
	return r
}

// EnsureDir 确保目录存在
func EnsureDir(dir string) error {
	if dir == "" {
		return nil
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o755)
	}
	return nil
}
