package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mcbill1/birthdaymemo/internal/i18n"
)

// Config 服务端配置，保存在可执行文件同目录的 config.json。
// 仅包含启动相关与需重启生效的设置；动态可编辑设置（SMTP、邮件模板等）存于数据库。
type Config struct {
	Language                  string `json:"language"`                     // 控制台/默认语言: en | zh
	ListenAddress             string `json:"listen_address"`               // 监听地址
	ListenPort                int    `json:"listen_port"`                  // 监听端口
	OperationLogRetentionDays int    `json:"operation_log_retention_days"` // 操作日志保留天数，0=无限
	ExternalURL               string `json:"external_url"`                 // 外部访问基础地址（带 http/https），用于邮件确认链接等
}

// Default 返回默认配置（不写出文件）
func Default() *Config {
	return &Config{
		Language:                  "en",
		ListenAddress:             "0.0.0.0",
		ListenPort:                0, // 首次配置时随机生成
		OperationLogRetentionDays: 14,
	}
}

// ConfigPath 返回可执行文件同目录下的 config.json 路径
func ConfigPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "config.json"), nil
}

// Load 从指定路径读取配置。exists=false 表示文件不存在（需首次配置）。
func Load(path string) (*Config, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), false, nil
		}
		return nil, false, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, true, err
	}
	if c.Language == "" {
		c.Language = "en"
	}
	if c.ListenAddress == "" {
		c.ListenAddress = "0.0.0.0"
	}
	return &c, true, nil
}

// Save 将配置写入指定路径
func Save(path string, c *Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// allowedListenAddresses 允许的监听地址白名单
var allowedListenAddresses = map[string]bool{
	"0.0.0.0":   true,
	"127.0.0.1": true,
	"::":        true,
	"::1":       true,
}

// ValidateListenAddress 校验监听地址，仅允许常用值；空则默认 0.0.0.0
func ValidateListenAddress(addr string, lang string) (string, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "0.0.0.0", nil
	}
	if !allowedListenAddresses[addr] {
		return "", fmt.Errorf("%s", i18n.T(lang, "validation.listenAddress"))
	}
	return addr, nil
}

// ValidateListenPort 校验监听端口，范围 1-65535
func ValidateListenPort(portStr string, lang string) (int, error) {
	portStr = strings.TrimSpace(portStr)
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%s", i18n.T(lang, "validation.listenPort"))
	}
	return port, nil
}

// ValidateUsername 校验用户名，3-10 字符
func ValidateUsername(name string, lang string) error {
	n := len([]rune(name))
	if n < 3 || n > 10 {
		return fmt.Errorf("%s", i18n.T(lang, "validation.username"))
	}
	return nil
}

// ValidatePassword 校验密码策略：必须同时包含大写字母、小写字母和数字
func ValidatePassword(pw string, lang string) error {
	if len(pw) < 8 {
		return fmt.Errorf("%s", i18n.T(lang, "validation.passwordLength"))
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range pw {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("%s", i18n.T(lang, "validation.passwordComplexity"))
	}
	return nil
}

// ValidateRetentionDays 校验操作日志保留天数，-1 视为 0(无限)
func ValidateRetentionDays(s string, lang string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 14, nil
	}
	d, err := strconv.Atoi(s)
	if err != nil || d < 0 {
		return 0, fmt.Errorf("%s", i18n.T(lang, "validation.retentionDays"))
	}
	return d, nil
}

// ValidateExternalURL 校验外部访问地址：空字符串合法（表示未配置）。
// 非空时必须以 http:// 或 https:// 开头；自动去除结尾的 "/"。
func ValidateExternalURL(raw string, lang string) (string, error) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", nil
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return "", fmt.Errorf("%s", i18n.T(lang, "validation.externalUrl"))
	}
	// 去除结尾多余的 "/"，避免拼接链接时出现 //
	u = strings.TrimRight(u, "/")
	return u, nil
}

// LanguageLabel 语言显示标签
func LanguageLabel(code string) string {
	switch code {
	case "zh":
		return "中文"
	case "en":
		return "English"
	default:
		return code
	}
}
