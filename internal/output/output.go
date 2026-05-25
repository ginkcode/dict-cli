package output

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/gink/dict-cli/internal/dict"
	"golang.org/x/term"
)

const (
	typeColWidth    = 12
	grammarColWidth = 14
	defaultMeaningW = 40
	defaultExamplesW = 70
	maxTableWidth   = 132
)

var (
	green  = "\033[32m"
	red    = "\033[31m"
	bold   = "\033[1m"
	reset  = "\033[0m"
	isatty bool
)

func init() {
	fi, _ := os.Stdout.Stat()
	isatty = (fi.Mode() & os.ModeCharDevice) != 0
}

func color(c, s string) string {
	if !isatty {
		return s
	}
	return c + s + reset
}

func boldText(s string) string {
	if !isatty {
		return s
	}
	return bold + s + reset
}

func PrintResults(valid, invalid []dict.Result) {
	if len(valid) > 0 {
		fmt.Printf("\n%s\n\n", color(green, fmt.Sprintf("VALID WORDS (%d)", len(valid))))
		for _, r := range valid {
			printWordEntry(r)
		}
	}

	if len(invalid) > 0 {
		fmt.Printf("\n%s\n\n", color(red, fmt.Sprintf("INVALID WORDS (%d)", len(invalid))))
		printInvalidTable(invalid)
	}
}

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil && w > 0 {
		return w
	}
	return 0
}

func computeColumns(termW int) [4]int {
	w := termW
	if w <= 0 || w > maxTableWidth {
		w = maxTableWidth
	}

	totalFixed := typeColWidth + grammarColWidth + 13
	remaining := w - totalFixed
	if remaining < 20 {
		remaining = 20
	}

	ratio := float64(defaultMeaningW) / float64(defaultMeaningW+defaultExamplesW)
	meaningColW := int(float64(remaining) * ratio)
	examplesColW := remaining - meaningColW

	if meaningColW < 10 {
		meaningColW = 10
	}
	if examplesColW < 10 {
		examplesColW = 10
	}

	return [4]int{typeColWidth, grammarColWidth, meaningColW, examplesColW}
}

func printWordEntry(r dict.Result) {
	if r.Pronunciation != "" {
		fmt.Printf("  %s  %s\n\n", boldText(r.Word), r.Pronunciation)
	} else {
		fmt.Printf("  %s\n\n", boldText(r.Word))
	}

	colW := computeColumns(termWidth())
	headers := []string{"TYPE", "GRAMMAR", "MEANING", "EXAMPLES"}

	fmt.Println(drawTop(colW))
	fmt.Println(drawRow(colW, padCols(colW, headers)))
	fmt.Println(drawSep(colW))

	meaningW := colW[2]
	examplesW := colW[3]

	for _, m := range r.Meanings {
		var examplesCol string
		if len(m.Examples) > 0 {
			numbered := make([]string, len(m.Examples))
			for j, ex := range m.Examples {
				prefix := fmt.Sprintf("%d. ", j+1)
				numbered[j] = prefix + wrapText(ex, examplesW-utf8.RuneCountInString(prefix))
			}
			examplesCol = strings.Join(numbered, "\n")
		}
		meaningCol := wrapText(m.Meaning, meaningW)
		cols := []string{m.Type, m.Grammar, meaningCol, examplesCol}

		for _, line := range expandLines(cols) {
			fmt.Println(drawRow(colW, padCols(colW, line)))
		}
	}

	fmt.Println(drawBot(colW))
	fmt.Println()
}

func expandLines(cells []string) [][]string {
	cellLines := make([][]string, 4)
	maxLines := 0
	for i := range 4 {
		cellLines[i] = strings.Split(cells[i], "\n")
		if len(cellLines[i]) > maxLines {
			maxLines = len(cellLines[i])
		}
	}

	rows := make([][]string, maxLines)
	for l := 0; l < maxLines; l++ {
		row := make([]string, 4)
		for i := range 4 {
			if l < len(cellLines[i]) {
				row[i] = cellLines[i][l]
			}
		}
		rows[l] = row
	}
	return rows
}

func padCols(w [4]int, cells []string) []string {
	padded := make([]string, 4)
	for i, c := range cells {
		if i >= 4 {
			break
		}
		padded[i] = fmt.Sprintf("%-*s", w[i], c)
	}
	return padded
}

func drawTop(w [4]int) string {
	return drawLine(w, '┌', '┬', '┐')
}

func drawSep(w [4]int) string {
	return drawLine(w, '├', '┼', '┤')
}

func drawBot(w [4]int) string {
	return drawLine(w, '└', '┴', '┘')
}

func drawLine(w [4]int, left, mid, right rune) string {
	var b strings.Builder
	b.WriteRune(left)
	for i, width := range w {
		b.WriteString(strings.Repeat("─", width+2))
		if i < len(w)-1 {
			b.WriteRune(mid)
		}
	}
	b.WriteRune(right)
	return b.String()
}

func drawRow(w [4]int, cells []string) string {
	var b strings.Builder
	b.WriteRune('│')
	for i, cell := range cells {
		fmt.Fprintf(&b, " %-*s │", w[i], cell)
	}
	return b.String()
}

func wrapText(s string, maxWidth int) string {
	if utf8.RuneCountInString(s) <= maxWidth {
		return s
	}

	var lines []string
	for utf8.RuneCountInString(s) > maxWidth {
		cut := maxWidth
		for i := cut; i > 0; i-- {
			r, _ := utf8.DecodeLastRuneInString(s[:i])
			if r == ' ' || r == ',' || r == ';' || r == '/' || r == '-' || r == '(' {
				cut = i
				break
			}
		}
		lines = append(lines, strings.TrimSpace(s[:cut]))
		s = strings.TrimSpace(s[cut:])
	}
	if s != "" {
		lines = append(lines, s)
	}
	return strings.Join(lines, "\n")
}

func colorPad(s, c string, w int) string {
	colored := color(c, s)
	// %-*s pads by bytes; ANSI escapes add invisible bytes, so widen the
	// format width by the number of escape bytes to keep visual width == w.
	return fmt.Sprintf("%-*s", w+len(colored)-len(s), colored)
}

func printInvalidTable(invalid []dict.Result) {
	w := 32

	fmt.Printf("┌%s┐\n", strings.Repeat("─", w+2))
	fmt.Printf("│ %s │\n", colorPad("WORD", red, w))
	fmt.Printf("├%s┤\n", strings.Repeat("─", w+2))

	for _, r := range invalid {
		word := r.Word
		if r.Err != nil {
			word = fmt.Sprintf("%s (%s)", r.Word, r.Err.Error())
		}
		fmt.Printf("│ %s │\n", colorPad(word, red, w))
	}

	fmt.Printf("└%s┘\n\n", strings.Repeat("─", w+2))
}
