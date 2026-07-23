//go:build windows

package logger

import (
	"os"
	"syscall"
	"unsafe"
)

// enableVT 在 Windows 上启用控制台虚拟终端处理和 UTF-8 输出代码页。
// 返回是否成功启用 VT。Windows 10 之前版本不支持 VT，返回 false。
func enableVT() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")

	// 设置输出代码页为 UTF-8 (CP65001)，确保中文等非 ASCII 字符正确显示
	setCP := kernel32.NewProc("SetConsoleOutputCP")
	setCP.Call(65001)

	// 启用虚拟终端处理，使 ANSI 颜色码生效
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")
	handle := syscall.Handle(os.Stdout.Fd())

	var mode uint32
	r, _, _ := getMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return false
	}
	// ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	mode |= 0x0004
	r, _, _ = setMode.Call(uintptr(handle), uintptr(mode))
	return r != 0
}
