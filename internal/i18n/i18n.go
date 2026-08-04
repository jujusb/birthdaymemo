package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

//go:embed langs/en.json langs/zh.json
var builtinFS embed.FS

// MetaLanguageNameKey 语言文件中表示自身名称的保留键
const MetaLanguageNameKey = "_language_name"

// translations 某语言所有翻译
type translations map[string]string

// Language 可用语言信息
type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var (
	mu    sync.RWMutex
	store = map[string]translations{} // code -> translations
	order []string                    // 加载顺序，内置优先
)

func init() {
	if t, name, ok := loadBuiltin("en"); ok {
		store["en"] = t
		_ = name
		order = append(order, "en")
	}
	if t, name, ok := loadBuiltin("zh"); ok {
		store["zh"] = t
		_ = name
		order = append(order, "zh")
	}
}

// loadBuiltin 从 embed 读取内置语言文件
func loadBuiltin(code string) (translations, string, bool) {
	data, err := builtinFS.ReadFile("langs/" + code + ".json")
	if err != nil {
		return nil, "", false
	}
	return parseFile(data)
}

// parseFile 解析语言 JSON，返回翻译与自身名称
func parseFile(data []byte) (translations, string, bool) {
	var t translations
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, "", false
	}
	name := t[MetaLanguageNameKey]
	if name == "" {
		name = "Unknown"
	}
	return t, name, true
}

// LoadUserDir 从指定目录加载用户追加的语言文件（en/zh 之外）。
// 返回加载错误列表（不影响已加载的其他语言）。
func LoadUserDir(dir string) []error {
	var errs []error
	entries, err := os.ReadDir(dir)
	if err != nil {
		// 目录不存在不算错误，自动创建
		if os.IsNotExist(err) {
			_ = os.MkdirAll(dir, 0o755)
			return nil
		}
		return []error{err}
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		code := strings.TrimSuffix(name, ".json")
		if code == "en" || code == "zh" || !isValidCode(code) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		if err := ValidateFile(data, "en"); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		t, name, ok := parseFile(data)
		if !ok {
			errs = append(errs, fmt.Errorf("%s: invalid json", name))
			continue
		}
		mu.Lock()
		if _, exists := store[code]; !exists {
			order = append(order, code)
		}
		store[code] = t
		mu.Unlock()
		_ = name
	}
	return errs
}

// isValidCode 校验语言代码仅含小写字母与连字符
func isValidCode(code string) bool {
	if code == "" {
		return false
	}
	for _, r := range code {
		if !(r >= 'a' && r <= 'z') && r != '-' {
			return false
		}
	}
	return true
}

// T 翻译指定语言的键，支持 fmt 参数；找不到时回退到 en 再回退到 key
func T(lang, key string, args ...any) string {
	mu.RLock()
	defer mu.RUnlock()
	t, ok := store[lang]
	if !ok {
		t = store["en"]
	}
	if t == nil {
		if len(args) > 0 {
			return fmt.Sprintf(key, args...)
		}
		return key
	}
	if s, ok := t[key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(s, args...)
		}
		return s
	}
	if lang != "en" {
		if s, ok := store["en"][key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(s, args...)
			}
			return s
		}
	}
	return key
}

// Strings 返回指定语言的全部翻译字符串（找不到时回退到 en）
func Strings(lang string) map[string]string {
	mu.RLock()
	defer mu.RUnlock()
	if t, ok := store[lang]; ok {
		return t
	}
	if t, ok := store["en"]; ok {
		return t
	}
	return map[string]string{}
}

// Available 返回全部可用语言（内置 + 用户追加），按加载顺序
func Available() []Language {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Language, 0, len(order))
	for _, code := range order {
		t := store[code]
		name := "Unknown"
		if t != nil {
			if n := t[MetaLanguageNameKey]; n != "" {
				name = n
			}
		}
		out = append(out, Language{Code: code, Name: name})
	}
	return out
}

// ReferenceKeys 返回参考语言(en)的全部键（已排序）
func ReferenceKeys() []string {
	mu.RLock()
	defer mu.RUnlock()
	keys := make([]string, 0, len(store["en"]))
	for k := range store["en"] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ValidateFile 校验语言文件内容是否包含参考语言(en)的全部键。
// 返回缺失键的错误（列出最多 20 个）。
func ValidateFile(data []byte, refCode string) error {
	var t translations
	if err := json.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	mu.RLock()
	ref := store[refCode]
	mu.RUnlock()
	if ref == nil {
		return fmt.Errorf("reference language %s not found", refCode)
	}
	var missing []string
	for k := range ref {
		if _, ok := t[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		if len(missing) > 20 {
			missing = missing[:20]
		}
		return fmt.Errorf("missing keys: %s", strings.Join(missing, ", "))
	}
	return nil
}

// SampleFile 返回英文版语言文件的格式化 JSON 字节（供下载作为翻译模板）
func SampleFile() []byte {
	mu.RLock()
	t := store["en"]
	mu.RUnlock()
	if t == nil {
		return []byte("{}")
	}
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("{\n")
	for i, k := range keys {
		b.WriteString("  ")
		kb, _ := json.Marshal(k)
		b.Write(kb)
		b.WriteString(": ")
		vb, _ := json.Marshal(t[k])
		b.Write(vb)
		if i < len(keys)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("}")
	return []byte(b.String())
}
