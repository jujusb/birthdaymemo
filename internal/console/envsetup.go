package console

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/mcbill1/birthdaymemo/internal/config"
)

// HasAdminEnv 是否设置了管理员相关的环境变量。
func HasAdminEnv() bool {
	if strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_ADMIN_USER")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_ADMIN_PASSWORD")) != "" {
		return true
	}
	return false
}

// EnvSetup 非交互式管理员配置（docker compose 路径）。
// 用户名取自 BIRTHDAYMEMO_ADMIN_USER（为空则默认 admin），显式设置但非法时返回错误。
// 密码取自 BIRTHDAYMEMO_ADMIN_PASSWORD；缺失或不满足密码策略时自动生成合规随机密码，
// generated=true 表示调用方应将密码打印到日志中（仅首次创建时展示一次）。
func EnvSetup(lang string) (user, pass string, generated bool, err error) {
	user = strings.TrimSpace(os.Getenv("BIRTHDAYMEMO_ADMIN_USER"))
	if user == "" {
		user = "admin"
	}
	if err := config.ValidateUsername(user, lang); err != nil {
		return "", "", false, fmt.Errorf("invalid BIRTHDAYMEMO_ADMIN_USER: %w", err)
	}
	pw := os.Getenv("BIRTHDAYMEMO_ADMIN_PASSWORD")
	// 注意：密码不 TrimSpace，允许首尾空格作为密码的一部分
	if pw == "" || config.ValidatePassword(pw, lang) != nil {
		pw, err = GeneratePassword(16)
		if err != nil {
			return "", "", false, err
		}
		return user, pw, true, nil
	}
	return user, pw, false, nil
}

// GeneratePassword 生成长度为 n 的合规随机密码（至少包含大写、小写和数字）。
func GeneratePassword(n int) (string, error) {
	if n < 8 {
		n = 8
	}
	const upper = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	const lower = "abcdefghijkmnopqrstuvwxyz"
	const digit = "23456789"
	all := upper + lower + digit
	pick := func(set string) (byte, error) {
		i, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
		if err != nil {
			return 0, err
		}
		return set[i.Int64()], nil
	}
	buf := make([]byte, n)
	// 保证三类字符至少各出现一次
	for i := range buf {
		b, err := pick(all)
		if err != nil {
			return "", err
		}
		buf[i] = b
	}
	for _, set := range []string{upper, lower, digit} {
		b, err := pick(set)
		if err != nil {
			return "", err
		}
		pos, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
		if err != nil {
			return "", err
		}
		buf[pos.Int64()] = b
	}
	// Fisher-Yates 洗牌，避免固定位置模式
	for i := n - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		buf[i], buf[j.Int64()] = buf[j.Int64()], buf[i]
	}
	return string(buf), nil
}
