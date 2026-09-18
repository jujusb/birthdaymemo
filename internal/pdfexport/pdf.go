package pdfexport

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/models"
	_ "golang.org/x/image/webp" // 注册 WebP 解码器
)

// RangeType 导出范围类型
const (
	RangeYear       = "year"
	RangeMonth      = "month"
	RangeSchoolYear = "school_year" // 学年：进行中的学年（9 月起共 12 页；未到 9 月则从去年 9 月起）
)

// Range 导出日期范围
// 注：年份不从前端传入，由后端使用 time.Now().Year() 自动填充
type Range struct {
	Type  string `json:"type"`
	Month int    `json:"month,omitempty"` // 1-12，仅 type=month 使用
	Year  int    `json:"-"`               // 仅内部使用，由后端填充
	Lang  string `json:"lang,omitempty"`  // 前端 UI 语言（星期表头/脚注/默认标题本地化），空则回退 en
}

// BirthdayEntry PDF 导出用生日条目
type BirthdayEntry struct {
	Name       string `json:"name"`
	Gender     string `json:"gender"`
	BirthYear  int    `json:"birth_year"`
	BirthMonth int    `json:"birth_month"`
	BirthDay   int    `json:"birth_day"`
}

// Resources 单次使用的资源（字体/背景图/副标题图），用完销毁，不会持久化
// 字体优先级：CustomBytes > PresetKey > 默认（站酷小薇）
// 任务5：副标题通过 SubtitleImage（PNG 字节）渲染，使用用户端浏览器字体
type Resources struct {
	TitleFontKey  string
	TableFontKey  string
	TitleFont     []byte // 自定义上传字体（base64 解码后）
	TableFont     []byte
	BgImage       []byte // 背景图（任意常见格式：PNG/JPEG/WebP/GIF，自动转 PNG）
	SubtitleImage []byte // 任务5：副标题 PNG（浏览器字体渲染的图片）
}

// monthShortNames 英文月份缩写（用于 {month} 变量）
var monthShortNames = []string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}

// Generator PDF 生成器
type Generator struct{}

// NewGenerator 创建生成器
func NewGenerator() *Generator {
	return &Generator{}
}

// resolveFontBytes 解析字体字节：自定义 > 预设 > 默认
func resolveFontBytes(custom []byte, key string) []byte {
	if len(custom) > 0 {
		return custom
	}
	if key != "" {
		if b := FontFileByKey(key); b != nil {
			return b
		}
	}
	return DefaultFontBytes()
}

// Generate 生成 PDF，返回字节
// 注：fpdf v0.9.0 在处理超过 65536 字符的字体（如 Noto Sans SC 变量字体）时会 panic
// 此处 recover 防止服务器进程崩溃，转为返回错误
func (g *Generator) Generate(r Range, entries []BirthdayEntry, ps models.PdfSetting, now time.Time, res Resources) (data []byte, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("pdf generation panic: %v", rec)
			data = nil
		}
	}()

	// 年份由后端从当前时间填充（前端不传 year）
	if r.Year == 0 {
		r.Year = now.Year()
	}

	// 本地化语言：未知/空一律回退 en（i18n.T 自带回退）
	lang := normalizeLang(r.Lang)

	// 横版 A4
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 10)

	// 默认字体族（站酷小薇，覆盖中英文）
	defFont := DefaultFontBytes()
	pdf.AddUTF8FontFromBytes(DefaultFontFamily, "", defFont)
	pdf.AddUTF8FontFromBytes(DefaultFontFamily, "B", defFont)
	pdf.SetFont(DefaultFontFamily, "", 10)

	// 注册用户选择的字体（标题/表格，副标题固定使用 Helvetica）
	titleFam := DefaultFontFamily
	if titleBytes := resolveFontBytes(res.TitleFont, res.TitleFontKey); titleBytes != nil && !bytes.Equal(titleBytes, defFont) {
		pdf.AddUTF8FontFromBytes("ftitle", "", titleBytes)
		pdf.AddUTF8FontFromBytes("ftitle", "B", titleBytes)
		titleFam = "ftitle"
	}
	tblFam := DefaultFontFamily
	if tblBytes := resolveFontBytes(res.TableFont, res.TableFontKey); tblBytes != nil && !bytes.Equal(tblBytes, defFont) {
		pdf.AddUTF8FontFromBytes("ftbl", "", tblBytes)
		pdf.AddUTF8FontFromBytes("ftbl", "B", tblBytes)
		tblFam = "ftbl"
	}

	if r.Type == RangeMonth {
		g.drawMonth(pdf, r.Year, r.Month, entries, ps, now, res, titleFam, tblFam, lang)
	} else if r.Type == RangeSchoolYear {
		// 学年：始终导出进行中的学年（9 月起共 12 页）。
		// 当前月份 >= 9 月：当年 9 月至次年 8 月；否则：去年 9 月至当年 8 月。
		startYear := r.Year
		if now.Month() < time.September {
			startYear--
		}
		for i := 0; i < 12; i++ {
			m := 9 + i
			y := startYear
			if m > 12 {
				m -= 12
				y++
			}
			g.drawMonth(pdf, y, m, entries, ps, now, res, titleFam, tblFam, lang)
		}
	} else {
		for m := 1; m <= 12; m++ {
			g.drawMonth(pdf, r.Year, m, entries, ps, now, res, titleFam, tblFam, lang)
		}
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// normalizeLang 规范化语言代码：去空格转小写，空则回退 en
// （未知语言由 i18n.T 自动回退 en）
func normalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return "en"
	}
	return lang
}

// drawMonth 绘制单月日历页（横版 A4，新版日历网格布局）
func (g *Generator) drawMonth(pdf *fpdf.Fpdf, year, month int, entries []BirthdayEntry, ps models.PdfSetting, now time.Time, res Resources, titleFam, tblFam, lang string) {
	pdf.AddPage()
	w, h := pdf.GetPageSize() // 297 x 210

	// 1. 背景
	g.drawBackground(pdf, w, h, ps, res)

	margin := 10.0
	innerW := w - margin*2

	// 当月生日
	var monthBd []BirthdayEntry
	for _, e := range entries {
		if e.BirthMonth == month {
			monthBd = append(monthBd, e)
		}
	}
	sortByDay(monthBd)

	// 2. 大行：月份文字（居中）
	titleText := applyTitleVars(ps.TitleText, month, year, len(monthBd), lang)
	pdf.SetFont(titleFam, "B", 24)
	pdf.SetTextColor(parseRGB(ps.TextColor))
	pdf.SetXY(margin, margin)
	pdf.CellFormat(innerW, 14, titleText, "", 1, "C", false, 0, "")

	// 3. 小行：副标题（任务5：使用浏览器字体渲染的图片，避免 fpdf 无法渲染 emoji）
	// 前端通过 Canvas 使用浏览器字体渲染副标题为 PNG，发送给后端嵌入 PDF
	// 这样预览（HTML）和下载（PDF）都使用同一字体，达到一致效果
	if len(res.SubtitleImage) > 0 {
		tmp, err := os.CreateTemp("", "bdmsub-*.png")
		if err == nil {
			if _, werr := tmp.Write(res.SubtitleImage); werr == nil {
				_ = tmp.Close()
				defer os.Remove(tmp.Name())
				opt := fpdf.ImageOptions{ImageType: "png", ReadDpi: false}
				// 副标题区域：与原文字位置一致，277mm × 6mm，居中
				pdf.ImageOptions(tmp.Name(), margin, margin+15, innerW, 6, false, opt, 0, "")
			} else {
				_ = tmp.Close()
				// 图片写入失败，降级为文字
				drawSubtitleText(pdf, ps, tblFam, margin, innerW)
			}
		} else {
			// 临时文件创建失败，降级为文字
			drawSubtitleText(pdf, ps, tblFam, margin, innerW)
		}
	} else {
		// 无图片资源，降级为文字（sanitize 过滤 emoji）
		drawSubtitleText(pdf, ps, tblFam, margin, innerW)
	}

	// 4. 日历表格区域（外圆角框 + 内部网格）
	calendarTopY := margin + 24
	footnoteH := 14.0
	calendarBottomY := h - margin - footnoteH
	calendarH := calendarBottomY - calendarTopY

	headerH := 8.0
	rowsCount := 6
	rowH := (calendarH - headerH) / float64(rowsCount)
	colW := innerW / 7.0
	gap := 1.2 // 单元格间隙

	// 透明度：外框 TableOpacity/100，内格 CellOpacity/100（任务6：拆分为两类）
	outerAlpha := float64(ps.TableOpacity) / 100.0
	if outerAlpha > 1.0 {
		outerAlpha = 1.0
	}
	if outerAlpha < 0.0 {
		outerAlpha = 0.0
	}
	cellAlpha := float64(ps.CellOpacity) / 100.0
	if cellAlpha > 1.0 {
		cellAlpha = 1.0
	}
	if cellAlpha < 0.0 {
		cellAlpha = 0.0
	}

	tblR, tblG, tblB := parseRGB(ps.TableBgColor)
	cellR, cellG, cellB := parseRGB(ps.CellBgColor)

	// 4.1 绘制外圆角框（应用透明度 + 任务6：可选边框）
	pdf.SetAlpha(outerAlpha, "Normal")
	pdf.SetFillColor(tblR, tblG, tblB)
	pdf.RoundedRect(margin, calendarTopY, innerW, calendarH, 3, "1234", "F")
	pdf.SetAlpha(1.0, "Normal")
	// 任务6：外框边框（可选）
	if ps.TableBorderEnabled {
		borderAlpha := float64(ps.TableBorderOpacity) / 100.0
		if borderAlpha > 1.0 {
			borderAlpha = 1.0
		}
		bR, bG, bB := parseRGB(ps.TableBorderColor)
		pdf.SetAlpha(borderAlpha, "Normal")
		pdf.SetDrawColor(bR, bG, bB)
		pdf.SetLineWidth(0.4)
		pdf.RoundedRect(margin, calendarTopY, innerW, calendarH, 3, "1234", "D")
		pdf.SetAlpha(1.0, "Normal")
	}

	// 4.2 表头（星期：按导出语言本地化，而非硬编码中文）
	weekdayKeys := []string{
		"calendar.sun", "calendar.mon", "calendar.tue", "calendar.wed",
		"calendar.thu", "calendar.fri", "calendar.sat",
	}
	pdf.SetFont(tblFam, "B", 12)
	pdf.SetTextColor(parseRGB(ps.TextColor))
	for i, key := range weekdayKeys {
		pdf.SetXY(margin+float64(i)*colW, calendarTopY)
		pdf.CellFormat(colW, headerH, i18n.T(lang, key), "", 0, "C", false, 0, "")
	}

	// 4.3 计算月份信息
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, now.Location())
	daysInMonthCount := daysInMonth(year, month)
	firstWeekday := int(firstDay.Weekday()) // 0=Sunday
	grouped := groupByDay(monthBd)

	// 收集溢出脚注
	type footnote struct {
		Index int
		Day   int
		Names []string
	}
	var footnotes []footnote
	footnoteCounter := 0

	// 4.4 绘制日历单元格
	// 任务4：非当月日期不预留方格 — 不绘制背景/边框
	y := calendarTopY + headerH
	for row := 0; row < rowsCount; row++ {
		for col := 0; col < 7; col++ {
			cellX := margin + float64(col)*colW
			dayNum := row*7 + col - firstWeekday + 1

			// 任务4：非当月日期不预留方格（不绘制背景和边框）
			if dayNum < 1 || dayNum > daysInMonthCount {
				continue
			}

			// 单元格内部位置（留出间隙）
			cellXPad := cellX + gap
			cellYPad := y + gap
			cellW := colW - gap*2
			cellHPad := rowH - gap*2

			// 单元格背景（任务6：使用 CellBgColor + CellOpacity）
			pdf.SetAlpha(cellAlpha, "Normal")
			pdf.SetFillColor(cellR, cellG, cellB)
			pdf.RoundedRect(cellXPad, cellYPad, cellW, cellHPad, 1.5, "1234", "F")
			pdf.SetAlpha(1.0, "Normal")

			// 任务6：单元格边框（可选）
			if ps.CellBorderEnabled {
				cellBorderAlpha := float64(ps.CellBorderOpacity) / 100.0
				if cellBorderAlpha > 1.0 {
					cellBorderAlpha = 1.0
				}
				cbR, cbG, cbB := parseRGB(ps.CellBorderColor)
				pdf.SetAlpha(cellBorderAlpha, "Normal")
				pdf.SetDrawColor(cbR, cbG, cbB)
				pdf.SetLineWidth(0.2)
				pdf.RoundedRect(cellXPad, cellYPad, cellW, cellHPad, 1.5, "1234", "D")
				pdf.SetAlpha(1.0, "Normal")
			}

			items := grouped[dayNum]

			// 日期数字（居中上方，加大字号）
			pdf.SetTextColor(parseRGB(ps.TextColor))
			pdf.SetFont(tblFam, "B", 14)
			pdf.SetXY(cellXPad, cellYPad+1)
			pdf.CellFormat(cellW, 6, strconv.Itoa(dayNum), "", 0, "C", false, 0, "")

			// 索引数字（如果 >2 个生日，右上角 [N]）
			if len(items) > 2 {
				footnoteCounter++
				idxStr := fmt.Sprintf("[%d]", footnoteCounter)
				pdf.SetFont(tblFam, "", 8)
				pdf.SetXY(cellXPad+cellW-9, cellYPad+1)
				pdf.CellFormat(8, 4, idxStr, "", 0, "R", false, 0, "")

				// 显示前 2 个名字（左对齐 + 性别色圆点）
				pdf.SetFont(tblFam, "", 10)
				nameY := cellYPad + 9
				for i := 0; i < 2 && i < len(items); i++ {
					drawNameWithGenderDot(pdf, items[i], cellXPad+1, nameY, cellW-2, ageSuffix(items[i], year, ps.ShowAge, ps.ShowBirthYear))
					nameY += 5
				}

				var names []string
				for _, it := range items {
					names = append(names, it.Name+ageSuffix(it, year, ps.ShowAge, ps.ShowBirthYear))
				}
				footnotes = append(footnotes, footnote{Index: footnoteCounter, Day: dayNum, Names: names})
			} else if len(items) > 0 {
				// 显示所有名字（左对齐 + 性别色圆点）
				pdf.SetFont(tblFam, "", 10)
				nameY := cellYPad + 9
				for _, it := range items {
					drawNameWithGenderDot(pdf, it, cellXPad+1, nameY, cellW-2, ageSuffix(it, year, ps.ShowAge, ps.ShowBirthYear))
					nameY += 5
				}
			}
		}
		y += rowH
	}

	// 5. 脚注（按导出语言本地化）
	if len(footnotes) > 0 {
		pdf.SetFont(tblFam, "", 9)
		pdf.SetTextColor(parseRGB(ps.TextColor))
		footY := h - margin - footnoteH + 2
		for _, fn := range footnotes {
			txt := strings.NewReplacer(
				"{index}", strconv.Itoa(fn.Index),
				"{day}", strconv.Itoa(fn.Day),
				"{names}", strings.Join(fn.Names, ", "),
			).Replace(i18n.T(lang, "export.footnote"))
			pdf.SetXY(margin, footY)
			pdf.MultiCell(innerW, 4, txt, "", "L", false)
			footY += 4
			if footY > h-margin-2 {
				break
			}
		}
	}
}

// genderRGB 根据 gender 字符串返回 RGB 颜色
// male=#4A90D9(74,144,217) / female=#E91E63(233,30,99) / none=#9E9E9E(158,158,158)
func genderRGB(g string) (int, int, int) {
	switch g {
	case "male":
		return 74, 144, 217
	case "female":
		return 233, 30, 99
	default: // "none" 或空值
		return 158, 158, 158
	}
}

// drawSubtitleText 降级路径：副标题用 fpdf 文字渲染（过滤 emoji）
// 任务5：主路径使用 SubtitleImage 图片渲染；此函数仅在图片缺失/失败时使用
func drawSubtitleText(pdf *fpdf.Fpdf, ps models.PdfSetting, tblFam string, margin, innerW float64) {
	subtitleText := sanitizePDFText(ps.SubtitleText)
	if subtitleText == "" {
		subtitleText = "BirthDayMemo"
	}
	pdf.SetFont(tblFam, "", 11)
	pdf.SetTextColor(parseRGB(ps.TextColor))
	pdf.SetXY(margin, margin+15)
	pdf.CellFormat(innerW, 6, subtitleText, "", 0, "C", false, 0, "")
}

// ageSuffix 返回姓名后的年龄/年份后缀：两项都开为 " (35 · 1990)"，
// 仅年龄为 " (35)"，仅年份为 " (1990)"；都关闭或出生年份未知时返回空串。
// 年龄指该年份生日时满的岁数（pageYear - birthYear）。
func ageSuffix(entry BirthdayEntry, pageYear int, showAge, showBirthYear bool) string {
	if (!showAge && !showBirthYear) || entry.BirthYear <= 0 {
		return ""
	}
	age := pageYear - entry.BirthYear
	if age < 0 {
		return ""
	}
	switch {
	case showAge && showBirthYear:
		return " (" + strconv.Itoa(age) + " · " + strconv.Itoa(entry.BirthYear) + ")"
	case showAge:
		return " (" + strconv.Itoa(age) + ")"
	default:
		return " (" + strconv.Itoa(entry.BirthYear) + ")"
	}
}

// drawNameWithGenderDot 在单元格内左对齐绘制"性别色圆点 + 姓名"
// x,y 为名字区域的左上角；w 为可用宽度；先画一个小圆点，再画名字（左对齐）
// suffix 为姓名后缀（如年龄），随名字一起截断以适应剩余宽度
// 名字会被自动截断以适应剩余宽度
func drawNameWithGenderDot(pdf *fpdf.Fpdf, entry BirthdayEntry, x, y, w float64, suffix string) {
	// 1. 性别色圆点（半径 0.8mm，垂直居中于文字行）
	dotR := 0.8
	dotCx := x + dotR + 0.3
	dotCy := y + 2.5 // 文字行高约 5mm，居中即 +2.5
	r, g, b := genderRGB(entry.Gender)
	pdf.SetFillColor(r, g, b)
	pdf.SetDrawColor(r, g, b)
	pdf.Circle(dotCx, dotCy, dotR, "F")

	// 2. 姓名（左对齐，留出圆点 + 间距的空间）
	textX := dotCx + dotR + 0.6
	availW := w - (textX - x)
	if availW < 1 {
		availW = 1
	}
	pdf.SetXY(textX, y)
	pdf.CellFormat(availW, 5, truncateName(entry.Name+suffix, availW), "", 0, "L", false, 0, "")
}

// applyTitleVars 替换标题中的变量（与前端 ExportView 的 applyTitleVars 对齐）：
// {month} / {monthShort}=英文月份缩写, {monthNum}=月份数字, {count}=当月人数, {year}=年份
func applyTitleVars(text string, month, year, count int, lang string) string {
	if text == "" || text == models.DefaultPdfTitleText {
		// 空或沿用旧版中文默认值时，按导出语言取本地化默认标题
		text = i18n.T(lang, "export.defaultTitle")
	}
	monthName := ""
	if month >= 1 && month <= 12 {
		monthName = monthShortNames[month-1]
	}
	r := strings.NewReplacer(
		"{month}", monthName,
		"{monthShort}", monthName,
		"{monthNum}", strconv.Itoa(month),
		"{count}", strconv.Itoa(count),
		"{year}", strconv.Itoa(year),
	)
	return sanitizePDFText(r.Replace(text))
}

// sanitizePDFText 去除 fpdf 无法渲染的字形（BMP 之外的码点：emoji、稀有符号等）
// fpdf v0.9.0 的 CID 映射表硬编码使用 65536 长度的数组，超出会 panic
func sanitizePDFText(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r > 0xFFFF {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// daysInMonth 返回某年某月的天数
func daysInMonth(year, month int) int {
	if month < 1 || month > 12 {
		return 30
	}
	return time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
}

// truncateName 根据列宽截断姓名
func truncateName(name string, colW float64) string {
	name = sanitizePDFText(name)
	maxChars := int(colW / 3.2)
	if maxChars < 3 {
		maxChars = 3
	}
	runes := []rune(name)
	if len(runes) <= maxChars {
		return name
	}
	return string(runes[:maxChars-1]) + "…"
}

// drawBackground 绘制页面背景（支持任意常见图片格式，自动转 PNG）
func (g *Generator) drawBackground(pdf *fpdf.Fpdf, w, h float64, ps models.PdfSetting, res Resources) {
	switch ps.BackgroundType {
	case models.PdfBgSolid:
		r, gr, b := parseRGB(ps.BackgroundColor)
		pdf.SetFillColor(r, gr, b)
		pdf.Rect(0, 0, w, h, "F")
	case models.PdfBgGradient:
		g.drawGradient(pdf, w, h, ps.GradientStart, ps.GradientEnd)
	case models.PdfBgImage:
		if len(res.BgImage) > 0 {
			// 转换为 PNG（兼容 JPEG/PNG/WebP/GIF 等）
			pngData := convertImageToPNG(res.BgImage)
			if ps.Blur > 0 {
				pngData = blurImage(pngData, ps.Blur)
			}
			// 写入临时文件用于嵌入
			tmp, err := os.CreateTemp("", "bdmbg-*.png")
			if err == nil {
				if _, err := tmp.Write(pngData); err == nil {
					_ = tmp.Close()
					defer os.Remove(tmp.Name())
					opt := fpdf.ImageOptions{ImageType: "png", ReadDpi: false}
					pdf.ImageOptions(tmp.Name(), 0, 0, w, h, false, opt, 0, "")
				} else {
					_ = tmp.Close()
					pdf.SetFillColor(255, 255, 255)
					pdf.Rect(0, 0, w, h, "F")
				}
			} else {
				pdf.SetFillColor(255, 255, 255)
				pdf.Rect(0, 0, w, h, "F")
			}
		} else {
			pdf.SetFillColor(255, 255, 255)
			pdf.Rect(0, 0, w, h, "F")
		}
	default: // white
		pdf.SetFillColor(255, 255, 255)
		pdf.Rect(0, 0, w, h, "F")
	}
}

// convertImageToPNG 将任意常见图片格式（PNG/JPEG/GIF 等）转换为 PNG 字节
// 失败时返回原始数据（让下游报错），同时通过 image.DecodeConfig 也会失败
// 注：WebP 需要 golang.org/x/image/webp 包，已通过 _ import 引入
func convertImageToPNG(data []byte) []byte {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data // 解码失败，返回原数据让下游处理
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return data
	}
	return buf.Bytes()
}

// drawGradient 纵向渐变（条带插值）
func (g *Generator) drawGradient(pdf *fpdf.Fpdf, w, h float64, startHex, endHex string) {
	r1, g1, b1 := parseRGB(startHex)
	r2, g2, b2 := parseRGB(endHex)
	steps := 64
	bandH := h / float64(steps)
	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps)
		r := lerp(r1, r2, t)
		gr := lerp(g1, g2, t)
		b := lerp(b1, b2, t)
		pdf.SetFillColor(r, gr, b)
		pdf.Rect(0, float64(i)*bandH, w, bandH+0.5, "F")
	}
}

// blurImage 对图片数据做盒模糊
func blurImage(data []byte, radius int) []byte {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}
	boxBlurred := boxBlur(img, radius)
	var buf bytes.Buffer
	if err := png.Encode(&buf, boxBlurred); err != nil {
		return data
	}
	return buf.Bytes()
}

// boxBlur 简单盒模糊
func boxBlur(src image.Image, radius int) image.Image {
	if radius < 1 {
		return src
	}
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var r, g, b, a, count uint32
			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					nx := x + dx
					ny := y + dy
					if nx < bounds.Min.X || nx >= bounds.Max.X || ny < bounds.Min.Y || ny >= bounds.Max.Y {
						continue
					}
					c := color.NRGBAModel.Convert(src.At(nx, ny)).(color.NRGBA)
					r += uint32(c.R)
					g += uint32(c.G)
					b += uint32(c.B)
					a += uint32(c.A)
					count++
				}
			}
			if count == 0 {
				count = 1
			}
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(r / count), G: uint8(g / count), B: uint8(b / count), A: uint8(a / count),
			})
		}
	}
	return dst
}

// parseRGB 解析 #RRGGBB 为 0-255
func parseRGB(hexStr string) (int, int, int) {
	s := strings.TrimPrefix(hexStr, "#")
	if len(s) != 6 {
		return 51, 51, 51
	}
	r, err1 := strconv.ParseUint(s[0:2], 16, 8)
	g, err2 := strconv.ParseUint(s[2:4], 16, 8)
	b, err3 := strconv.ParseUint(s[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return 51, 51, 51
	}
	return int(r), int(g), int(b)
}

func lerp(a, b int, t float64) int {
	return int(math.Round(float64(a) + (float64(b)-float64(a))*t))
}

func sortByDay(items []BirthdayEntry) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].BirthDay < items[j-1].BirthDay; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func groupByDay(items []BirthdayEntry) map[int][]BirthdayEntry {
	m := map[int][]BirthdayEntry{}
	for _, it := range items {
		m[it.BirthDay] = append(m[it.BirthDay], it)
	}
	return m
}
