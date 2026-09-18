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
	GuestOnly                 bool   `json:"guest_only"`                   // 访客只读模式：仅挂载公开分享接口（与 --guest-only / BIRTHDAYMEMO_GUEST_ONLY 等价）
	TagNameMaxLength          int    `json:"tag_name_max_length"`          // 标签名称最大字符数（与 BIRTHDAYMEMO_TAG_NAME_MAX_LENGTH 等价），默认 20
}

// DefaultTagNameMaxLength 标签名称最大字符数默认值
const DefaultTagNameMaxLength = 20

// MaxTagNameMaxLength 标签名称最大字符数上限（gorm 列宽与之对齐）
const MaxTagNameMaxLength = 100

// Default 返回默认配置（不写出文件）
func Default() *Config {
	return &Config{
		Language:                  "en",
		ListenAddress:             "0.0.0.0",
		ListenPort:                0, // 首次配置时随机生成
		OperationLogRetentionDays: 14,
		TagNameMaxLength:          DefaultTagNameMaxLength,
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
	if c.TagNameMaxLength < 1 {
		c.TagNameMaxLength = DefaultTagNameMaxLength
	}
	if c.TagNameMaxLength > MaxTagNameMaxLength {
		c.TagNameMaxLength = MaxTagNameMaxLength
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

// ValidateTagNameMaxLength 校验标签名称最大字符数，范围 1-100
func ValidateTagNameMaxLength(s string, lang string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return DefaultTagNameMaxLength, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > MaxTagNameMaxLength {
		return 0, fmt.Errorf("%s", i18n.T(lang, "validation.tagNameMaxLength"))
	}
	return n, nil
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

// HasAnyEnv 是否设置了任意 BIRTHDAYMEMO_* 应用配置环境变量。
// 用于判断是否应走非交互式（compose/docker）配置路径。
func HasAnyEnv() bool {
	for _, k := range []string{
		"BIRTHDAYMEMO_LANGUAGE",
		"BIRTHDAYMEMO_LISTEN_ADDRESS",
		"BIRTHDAYMEMO_LISTEN_PORT",
		"BIRTHDAYMEMO_RETENTION_DAYS",
		"BIRTHDAYMEMO_EXTERNAL_URL",
		"BIRTHDAYMEMO_GUEST_ONLY",
		"BIRTHDAYMEMO_TAG_NAME_MAX_LENGTH",
	} {
		if strings.TrimSpace(os.Getenv(k)) != "" {
			return true
		}
	}
	return false
}

// ApplyEnv 将 BIRTHDAYMEMO_* 环境变量覆盖到 c（env > 文件 > 默认）。
// 仅处理被显式设置（去除空白后非空）的变量；未设置的保持原值。
// 返回 changed 表示是否有字段被覆盖，调用方应在 changed 时 Save 回文件。
func ApplyEnv(c *Config) (changed bool, err error) {
	// 语言先解析，后续校验的错误信息需要它
	if v := strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_LANGUAGE")); v != "" {
		if v != "zh" && v != "en" {
			return false, fmt.Errorf("invalid BIRTHDAYMEMO_LANGUAGE %q: must be zh or en", v)
		}
		if c.Language != v {
			c.Language = v
			changed = true
		}
	}
	lang := c.Language
	if lang == "" {
		lang = "en"
	}
	if v := strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_LISTEN_ADDRESS")); v != "" {
		addr, err := ValidateListenAddress(v, lang)
		if err != nil {
			return false, fmt.Errorf("invalid BIRTHDAYMEMO_LISTEN_ADDRESS: %w", err)
		}
		if c.ListenAddress != addr {
			c.ListenAddress = addr
			changed = true
		}
	}
	if v := strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_LISTEN_PORT")); v != "" {
		port, err := ValidateListenPort(v, lang)
		if err != nil {
			return false, fmt.Errorf("invalid BIRTHDAYMEMO_LISTEN_PORT: %w", err)
		}
		if c.ListenPort != port {
			c.ListenPort = port
			changed = true
		}
	}
	if v := strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_RETENTION_DAYS")); v != "" {
		d, err := ValidateRetentionDays(v, lang)
		if err != nil {
			return false, fmt.Errorf("invalid BIRTHDAYMEMO_RETENTION_DAYS: %w", err)
		}
		if c.OperationLogRetentionDays != d {
			c.OperationLogRetentionDays = d
			changed = true
		}
	}
	if v := strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_EXTERNAL_URL")); v != "" {
		u, err := ValidateExternalURL(v, lang)
		if err != nil {
			return false, fmt.Errorf("invalid BIRTHDAYMEMO_EXTERNAL_URL: %w", err)
		}
		if c.ExternalURL != u {
			c.ExternalURL = u
			changed = true
		}
	}
	if v := strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_GUEST_ONLY")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, fmt.Errorf("invalid BIRTHDAYMEMO_GUEST_ONLY %q: must be 1/true/0/false", v)
		}
		if c.GuestOnly != b {
			c.GuestOnly = b
			changed = true
		}
	}
	if v := strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_TAG_NAME_MAX_LENGTH")); v != "" {
		n, err := ValidateTagNameMaxLength(v, lang)
		if err != nil {
			return false, fmt.Errorf("invalid BIRTHDAYMEMO_TAG_NAME_MAX_LENGTH: %w", err)
		}
		if c.TagNameMaxLength != n {
			c.TagNameMaxLength = n
			changed = true
		}
	}
	return changed, nil
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
