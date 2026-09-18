package web

import (
	"net/http"
	"os"
	"path"
	"strings"

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
		// 只读分享（公开，仅 GET；写操作永不挂载）
		// 分享页导出窗口需要的 PDF 接口：纯生成/只读，无状态变更，故公开
		r.Get("/public/s/{token}", s.handlePublicShare)
		r.Get("/public/s/{token}/calendar", s.handlePublicShareCalendar)
		r.Get("/public/s/{token}/pdf-settings", s.handlePublicSharePdfSettings)
		r.Post("/public/s/{token}/pdf/export", s.handlePublicSharePdfExport)
		r.Get("/public/pdf/fonts", s.handlePublicPdfFonts)

		// 需认证接口
		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(s.db, s.cfg.Language))
			r.Post("/logout", s.handleLogout)
			r.Get("/me", s.handleMe)
			r.Post("/change-password", s.handleChangePassword)
			r.Get("/upcoming", s.handleUpcoming)
			r.Get("/calendar", s.handleCalendar)
			r.Get("/birthdays", s.listBirthdays)
			r.Get("/tags", s.listTags)
			r.Get("/settings", s.getSettings)
			r.Get("/settings/email/status", s.getEmailStatus)
			r.Get("/pdf-settings", s.getPdfSettings)
			r.Get("/pdf/fonts", s.listPresetFonts)
			r.Get("/share-links", s.listShareLinks)
			r.Get("/grants", s.listGrants)

			// 写接口组：guest 角色一律 403
			r.Group(func(r chi.Router) {
				r.Use(BlockGuestWrites(s.cfg.Language))
				r.Post("/settings/email/confirm", s.requestEmailConfirmation)
				r.Post("/birthdays", s.createBirthday)
				r.Put("/birthdays/{id}", s.updateBirthday)
				r.Delete("/birthdays/{id}", s.deleteBirthday)
				r.Post("/tags", s.createTag)
				r.Put("/tags/{id}", s.updateTag)
				r.Delete("/tags/{id}", s.deleteTag)
				r.Put("/settings", s.updateSettings)
				r.Put("/pdf-settings", s.updatePdfSettings)
				r.Post("/pdf/preview", s.pdfPreview)
				r.Post("/pdf/export", s.pdfExport)
				r.Post("/share-links", s.createShareLink)
				r.Put("/share-links/{id}", s.updateShareLink)
				r.Delete("/share-links/{id}", s.deleteShareLink)
				r.Post("/share-links/{id}/rotate", s.rotateShareLink)
				r.Post("/grants", s.createGrant)
				r.Put("/grants/{id}", s.updateGrant)
				r.Delete("/grants/{id}", s.deleteGrant)
			})

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

	s.mountStatic(r)
	return r
}

// GuestRoutes 构建访客只读路由（--guest-only 模式专用）。
// 仅挂载公开分享接口与分享页所需的静态资源；登录、认证、写接口、管理接口、
// 其他前端页面一律不存在（chi 默认 404，且无 SPA 回退）。
// 配合 compose 中的 birthdaymemo-guest 服务使用，与编辑服务共享同一 SQLite 文件（本进程只发 SELECT）。
func (s *Server) GuestRoutes() http.Handler {
	r := chi.NewRouter()

	// 全局安全响应头中间件
	r.Use(SecurityHeaders)

	r.Route("/api", func(r chi.Router) {
		r.Use(RequestLog)
		// 只读分享（公开；PDF 导出为纯生成/只读，无状态变更）
		r.Get("/public/s/{token}", s.handlePublicShare)
		r.Get("/public/s/{token}/calendar", s.handlePublicShareCalendar)
		r.Get("/public/s/{token}/pdf-settings", s.handlePublicSharePdfSettings)
		r.Post("/public/s/{token}/pdf/export", s.handlePublicSharePdfExport)
		r.Get("/public/pdf/fonts", s.handlePublicPdfFonts)
		// 分享页需要的语言资源（公开）
		r.Get("/languages", s.handleLanguages)
		r.Get("/i18n", s.handleI18n)
	})

	s.mountGuestStatic(r)
	return r
}

// mountGuestStatic 挂载访客模式的静态资源（严格锁定）。
// 分享页使用 hash 路由（/#/s/...），浏览器实际只请求 "/" 和静态资源文件，
// 因此可以做到：
//   - "/" → index.html（分享页外壳，按 hash 渲染 #/s/:token）
//   - 带扩展名的真实文件（/assets/*.js、*.css、favicon 等）→ 直接提供
//   - 其他一切服务端路径 → 404（无 SPA 回退：登录页、管理页等在访客服务上不存在）
func (s *Server) mountGuestStatic(r *chi.Mux) {
	if s.fs == nil {
		return
	}
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/" {
			s.serveStaticFile(w, r, "/index.html")
			return
		}
		// 仅提供真实存在的静态文件；导航路径一律 404
		if !strings.Contains(path.Base(p), ".") {
			http.NotFound(w, r)
			return
		}
		f, err := s.fs.Open(p)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		if stat, _ := f.Stat(); stat.IsDir() {
			http.NotFound(w, r)
			return
		}
		stat, _ := f.Stat()
		http.ServeContent(w, r, p, stat.ModTime(), f)
	})
}

// serveStaticFile 提供单个内嵌静态文件，缺失时 404
func (s *Server) serveStaticFile(w http.ResponseWriter, r *http.Request, name string) {
	f, err := s.fs.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	if stat, _ := f.Stat(); stat.IsDir() {
		http.NotFound(w, r)
		return
	}
	stat, _ := f.Stat()
	http.ServeContent(w, r, name, stat.ModTime(), f)
}

// mountStatic 挂载嵌入的前端静态资源与 SPA 回退
// 不使用 http.FileServer，因为它对 index.html 会产生 301 重定向
func (s *Server) mountStatic(r *chi.Mux) {
	if s.fs == nil {
		return
	}
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
