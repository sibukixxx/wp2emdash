package wpcli

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rokubunnoni-inc/wp2emdash/internal/walk"
)

// dirSizeAndCount returns the total byte size and file count under root.
// Symlinks are not followed; permission errors are ignored silently because
// audit data is intentionally best-effort.
func dirSizeAndCount(root string) (int64, int) {
	var size int64
	var count int
	_ = walk.Files(root, func(_ string, d fs.DirEntry) error {
		info, err := d.Info()
		if err != nil {
			return nil
		}
		size += info.Size()
		count++
		return nil
	})
	return size, count
}

func countFilesByExt(root, ext string) int {
	count := 0
	_ = walk.Files(root, func(_ string, d fs.DirEntry) error {
		if strings.EqualFold(filepath.Ext(d.Name()), ext) {
			count++
		}
		return nil
	})
	return count
}

// grepCount counts the total occurrences of any of the given needles across
// regular files under root. It scans line-by-line so a single line containing
// multiple needles is counted once per matching needle (mimicking grep -R).
func grepCount(root string, needles ...string) int {
	if len(needles) == 0 {
		return 0
	}
	total := 0
	_ = walk.Files(root, func(path string, _ fs.DirEntry) error {
		if !looksLikeText(path) {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer func() { _ = f.Close() }()
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 64*1024), 1024*1024)
		for s.Scan() {
			line := s.Text()
			for _, n := range needles {
				if strings.Contains(line, n) {
					total++
				}
			}
		}
		return nil
	})
	return total
}

func grepCountInRoots(roots []string, needles ...string) int {
	total := 0
	for _, r := range roots {
		if _, err := os.Stat(r); err != nil {
			continue
		}
		total += grepCount(r, needles...)
	}
	return total
}

func countLinesMatching(path string, needles []string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer func() {
		_ = f.Close()
	}()
	count := 0
	s := bufio.NewScanner(f)
	for s.Scan() {
		lc := strings.ToLower(s.Text())
		for _, n := range needles {
			if strings.Contains(lc, n) {
				count++
				break
			}
		}
	}
	return count
}

// looksLikeText filters obvious binary files out of grep walks. Anything
// without a known text extension is skipped.
func looksLikeText(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".php", ".html", ".htm", ".js", ".jsx", ".ts", ".tsx", ".css", ".scss",
		".json", ".yml", ".yaml", ".md", ".txt", ".xml", ".sh", ".env":
		return true
	case "":
		// .htaccess etc. are no-extension but text — accept.
		return true
	}
	return false
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return formatSize(float64(b), "B")
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"K", "M", "G", "T"}
	return formatSize(float64(b)/float64(div), suffixes[exp]+"B")
}

func formatSize(v float64, suffix string) string {
	switch {
	case v >= 100:
		return strings.TrimRight(strings.TrimRight(formatFloat(v, 0), "0"), ".") + suffix
	case v >= 10:
		return strings.TrimRight(strings.TrimRight(formatFloat(v, 1), "0"), ".") + suffix
	default:
		return strings.TrimRight(strings.TrimRight(formatFloat(v, 2), "0"), ".") + suffix
	}
}

func formatFloat(v float64, prec int) string {
	// Go's strconv.FormatFloat is heavyweight to import indirectly; just inline.
	// %f with precision is enough since human sizes don't need exponent form.
	switch prec {
	case 0:
		return trimDecimal(formatF(v, 0))
	case 1:
		return formatF(v, 1)
	default:
		return formatF(v, 2)
	}
}

func trimDecimal(s string) string {
	if i := strings.Index(s, "."); i >= 0 {
		return s[:i]
	}
	return s
}

// formatF is a tiny sprintf helper that doesn't pull in fmt's reflection path
// for hot loops. Currently we only call it on a few values per audit so it's
// fine to use fmt.Sprintf via the helper file.
func formatF(v float64, prec int) string {
	return sprintfFloat(v, prec)
}
