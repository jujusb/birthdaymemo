package pdfexport

import "embed"

//go:embed fonts/*.ttf
var fontsFS embed.FS

// PresetFont 预设字体信息
type PresetFont struct {
	Key    string `json:"key"`    // 唯一标识
	Name   string `json:"name"`   // 显示名称
	Style  string `json:"style"`  // 样式描述
	Family string `json:"family"` // PDF 字体族名（内部使用）
}

// PresetFonts 预设字体列表（5 种，中英文兼容）
var PresetFonts = []PresetFont{
	{Key: "noto-sans-sc", Name: "思源黑体 Noto Sans SC", Style: "现代无衬线", Family: "fnoto"},
	{Key: "zcool-xiaowei", Name: "站酷小薇 ZCOOL XiaoWei", Style: "细衬线", Family: "fxw"},
	{Key: "zcool-kuaile", Name: "站酷快乐 ZCOOL KuaiLe", Style: "圆润活泼", Family: "fkl"},
	{Key: "ma-shan-zheng", Name: "马善政 Ma Shan Zheng", Style: "毛笔行书", Family: "fms"},
	{Key: "long-cang", Name: "龙藏 Long Cang", Style: "硬笔行楷", Family: "flc"},
}

// FontFileByKey 根据预设 key 返回对应 TTF 字节
func FontFileByKey(key string) []byte {
	name := ""
	switch key {
	case "noto-sans-sc":
		name = "fonts/NotoSansSC.ttf"
	case "zcool-xiaowei":
		name = "fonts/ZCOOLXiaoWei.ttf"
	case "zcool-kuaile":
		name = "fonts/ZCOOLKuaiLe.ttf"
	case "ma-shan-zheng":
		name = "fonts/MaShanZheng.ttf"
	case "long-cang":
		name = "fonts/LongCang.ttf"
	default:
		return nil
	}
	data, err := fontsFS.ReadFile(name)
	if err != nil {
		return nil
	}
	return data
}

// FamilyByKey 根据预设 key 返回字体族名
func FamilyByKey(key string) string {
	for _, f := range PresetFonts {
		if f.Key == key {
			return f.Family
		}
	}
	return ""
}

// DefaultFontBytes 返回默认字体字节（站酷小薇，覆盖中英文且为静态字体不触发 fpdf CID 限制）
func DefaultFontBytes() []byte {
	return FontFileByKey("zcool-xiaowei")
}

// DefaultFontFamily 默认字体族名
const DefaultFontFamily = "fdef"
