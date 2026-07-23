package pdfexport

import (
	"os"
	"testing"
	"time"

	"github.com/mcbill1/birthdaymemo/internal/models"
)

// TestGeneratePDFWithEmbeddedFonts 验证使用嵌入字体能正常生成 PDF
func TestGeneratePDFWithEmbeddedFonts(t *testing.T) {
	g := NewGenerator()
	entries := []BirthdayEntry{
		{Name: "张三", BirthMonth: 7, BirthDay: 15},
		{Name: "李四", BirthMonth: 7, BirthDay: 20},
		{Name: "王五", BirthMonth: 7, BirthDay: 20},
		{Name: "赵六", BirthMonth: 7, BirthDay: 20},
	}
	ps := models.PdfSetting{
		TitleText:      "[{month}] 当月 {count} 人过生日",
		SubtitleText:   "BirthDayMemo",
		BackgroundType: models.PdfBgWhite,
		TableEffect:    models.PdfEffectNone,
		TableOpacity:   100,
		TableBgColor:   "#ffffff",
		TextColor:      "#333333",
	}
	now := time.Now()
	r := Range{Type: RangeMonth, Year: now.Year(), Month: int(now.Month())}

	// 测试 1: 默认字体
	t.Run("default_font", func(t *testing.T) {
		data, err := g.Generate(r, entries, ps, now, Resources{})
		if err != nil {
			t.Fatalf("default font generate failed: %v", err)
		}
		if len(data) < 1000 {
			t.Fatalf("PDF too small: %d bytes", len(data))
		}
		t.Logf("default font PDF: %d bytes", len(data))
		writeTemp(t, "default.pdf", data)
	})

	// 测试 2: 每个预设字体（逐个测试，避免一个失败影响其他）
	for _, pf := range PresetFonts {
		pf := pf
		t.Run(pf.Key, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s PANIC: %v", pf.Key, r)
				}
			}()
			res := Resources{
				TitleFontKey: pf.Key,
				TableFontKey: pf.Key,
			}
			data, err := g.Generate(r, entries, ps, now, res)
			if err != nil {
				t.Fatalf("%s generate failed: %v", pf.Key, err)
			}
			if len(data) < 1000 {
				t.Fatalf("%s PDF too small: %d bytes", pf.Key, len(data))
			}
			t.Logf("%s PDF: %d bytes", pf.Key, len(data))
			writeTemp(t, pf.Key+".pdf", data)
		})
	}

	// 测试 3: 整年范围
	t.Run("full_year", func(t *testing.T) {
		data, err := g.Generate(Range{Type: RangeYear, Year: now.Year()}, entries, ps, now, Resources{})
		if err != nil {
			t.Fatalf("full year generate failed: %v", err)
		}
		t.Logf("full year PDF: %d bytes", len(data))
		writeTemp(t, "full_year.pdf", data)
	})

	// 测试 4: 渐变背景 + 磨砂玻璃特效
	t.Run("gradient_acrylic", func(t *testing.T) {
		ps2 := ps
		ps2.BackgroundType = models.PdfBgGradient
		ps2.GradientStart = "#e0c3fc"
		ps2.GradientEnd = "#8ec5fc"
		ps2.TableEffect = models.PdfEffectAcrylic
		ps2.TableOpacity = 50
		data, err := g.Generate(r, entries, ps2, now, Resources{})
		if err != nil {
			t.Fatalf("gradient acrylic failed: %v", err)
		}
		t.Logf("gradient acrylic PDF: %d bytes", len(data))
		writeTemp(t, "gradient_acrylic.pdf", data)
	})

	// 测试 5: 含 emoji 的文本（应被自动过滤，不应 panic）
	t.Run("emoji_in_subtitle", func(t *testing.T) {
		ps2 := ps
		ps2.SubtitleText = "🎂 BirthDayMemo 🎉"
		ps2.TitleText = "[{month}] 当月 {count} 人过生日 🎂"
		// 也测试带 emoji 的姓名
		emojiEntries := []BirthdayEntry{
			{Name: "张三🎉", BirthMonth: int(now.Month()), BirthDay: 15},
			{Name: "李四🎂", BirthMonth: int(now.Month()), BirthDay: 20},
		}
		data, err := g.Generate(r, emojiEntries, ps2, now, Resources{})
		if err != nil {
			t.Fatalf("emoji subtitle failed: %v", err)
		}
		t.Logf("emoji subtitle PDF: %d bytes", len(data))
		writeTemp(t, "emoji.pdf", data)
	})
}

func writeTemp(t *testing.T, name string, data []byte) {
	t.Helper()
	path := os.TempDir() + string(os.PathSeparator) + "bdm_test_" + name
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Logf("write temp %s failed: %v", path, err)
		return
	}
	t.Logf("written: %s", path)
}
