package email

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	netsmtp "net/smtp"
	"strings"
	"time"

	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/logger"
	"github.com/mcbill1/birthdaymemo/internal/models"
)

// Sender 邮件发送器
type Sender struct{}

// NewSender 创建邮件发送器
func NewSender() *Sender {
	return &Sender{}
}

// Test 测试 SMTP 连接（发送一封测试邮件给发件人自己）
func (s *Sender) Test(setting models.SmtpSetting) error {
	if setting.From == "" {
		return fmt.Errorf("%s", i18n.T("en", "validation.fromEmpty"))
	}
	return s.send(setting, setting.From, "BirthDayMemo SMTP Test", "This is a test email from BirthDayMemo.")
}

// Send 发送邮件
func (s *Sender) Send(setting models.SmtpSetting, to, subject, body string) error {
	if to == "" {
		return fmt.Errorf("%s", i18n.T("en", "validation.recipientEmpty"))
	}
	return s.send(setting, to, subject, body)
}

// send 根据 encryption 发送邮件
func (s *Sender) send(setting models.SmtpSetting, to, subject, body string) error {
	addr := net.JoinHostPort(setting.Host, fmt.Sprintf("%d", setting.Port))
	msg := buildMessage(setting.From, to, subject, body)

	start := time.Now()
	var err error
	switch setting.Encryption {
	case models.SMTPEncryptionTLS:
		err = sendTLS(setting, addr, msg, to)
	case models.SMTPEncryptionNone:
		var auth netsmtp.Auth
		if setting.Username != "" && setting.Password != "" {
			auth = netsmtp.PlainAuth("", setting.Username, setting.Password, setting.Host)
		}
		err = netsmtp.SendMail(addr, auth, setting.From, []string{to}, msg)
	default:
		var auth netsmtp.Auth
		if setting.Username != "" && setting.Password != "" {
			auth = netsmtp.PlainAuth("", setting.Username, setting.Password, setting.Host)
		}
		err = netsmtp.SendMail(addr, auth, setting.From, []string{to}, msg)
	}
	elapsed := time.Since(start)
	if err != nil {
		logger.Error("email send failed: to=%s subject=%q elapsed=%s err=%v", to, subject, elapsed, err)
		return err
	}
	logger.Info("email sent: to=%s subject=%q elapsed=%s bytes=%d", to, subject, elapsed, len(msg))
	return nil
}

// sendTLS 隐式 TLS 发送
func sendTLS(setting models.SmtpSetting, addr string, msg []byte, to string) error {
	tlsConf := &tls.Config{ServerName: setting.Host}
	conn, err := tls.Dial("tcp", addr, tlsConf)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	c, err := netsmtp.NewClient(conn, setting.Host)
	if err != nil {
		return err
	}
	defer c.Close()
	if setting.Username != "" && setting.Password != "" {
		if err := c.Auth(netsmtp.PlainAuth("", setting.Username, setting.Password, setting.Host)); err != nil {
			return err
		}
	}
	if err := c.Mail(setting.From); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// buildMessage 构造邮件内容
func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mimeEncode(subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

// mimeEncode 简单对主题做 UTF-8 编码（避免部分 SMTP 乱码）
func mimeEncode(s string) string {
	hasNonASCII := false
	for _, r := range s {
		if r > 127 {
			hasNonASCII = true
			break
		}
	}
	if !hasNonASCII {
		return s
	}
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
}

// PersonVars 单个生日人的循环变量数据
type PersonVars struct {
	Name string // {birthday_person_N}
	Date string // {birthday_date_N}
	Age  string // {birthday_age_N}
	Sex  string // {birthday_sex_N}：性别代词（中文：他/她/TA，英文：his/her/their）
	Tags string // {tags_N} 或 {标签_N}：合并的标签名，逗号分隔
}

// RenderTemplateLoop 渲染邮件模板，支持 [Start loop]...[End loop] 循环块
// 任务7：块级循环语法
//   - 静态变量 {user}、{count} 等照常替换
//   - [Start loop] 与 [End loop] 之间的内容会按每个生日人重复一次
//   - 每次重复时，{birthday_person_N}、{birthday_date_N}、{birthday_age_N}、
//     {birthday_sex_N}、{tags_N}、{标签_N} 会被替换为对应人的数据
//
// 示例模板：
//
//	亲爱的 {user}：
//	共有 {count} 位朋友即将过生日：
//	[Start loop]
//	您的{birthday_person_N}好友即将迎来{birthday_sex_N}的{birthday_age_N}岁生日
//	Tag：{tags_N}
//	[End loop]
//	请记得送上祝福！
//
// 若 persons 为空，循环块会被整体删除（不留空行）
func RenderTemplateLoop(tpl string, staticVars map[string]string, persons []PersonVars) string {
	lines := strings.Split(tpl, "\n")
	out := make([]string, 0, len(lines)+len(persons)*3)

	i := 0
	for i < len(lines) {
		raw := lines[i]
		// 去除首尾空白后比较，允许 "[Start loop]" 前后有空格
		trimmed := strings.TrimSpace(raw)
		// 不区分大小写匹配循环标记，方便用户书写
		if strings.EqualFold(trimmed, "[Start loop]") {
			// 收集循环块内的行
			block := make([]string, 0, 4)
			i++
			for i < len(lines) && !strings.EqualFold(strings.TrimSpace(lines[i]), "[End loop]") {
				block = append(block, lines[i])
				i++
			}
			// 跳过 [End loop]
			if i < len(lines) {
				i++
			}
			// 为每个生日人重复一次循环块
			if len(persons) > 0 {
				for _, p := range persons {
					for _, bl := range block {
						newLine := bl
						newLine = strings.ReplaceAll(newLine, "{birthday_person_N}", p.Name)
						newLine = strings.ReplaceAll(newLine, "{birthday_date_N}", p.Date)
						newLine = strings.ReplaceAll(newLine, "{birthday_age_N}", p.Age)
						newLine = strings.ReplaceAll(newLine, "{birthday_sex_N}", p.Sex)
						newLine = strings.ReplaceAll(newLine, "{tags_N}", p.Tags)
						newLine = strings.ReplaceAll(newLine, "{标签_N}", p.Tags)
						// 同时替换此行可能包含的静态变量
						for k, v := range staticVars {
							newLine = strings.ReplaceAll(newLine, "{"+k+"}", v)
						}
						out = append(out, newLine)
					}
				}
			}
			// 没有人时跳过整个循环块
			continue
		}
		// 静态行：直接替换静态变量
		newLine := raw
		for k, v := range staticVars {
			newLine = strings.ReplaceAll(newLine, "{"+k+"}", v)
		}
		out = append(out, newLine)
		i++
	}
	return strings.Join(out, "\n")
}
