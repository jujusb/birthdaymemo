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
	"strconv"
	"strings"
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
	"golang.org/x/term"
	"gorm.io/gorm"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	resetPwd := flag.Bool("reset-admin-password", false, "Interactively reset admin password (reads from terminal, not args)")
	guestOnly := flag.Bool("guest-only", false, "Guest-only read-only mode: serve only public share links (for the birthdaymemo-guest compose service)")
	flag.Parse()

	// 环境变量与 flag 等价（compose 服务用 env 更方便）
	if !*guestOnly {
		if v := strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_GUEST_ONLY")); v != "" {
			if b, err := strconv.ParseBool(v); err == nil && b {
				*guestOnly = true
			}
		}
	}

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

	// compose/docker 配置：BIRTHDAYMEMO_* 环境变量覆盖文件（env > 文件 > 默认）
	envChanged, err := config.ApplyEnv(cfg)
	if err != nil {
		logger.Error("%v", err)
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

	// --reset-admin-password 模式：交互式重置后退出（guest 模式下禁用）
	if *resetPwd {
		if *guestOnly {
			logger.Error("reset-admin-password is not allowed in guest-only mode")
			os.Exit(1)
		}
		runResetAdminPassword(db, cfg.Language)
		return
	}

	// 首次运行：进入配置向导并创建管理员
	// guest-only 模式跳过整个首次运行块：访客服务绝不创建用户、不写回 config.json；
	// 空数据库时直接报错退出（请先启动编辑服务完成初始化）。
	// 管理员缺失时才需要初始化（兼容 config 存在但数据库被清空的情况）。
	// 非交互式（compose env 或无 TTY）走环境变量配置；本地裸机运行保留交互式向导。
	var adminCount int64
	if err := db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&adminCount).Error; err != nil {
		logger.Error("query admin failed: %v", err)
		os.Exit(1)
	}
	if *guestOnly {
		if adminCount == 0 {
			logger.Error("guest-only mode requires an initialized database (no admin found); start the edit service first")
			os.Exit(1)
		}
	} else if adminCount == 0 {
		isEnvMode := config.HasAnyEnv() || console.HasAdminEnv()
		isTTY := term.IsTerminal(int(os.Stdin.Fd()))
		if isEnvMode || !isTTY {
			// 非交互式：docker compose / 环境变量 / 无 TTY
			adminUser, _ := runEnvFirstRun(cfg, cfgPath, db)
			logger.Info("admin user '%s' created", adminUser)
		} else {
			adminUser, adminPass := runFirstRun(cfg, cfgPath, db)
			logger.Info("admin user '%s' created", adminUser)
			_ = adminPass
		}
	} else if exists && envChanged {
		// compose 是配置来源：运行时覆盖写回文件
		if err := config.Save(cfgPath, cfg); err != nil {
			logger.Error("save config failed: %v", err)
			os.Exit(1)
		}
	}

	// 加载用户语言文件
	if errs := i18n.LoadUserDir(langDir); len(errs) > 0 {
		for _, e := range errs {
			logger.Error("load language file: %v", e)
		}
	}

	// guest-only 模式：不启动提醒调度器（避免与编辑服务重复发邮件），
	// 不启动操作日志清理协程（写操作归编辑服务），只提供只读分享服务。
	var scheduler *reminder.Scheduler
	var stopCleanup chan struct{}
	if !*guestOnly {
		// 启动提醒调度器
		scheduler = reminder.NewScheduler(db, email.NewSender())
		scheduler.Start()
		defer scheduler.Stop()

		// 启动操作日志清理协程
		stopCleanup = make(chan struct{})
		go runAuditCleanup(db, cfg, stopCleanup)
		defer close(stopCleanup)
	}

	// 嵌入的前端静态资源
	sub, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		logger.Error("embed sub fs failed: %v", err)
		os.Exit(1)
	}
	server := web.NewServer(db, cfg, langDir, dataDir, http.FS(sub))

	addr := fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort)
	handler := server.Routes()
	if *guestOnly {
		handler = server.GuestRoutes()
	}
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 监听信号，优雅关闭
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		if *guestOnly {
			logger.Info("guest-only server listening on %s (public share links only)", addr)
		} else {
			logger.Info("server listening on %s", addr)
		}
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

// runEnvFirstRun 非交互式首次运行：环境变量已在 ApplyEnv 中覆盖，
// 此处补齐 docker 默认值、落盘并创建管理员。返回用户名与密码是否为自动生成。
func runEnvFirstRun(cfg *config.Config, cfgPath string, db *gorm.DB) (string, bool) {
	if cfg.ListenPort == 0 {
		cfg.ListenPort = 8080
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = "0.0.0.0"
	}
	if cfg.Language == "" {
		cfg.Language = "en"
	}
	adminUser, adminPass, generated, err := console.EnvSetup(cfg.Language)
	if err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		logger.Error("save config failed: %v", err)
		os.Exit(1)
	}
	if err := createAdmin(db, adminUser, adminPass, cfg.Language); err != nil {
		logger.Error("create admin failed: %v", err)
		os.Exit(1)
	}
	if generated {
		// 自动生成的密码仅在此处展示一次，请及时登录修改
		logger.Info("no valid BIRTHDAYMEMO_ADMIN_PASSWORD provided, generated random password for '%s': %s", adminUser, adminPass)
	}
	return adminUser, generated
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
