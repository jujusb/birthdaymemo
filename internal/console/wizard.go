package console

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/mcbill1/birthdaymemo/internal/config"
	"golang.org/x/term"
)

// Reader 封装标准输入读取
type Reader struct {
	r *bufio.Reader
}

// New 创建读取器
func New() *Reader {
	return &Reader{r: bufio.NewReader(os.Stdin)}
}

// ReadLine 读取一行并去除空白
func (rd *Reader) ReadLine() string {
	line, _ := rd.r.ReadString('\n')
	return strings.TrimSpace(line)
}

// ReadPassword 读取密码（不回显）
func (rd *Reader) ReadPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		bytes, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
	// 非终端环境回退为普通读取
	return rd.ReadLine(), nil
}

// FirstRunSetup 首次运行交互式配置向导，返回填好的配置与管理员凭据
func FirstRunSetup() (*config.Config, string, string) {
	rd := New()
	fmt.Println(strings.Repeat("-", 40))

	// 1. 语言
	fmt.Println("Select language / 选择语言:")
	fmt.Println("  1. English")
	fmt.Println("  2. 中文")
	lang := "en"
	for {
		fmt.Print("> ")
		choice := rd.ReadLine()
		switch choice {
		case "1", "":
			lang = "en"
		case "2":
			lang = "zh"
		default:
			fmt.Println("Invalid input, enter 1 or 2 / 输入有误，请输入 1 或 2")
			continue
		}
		break
	}
	useZh := lang == "zh"

	// 2. 监听地址
	if useZh {
		fmt.Println("\n监听地址 (支持 IPv6)，可选: 0.0.0.0 / 127.0.0.1 / :: / ::1，留空默认 0.0.0.0")
	} else {
		fmt.Println("\nListen address (IPv6 supported), options: 0.0.0.0 / 127.0.0.1 / :: / ::1, empty = 0.0.0.0")
	}
	var listenAddr string
	for {
		fmt.Print("> ")
		addr, err := config.ValidateListenAddress(rd.ReadLine(), lang)
		if err != nil {
			fmt.Println(err)
			continue
		}
		listenAddr = addr
		break
	}

	// 3. 监听端口（默认 10000 以上随机）
	defaultPort := 10000 + rand.Intn(55535)
	if useZh {
		fmt.Printf("\n监听端口 (1-65535，留空默认随机 %d)\n", defaultPort)
	} else {
		fmt.Printf("\nListen port (1-65535, empty = random %d)\n", defaultPort)
	}
	var listenPort int
	for {
		fmt.Print("> ")
		input := rd.ReadLine()
		if input == "" {
			listenPort = defaultPort
			break
		}
		port, err := config.ValidateListenPort(input, lang)
		if err != nil {
			fmt.Println(err)
			continue
		}
		listenPort = port
		break
	}

	// 4. 管理员用户名
	if useZh {
		fmt.Println("\n管理员用户名 (3-10 字符)")
	} else {
		fmt.Println("\nAdmin username (3-10 characters)")
	}
	var adminUser string
	for {
		fmt.Print("> ")
		name := rd.ReadLine()
		if err := config.ValidateUsername(name, lang); err != nil {
			fmt.Println(err)
			continue
		}
		adminUser = name
		break
	}

	// 5. 管理员密码
	if useZh {
		fmt.Println("\n管理员密码 (至少 8 位，必须包含大写、小写和数字)")
	} else {
		fmt.Println("\nAdmin password (min 8 chars, must contain uppercase, lowercase and digits)")
	}
	var adminPass string
	for {
		fmt.Print("password> ")
		pw, err := rd.ReadPassword()
		if err != nil {
			fmt.Println(err)
			continue
		}
		if err := config.ValidatePassword(pw, lang); err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Print("confirm > ")
		pw2, _ := rd.ReadPassword()
		if pw != pw2 {
			if useZh {
				fmt.Println("两次输入不一致，请重新输入")
			} else {
				fmt.Println("Passwords do not match, try again")
			}
			continue
		}
		adminPass = pw
		break
	}

	// 6. 操作日志保留天数
	if useZh {
		fmt.Println("\n操作日志保留天数 (默认 14，0=无限保存)")
	} else {
		fmt.Println("\nOperation log retention days (default 14, 0 = unlimited)")
	}
	var retention int
	for {
		fmt.Print("> ")
		input := rd.ReadLine()
		d, err := config.ValidateRetentionDays(input, lang)
		if err != nil {
			fmt.Println(err)
			continue
		}
		retention = d
		break
	}

	cfg := &config.Config{
		Language:                  lang,
		ListenAddress:             listenAddr,
		ListenPort:                listenPort,
		OperationLogRetentionDays: retention,
	}

	if useZh {
		fmt.Println("\n配置完成，正在启动...")
	} else {
		fmt.Println("\nConfiguration complete, starting...")
	}
	return cfg, adminUser, adminPass
}

// ResetAdminPassword 命令行强制重置管理员密码流程
func ResetAdminPassword(useZh bool) (string, string, error) {
	lang := "en"
	if useZh {
		lang = "zh"
	}
	rd := New()
	if useZh {
		fmt.Println("\n重置管理员密码")
		fmt.Println("管理员用户名 (3-10 字符)")
	} else {
		fmt.Println("\nReset admin password")
		fmt.Println("Admin username (3-10 characters)")
	}
	var adminUser string
	for {
		fmt.Print("> ")
		name := rd.ReadLine()
		if err := config.ValidateUsername(name, lang); err != nil {
			fmt.Println(err)
			continue
		}
		adminUser = name
		break
	}

	if useZh {
		fmt.Println("新密码 (至少 8 位，必须包含大写、小写和数字)")
	} else {
		fmt.Println("New password (min 8 chars, must contain uppercase, lowercase and digits)")
	}
	for {
		fmt.Print("password> ")
		pw, err := rd.ReadPassword()
		if err != nil {
			return "", "", err
		}
		if err := config.ValidatePassword(pw, lang); err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Print("confirm > ")
		pw2, _ := rd.ReadPassword()
		if pw != pw2 {
			if useZh {
				fmt.Println("两次输入不一致，请重新输入")
			} else {
				fmt.Println("Passwords do not match, try again")
			}
			continue
		}
		return adminUser, pw, nil
	}
}
