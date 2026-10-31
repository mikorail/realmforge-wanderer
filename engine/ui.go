package engine

import "strings"

// Screen is the whole terminal grid the host draws: map, sidebar, log and every menu.
const (
	SW = 112
	SH = 34
	MW = 80 // zone width
	MH = 25 // zone height
)

// Qud-like 16 colour palette plus the viridian backgrounds.
const (
	CBlack byte = iota
	CDarkGray
	CGray
	CWhite
	CDarkBlue
	CBlue
	CDarkGreen
	CGreen
	CDarkCyan
	CCyan
	CDarkRed
	CRed
	CDarkMagenta
	CMagenta
	CBrown
	CYellow
	CDarkOrange
	COrange
	CBg
	CBgLight
	CBgDark
	CBgSel
	CBgRed
	NColors
)

var Palette = [NColors]string{
	"#0f3b3a", "#155352", "#b1c9c3", "#ffffff", "#0048bd", "#0096ff", "#009403", "#00c420",
	"#40a4b9", "#77bfcf", "#a64a2e", "#d74200", "#b154cf", "#da5bd6", "#98875f", "#cfc041",
	"#f15f22", "#e99f10", "#041312", "#0d2b29", "#020a09", "#1b4d48", "#3a1410",
}

// Qud markup letters → palette
var colorCode = map[byte]byte{
	'k': CBlack, 'K': CDarkGray, 'y': CGray, 'Y': CWhite, 'b': CDarkBlue, 'B': CBlue, 'g': CDarkGreen, 'G': CGreen,
	'c': CDarkCyan, 'C': CCyan, 'r': CDarkRed, 'R': CRed, 'm': CDarkMagenta, 'M': CMagenta, 'w': CBrown, 'W': CYellow,
	'o': CDarkOrange, 'O': COrange,
}

type Screen struct {
	G []rune
	F []byte
	B []byte
}

func NewScreen() *Screen {
	return &Screen{G: make([]rune, SW*SH), F: make([]byte, SW*SH), B: make([]byte, SW*SH)}
}

func (s *Screen) Clear() {
	for i := range s.G {
		s.G[i] = ' '
		s.F[i] = CGray
		s.B[i] = CBg
	}
}

func (s *Screen) Put(x, y int, r rune, f, b byte) {
	if x < 0 || y < 0 || x >= SW || y >= SH {
		return
	}
	i := y*SW + x
	s.G[i] = r
	s.F[i] = f
	s.B[i] = b
}

func (s *Screen) Fill(x0, y0, w, h int, b byte) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			s.Put(x, y, ' ', CGray, b)
		}
	}
}

// Text draws Qud markup: {{R|red text}} and &R colour switches. Returns the x after the text.
func (s *Screen) Text(x, y int, str string, f byte) int {
	return s.TextB(x, y, str, f, 255, SW)
}

// TextB draws markup clipped at maxX with an optional background (255 = keep).
func (s *Screen) TextB(x, y int, str string, f, b byte, maxX int) int {
	for _, run := range parseMarkup(str, f) {
		for _, r := range run.s {
			if x >= maxX {
				return x
			}
			if x >= 0 && x < SW && y >= 0 && y < SH {
				i := y*SW + x
				s.G[i] = r
				s.F[i] = run.c
				if b != 255 {
					s.B[i] = b
				}
			}
			x++
		}
	}
	return x
}

type run struct {
	s string
	c byte
}

func parseMarkup(str string, def byte) []run {
	var out []run
	cur := def
	var stack []byte
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			out = append(out, run{b.String(), cur})
			b.Reset()
		}
	}
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c == '{' && i+3 < len(str) && str[i+1] == '{' && str[i+3] == '|' {
			if col, ok := colorCode[str[i+2]]; ok {
				flush()
				stack = append(stack, cur)
				cur = col
				i += 3
				continue
			}
		}
		if c == '}' && i+1 < len(str) && str[i+1] == '}' {
			flush()
			if len(stack) > 0 {
				cur = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}
			i++
			continue
		}
		if c == '&' && i+1 < len(str) {
			if col, ok := colorCode[str[i+1]]; ok {
				flush()
				cur = col
				i++
				continue
			}
		}
		b.WriteByte(c)
	}
	flush()
	return out
}

// Plain strips markup.
func Plain(str string) string {
	var b strings.Builder
	for _, r := range parseMarkup(str, 0) {
		b.WriteString(r.s)
	}
	return b.String()
}

// Wrap splits markup text into lines of at most w visible runes; colour spans left open re-open on the next line.
func Wrap(str string, w int) []string {
	var lines []string
	var open []string
	for _, para := range strings.Split(str, "\n") {
		words := strings.Fields(para)
		line := strings.Join(open, "")
		ln := 0
		for _, wd := range words {
			wl := len([]rune(Plain(wd)))
			if ln > 0 && ln+1+wl > w {
				lines = append(lines, line)
				line, ln = strings.Join(open, ""), 0
			}
			if ln > 0 {
				line += " "
				ln++
			}
			line += wd
			ln += wl
			open = trackOpen(open, wd)
		}
		lines = append(lines, line)
	}
	return lines
}

// trackOpen updates the stack of "{{X|" openers still unclosed after a word.
func trackOpen(open []string, wd string) []string {
	for i := 0; i < len(wd); i++ {
		if wd[i] == '{' && i+3 < len(wd) && wd[i+1] == '{' && wd[i+3] == '|' {
			open = append(open, wd[i:i+4])
			i += 3
		} else if wd[i] == '}' && i+1 < len(wd) && wd[i+1] == '}' && len(open) > 0 {
			open = open[:len(open)-1]
			i++
		}
	}
	return open
}

func (s *Screen) Box(x0, y0, w, h int, f byte, title string) {
	s.Fill(x0, y0, w, h, CBgDark)
	for x := x0; x < x0+w; x++ {
		s.Put(x, y0, '─', f, CBgDark)
		s.Put(x, y0+h-1, '─', f, CBgDark)
	}
	for y := y0; y < y0+h; y++ {
		s.Put(x0, y, '│', f, CBgDark)
		s.Put(x0+w-1, y, '│', f, CBgDark)
	}
	s.Put(x0, y0, '┌', f, CBgDark)
	s.Put(x0+w-1, y0, '┐', f, CBgDark)
	s.Put(x0, y0+h-1, '└', f, CBgDark)
	s.Put(x0+w-1, y0+h-1, '┘', f, CBgDark)
	if title != "" {
		s.TextB(x0+2, y0, "[ "+title+" ]", CYellow, CBgDark, x0+w-1)
	}
}

func Bar(cur, max, w int) (int, int) {
	if max <= 0 {
		return 0, w
	}
	f := cur * w / max
	if f < 0 {
		f = 0
	}
	if cur > 0 && f == 0 {
		f = 1
	}
	if f > w {
		f = w
	}
	return f, w - f
}
