package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mcbill1/birthdaymemo/internal/audit"
	"github.com/mcbill1/birthdaymemo/internal/auth"
	"github.com/mcbill1/birthdaymemo/internal/config"
	"github.com/mcbill1/birthdaymemo/internal/console"
	"github.com/mcbill1/birthdaymemo/internal/database"
	"github.com/mcbill1/birthdaymemo/internal/email"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/logger"
	"github.com/mcbill1/birthdaymemo/internal/models"
	"github.com/mcbill1/birthdaymemo/internal/reminder"
	"github.com/mcbill1/birthdaymemo/internal/version"
	"github.com/mcbill1/birthdaymemo/internal/web"
	"gorm.io/gorm"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	resetPwd := flag.Bool("reset-admin-password", false, "Interactively reset admin password (reads from terminal, not args)")
	flag.Parse()

	logger.Init()
	fmt.Println()
	fmt.Println(version.Banner())
	fmt.Println()

	cfgPath, err := config.ConfigPath()
	if err != nil {
		logger.Error("get config path failed: %v", err)
		os.Exit(1)
	}
	cfg, exists, err := config.Load(cfgPath)
	if err != nil {
		logger.Error("load config failed: %v", err)
		os.Exit(1)
	}

	// 数据目录与语言目录均位于可执行文件同目录
	dataDir, err := dataDirPath()
	if err != nil {
		logger.Error("get data dir failed: %v", err)
		os.Exit(1)
	}
	langDir := filepath.Join(dataDir, "languages")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		logger.Error("create language dir failed: %v", err)
		os.Exit(1)
	}

	// 初始化数据库
	db, err := database.Init()
	if err != nil {
		logger.Error("init database failed: %v", err)
		os.Exit(1)
	}
	// 注入数据库到 auth 包（用于持久化会话）
	auth.SetDB(db)

	// --reset-admin-password 模式：交互式重置后退出
	if *resetPwd {
		runResetAdminPassword(db, cfg.Language)
		return
	}

	// 首次运行：进入配置向导并创建管理员
	if !exists {
		adminUser, adminPass := runFirstRun(cfg, cfgPath, db)
		logger.Info("admin user '%s' created", adminUser)
		_ = adminPass
	}

	// 加载用户语言文件
	if errs := i18n.LoadUserDir(langDir); len(errs) > 0 {
		for _, e := range errs {
			logger.Error("load language file: %v", e)
		}
	}

	// 启动提醒调度器
	scheduler := reminder.NewScheduler(db, email.NewSender())
	scheduler.Start()
	defer scheduler.Stop()

	// 启动操作日志清理协程
	stopCleanup := make(chan struct{})
	go runAuditCleanup(db, cfg, stopCleanup)
	defer close(stopCleanup)

	// 嵌入的前端静态资源
	sub, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		logger.Error("embed sub fs failed: %v", err)
		os.Exit(1)
	}
	server := web.NewServer(db, cfg, langDir, dataDir, http.FS(sub))

	addr := fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort)
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 监听信号，优雅关闭
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		logger.Info("server listening on %s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server: %v", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Error("shutdown: %v", err)
	}
	logger.Info("bye")
}

// dataDirPath 返回可执行文件所在目录作为数据目录
func dataDirPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// runFirstRun 执行首次运行向导：保存配置并创建管理员
func runFirstRun(cfg *config.Config, cfgPath string, db *gorm.DB) (string, string) {
	newCfg, adminUser, adminPass := console.FirstRunSetup()
	*cfg = *newCfg
	if err := config.Save(cfgPath, cfg); err != nil {
		logger.Error("save config failed: %v", err)
		os.Exit(1)
	}
	if err := createAdmin(db, adminUser, adminPass, cfg.Language); err != nil {
		logger.Error("create admin failed: %v", err)
		os.Exit(1)
	}
	return adminUser, adminPass
}

// runResetAdminPassword 交互式重置管理员密码
func runResetAdminPassword(db *gorm.DB, lang string) {
	useZh := lang == "zh"
	username, password, err := console.ResetAdminPassword(useZh)
	if err != nil {
		logger.Error("read password failed: %v", err)
		os.Exit(1)
	}
	var u models.User
	if err := db.Where("username = ? AND role = ?", username, models.RoleAdmin).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if useZh {
				logger.Error("管理员用户 '%s' 不存在", username)
			} else {
				logger.Error("admin user '%s' not found", username)
			}
		} else {
			logger.Error("query admin: %v", err)
		}
		os.Exit(1)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		logger.Error("hash password: %v", err)
		os.Exit(1)
	}
	if err := db.Model(&models.User{}).Where("id = ?", u.ID).Updates(map[string]any{
		"password_hash":        hash,
		"must_change_password": true,
	}).Error; err != nil {
		logger.Error("update password: %v", err)
		os.Exit(1)
	}
	auth.DestroyUserSessions(u.ID)
	if useZh {
		logger.Info("管理员 '%s' 密码已重置", username)
	} else {
		logger.Info("admin '%s' password reset", username)
	}
}

// createAdmin 创建管理员账号并初始化其用户设置
// lang 为安装向导选择的语言，作为管理员账号的默认语言
func createAdmin(db *gorm.DB, username, password, lang string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	u := models.User{
		Username:           username,
		PasswordHash:       hash,
		Role:               models.RoleAdmin,
		MustChangePassword: false,
		Language:           lang,
		Theme:              models.ThemeLight,
		TopbarRangeDays:    30,
	}
	if err := db.Create(&u).Error; err != nil {
		return err
	}
	return database.EnsureUserSettings(db, u.ID)
}

// runAuditCleanup 定期清理操作日志
func runAuditCleanup(db *gorm.DB, cfg *config.Config, stop chan struct{}) {
	// 启动时先清理一次
	audit.CleanExpired(db, cfg.OperationLogRetentionDays)
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			audit.CleanExpired(db, cfg.OperationLogRetentionDays)
		}
	}
}
