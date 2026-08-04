//go:build !windows

package logger

// enableVT 在非 Windows 平台上直接返回 true（POSIX 终端原生支持 ANSI）。
func enableVT() bool {
	return true
}
