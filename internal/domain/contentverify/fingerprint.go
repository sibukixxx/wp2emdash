package contentverify

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var (
	tagRE     = regexp.MustCompile(`(?is)<[^>]+>`)
	headingRE = regexp.MustCompile(`(?is)<h([1-6])[^>]*>(.*?)</h[1-6]>`)
	linkRE    = regexp.MustCompile(`(?is)<a[^>]+href\s*=\s*["']([^"']+)["']`)
	imageRE   = regexp.MustCompile(`(?is)<img[^>]+src\s*=\s*["']([^"']+)["']`)
	spaceRE   = regexp.MustCompile(`\s+`)
)

func HashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func FingerprintHTML(raw string) Fingerprint {
	f := Fingerprint{Format: "html", RawSHA256: HashString(raw)}
	for _, m := range headingRE.FindAllStringSubmatch(raw, -1) {
		level := int(m[1][0] - '0')
		f.Headings = append(f.Headings, Heading{Level: level, Hash: HashString(visibleText(m[2]))})
	}
	for _, m := range linkRE.FindAllStringSubmatch(raw, -1) {
		f.Links = append(f.Links, normalizeURL(m[1]))
	}
	for _, m := range imageRE.FindAllStringSubmatch(raw, -1) {
		f.Images = append(f.Images, normalizeMedia(m[1]))
		f.ImageCount++
	}
	lower := strings.ToLower(raw)
	f.Lists = strings.Count(lower, "<ul") + strings.Count(lower, "<ol")
	f.Quotes = strings.Count(lower, "<blockquote")
	f.CodeBlocks = strings.Count(lower, "<pre")
	f.Embeds = strings.Count(lower, "<iframe")
	text := visibleText(raw)
	f.TextSHA256, f.TextLength = HashString(text), len([]rune(text))
	sort.Strings(f.Links)
	sort.Strings(f.Images)
	return f
}

// FingerprintPortableText accepts the raw JSON returned by `emdash content
// get --raw`. It walks the representation without depending on EmDash types.
func FingerprintPortableText(v any) Fingerprint {
	raw, _ := json.Marshal(v)
	f := Fingerprint{Format: "portable_text", RawSHA256: HashString(string(raw))}
	var texts []string
	var walk func(any)
	walk = func(node any) {
		switch x := node.(type) {
		case []any:
			for _, item := range x {
				walk(item)
			}
		case map[string]any:
			typ, _ := x["_type"].(string)
			style, _ := x["style"].(string)
			if typ == "block" {
				var block []string
				if children, ok := x["children"].([]any); ok {
					for _, child := range children {
						if c, ok := child.(map[string]any); ok {
							if s, ok := c["text"].(string); ok {
								block = append(block, s)
							}
						}
					}
				}
				text := normalizeText(strings.Join(block, ""))
				texts = append(texts, text)
				if len(style) == 2 && style[0] == 'h' && style[1] >= '1' && style[1] <= '6' {
					f.Headings = append(f.Headings, Heading{Level: int(style[1] - '0'), Hash: HashString(text)})
				}
				if _, ok := x["listItem"]; ok {
					f.Lists++
				}
				if style == "blockquote" {
					f.Quotes++
				}
			} else {
				switch typ {
				case "image":
					f.ImageCount++
					if ref := nestedStringValue(x, "url", "src", "originalUrl", "asset"); ref != "" {
						f.Images = append(f.Images, normalizeMedia(ref))
					}
				case "embed":
					f.Embeds++
					f.Links = append(f.Links, normalizeURL(stringValue(x, "url")))
				case "code":
					f.CodeBlocks++
				case "", "span", "link":
				default:
					f.UnknownBlocks++
				}
			}
			for key, value := range x {
				if key == "children" {
					continue
				}
				if key == "href" {
					if s, ok := value.(string); ok {
						f.Links = append(f.Links, normalizeURL(s))
					}
				}
				walk(value)
			}
		}
	}
	walk(v)
	text := normalizeText(strings.Join(texts, "\n"))
	f.TextSHA256, f.TextLength = HashString(text), len([]rune(text))
	sort.Strings(f.Links)
	sort.Strings(f.Images)
	return f
}

func visibleText(raw string) string {
	replaced := regexp.MustCompile(`(?is)</?(p|div|li|h[1-6]|blockquote|br)[^>]*>`).ReplaceAllString(raw, "\n")
	return normalizeText(html.UnescapeString(tagRE.ReplaceAllString(replaced, "")))
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, s)
	return strings.TrimSpace(spaceRE.ReplaceAllString(s, " "))
}

func normalizeURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}
	u.Fragment = ""
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if u.Path != "/" {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}
	return u.String()
}

func normalizeMedia(raw string) string {
	raw = normalizeURL(raw)
	if i := strings.Index(raw, "/wp-content/uploads/"); i >= 0 {
		return raw[i+len("/wp-content/uploads/"):]
	}
	return raw
}

func stringValue(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s, ok := m[key].(string); ok {
			return s
		}
	}
	return ""
}

func nestedStringValue(m map[string]any, keys ...string) string {
	if value := stringValue(m, keys...); value != "" {
		return value
	}
	for _, value := range m {
		if child, ok := value.(map[string]any); ok {
			if found := nestedStringValue(child, keys...); found != "" {
				return found
			}
		}
	}
	return ""
}
