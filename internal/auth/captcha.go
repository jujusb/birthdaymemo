package auth

import (
	"bytes"
	crand "crypto/rand"
	"encoding/base64"
	"math/big"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"image"
	"image/color"
	"image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const (
	captchaLen    = 4
	captchaTTL    = 5 * time.Minute
	captchaWidth  = 130
	captchaHeight = 48
)

// captchaChars 验证码字符集：大小写字母 + 数字，去除歧义字符
// 去掉：大写 I/O（与 1/0 混淆）、小写 l（与 1 混淆）、数字 0/1
var captchaChars = []byte("ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789")

type captchaEntry struct {
	code      string
	expiresAt time.Time
}

var (
	captchas = make(map[string]*captchaEntry)
	capMu    sync.Mutex
	fontFace *sfnt.Font
)

func init() {
	f, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		panic(err)
	}
	fontFace = f
}

// CreateCaptcha 生成一个新的图形验证码，返回 id 与 base64 编码的 PNG 图片
// 注意：code 仅在服务端保存，不返回给前端
func CreateCaptcha() (id, code, imageBase64 string, err error) {
	code = randomCaptchaCode(captchaLen)
	id = randomHex(16)

	capMu.Lock()
	captchas[id] = &captchaEntry{code: code, expiresAt: time.Now().Add(captchaTTL)}
	now := time.Now()
	for k, v := range captchas {
		if now.After(v.expiresAt) {
			delete(captchas, k)
		}
	}
	capMu.Unlock()

	img, err := drawCaptcha(code)
	if err != nil {
		return "", "", "", err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", "", "", err
	}
	imageBase64 = base64.StdEncoding.EncodeToString(buf.Bytes())
	return id, code, imageBase64, nil
}

// VerifyCaptcha 校验验证码（大小写不敏感），成功后立即删除（一次性）
func VerifyCaptcha(id, code string) bool {
	capMu.Lock()
	defer capMu.Unlock()
	entry, ok := captchas[id]
	if !ok {
		return false
	}
	delete(captchas, id)
	if time.Now().After(entry.expiresAt) {
		return false
	}
	// 大小写不敏感比较，便于用户输入
	return strings.EqualFold(entry.code, code)
}

// randomCaptchaCode 生成 n 位字母+数字验证码（使用 crypto/rand 保证安全）
func randomCaptchaCode(n int) string {
	b := make([]byte, n)
	for i := range b {
		num, err := crand.Int(crand.Reader, big.NewInt(int64(len(captchaChars))))
		if err != nil {
			b[i] = captchaChars[0]
			continue
		}
		b[i] = captchaChars[num.Int64()]
	}
	return string(b)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := crand.Read(b); err != nil {
		return ""
	}
	return bytesToHex(b)
}

func bytesToHex(b []byte) string {
	const hexd = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = hexd[c>>4]
		out[i*2+1] = hexd[c&0xf]
	}
	return string(out)
}

// drawCaptcha 绘制验证码图片
func drawCaptcha(code string) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
	for y := 0; y < captchaHeight; y++ {
		for x := 0; x < captchaWidth; x++ {
			img.Set(x, y, color.RGBA{R: 245, G: 247, B: 250, A: 255})
		}
	}

	// 干扰线
	for i := 0; i < 6; i++ {
		x1 := rand.IntN(captchaWidth)
		y1 := rand.IntN(captchaHeight)
		x2 := rand.IntN(captchaWidth)
		y2 := rand.IntN(captchaHeight)
		drawLine(img, x1, y1, x2, y2, randomLightColor())
	}
	// 噪点
	for i := 0; i < 80; i++ {
		x := rand.IntN(captchaWidth)
		y := rand.IntN(captchaHeight)
		img.Set(x, y, randomDarkColor())
	}

	face, err := opentype.NewFace(fontFace, &opentype.FaceOptions{
		Size:    26,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return img, err
	}
	defer face.Close()

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: face,
	}
	charW := captchaWidth / captchaLen
	for i, c := range code {
		x := i*charW + rand.IntN(8) + 6
		y := captchaHeight/2 + 8 + rand.IntN(6) - 3
		d.Dot = fixed.P(x, y)
		d.Src = image.NewUniform(randomDarkColor())
		d.DrawString(string(c))
	}
	return img, nil
}

func randomLightColor() color.Color {
	return color.RGBA{
		R: uint8(rand.IntN(156) + 100),
		G: uint8(rand.IntN(156) + 100),
		B: uint8(rand.IntN(156) + 100),
		A: 255,
	}
}

func randomDarkColor() color.Color {
	return color.RGBA{
		R: uint8(rand.IntN(120)),
		G: uint8(rand.IntN(120)),
		B: uint8(rand.IntN(120)),
		A: 255,
	}
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy
	for {
		if x1 >= 0 && x1 < captchaWidth && y1 >= 0 && y1 < captchaHeight {
			img.Set(x1, y1, c)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
