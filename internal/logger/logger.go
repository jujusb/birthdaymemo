package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mcbill1/birthdaymemo/internal/version"
	"golang.org/x/term"
)

// ANSI 颜色码
const (
	colorReset = "\x1b[0m"
	colorRed   = "\x1b[31m"
	colorGreen = "\x1b[32m"
)

// Level 日志级别
type Level int

const (
	LevelInfo Level = iota
	LevelError
)

var (
	mu        sync.Mutex
	useColors bool
	// 文件日志
	logFile   *os.File
	logDir    string
	fileDate  string // 当前日志文件对应的日期 YYYY-MM-DD
	fileReady bool
)

// Init 初始化日志：启用控制台颜色（Windows 下启用 VT 处理）并打开日志文件。
func Init() {
	// 控制台颜色：仅当 stdout 是终端时启用
	useColors = false
	if term.IsTerminal(int(os.Stdout.Fd())) {
		useColors = enableVT()
	}
	version.ColoredBanner = useColors

	// 日志文件目录位于可执行文件同目录的 logs/ 子目录
	exe, err := os.Executable()
	if err != nil {
		return
	}
	logDir = filepath.Join(filepath.Dir(exe), "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return
	}
	// 启动时轮转已有的 latest.log
	rotateExisting()
	// 打开新的 latest.log
	openLatest()
}

// ColorsEnabled 返回是否启用了控制台着色
func ColorsEnabled() bool {
	return useColors
}

// log 写入一行：[YYYY-MM-DD HH:MM:SS] [LEVEL] message
// 控制台输出对 [INFO]/[ERROR] 着色；文件输出纯文本。
func log(lvl Level, msg string) {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().Format("2006-01-02 15:04:05")
	plainTag := "[INFO]"
	switch lvl {
	case LevelError:
		plainTag = "[ERROR]"
	}

	// 控制台行（带颜色）
	consoleTag := plainTag
	if useColors {
		switch lvl {
		case LevelInfo:
			consoleTag = colorGreen + plainTag + colorReset
		case LevelError:
			consoleTag = colorRed + plainTag + colorReset
		}
	}
	fmt.Fprintf(os.Stdout, "[%s] %s %s\n", now, consoleTag, msg)

	// 文件行（纯文本，无 ANSI 码）
	if fileReady && logFile != nil {
		// 跨日轮转
		today := time.Now().Format("2006-01-02")
		if today != fileDate {
			logFile.Close()
			archiveFile(fileDate)
			openLatest()
		}
		fmt.Fprintf(logFile, "[%s] %s %s\n", now, plainTag, msg)
	}
}

// rotateExisting 启动时将已有的 latest.log 归档为 YYYY-MM-DD-NN.log
func rotateExisting() {
	latest := filepath.Join(logDir, "latest.log")
	info, err := os.Stat(latest)
	if err != nil || info.Size() == 0 {
		return
	}
	// 使用文件最后修改时间的日期作为归档日期
	date := info.ModTime().Format("2006-01-02")
	archiveFile(date)
	// 确保归档后 latest.log 不存在（archiveFile 已 rename）
	os.Remove(latest)
}

// archiveFile 将当前 latest.log 重命名为 date-NN.log
func archiveFile(date string) {
	latest := filepath.Join(logDir, "latest.log")
	if _, err := os.Stat(latest); err != nil {
		return
	}
	n := 1
	for {
		name := filepath.Join(logDir, fmt.Sprintf("%s-%d.log", date, n))
		if _, err := os.Stat(name); err != nil {
			_ = os.Rename(latest, name)
			return
		}
		n++
	}
}

// openLatest 打开（或创建）latest.log 供追加写入
func openLatest() {
	latest := filepath.Join(logDir, "latest.log")
	f, err := os.OpenFile(latest, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fileReady = false
		return
	}
	logFile = f
	fileDate = time.Now().Format("2006-01-02")
	fileReady = true
}

// Info 记录信息日志
func Info(format string, args ...any) {
	log(LevelInfo, fmt.Sprintf(format, args...))
}

// Error 记录错误日志
func Error(format string, args ...any) {
	log(LevelError, fmt.Sprintf(format, args...))
}

// LogFileInfo 日志文件信息
type LogFileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

// ListLogFiles 返回日志目录下所有 .log 文件，按修改时间倒序
func ListLogFiles() []LogFileInfo {
	mu.Lock()
	dir := logDir
	mu.Unlock()
	if dir == "" {
		return []LogFileInfo{}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []LogFileInfo{}
	}
	var out []LogFileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".log" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, LogFileInfo{
			Name:    name,
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	// 按名称倒序（latest.log 在前，归档文件按日期-序号顺序）
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Name > out[i].Name {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// ReadLogFile 读取指定日志文件内容；tail>0 时只返回最后 tail 行
func ReadLogFile(name string, tail int) (string, error) {
	mu.Lock()
	dir := logDir
	mu.Unlock()
	if dir == "" {
		return "", fmt.Errorf("log dir not initialized")
	}
	// 防止路径穿越
	clean := filepath.Base(name)
	if clean != name {
		return "", fmt.Errorf("invalid file name")
	}
	path := filepath.Join(dir, clean)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(data)
	if tail > 0 {
		lines := splitLines(content)
		if len(lines) > tail {
			lines = lines[len(lines)-tail:]
		}
		content = ""
		for i, l := range lines {
			if i > 0 {
				content += "\n"
			}
			content += l
		}
	}
	return content, nil
}

// splitLines 按换行符拆分（保留空行，去掉行尾换行符）
func splitLines(s string) []string {
	var lines []string
	cur := ""
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\n' {
			lines = append(lines, cur)
			cur = ""
		} else if c != '\r' {
			cur += string(c)
		}
	}
	if cur != "" || (len(s) > 0 && s[len(s)-1] != '\n') {
		lines = append(lines, cur)
	}
	return lines
}
