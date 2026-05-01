package seo

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ParseHtaccessRedirects parses an Apache .htaccess body and returns the
// redirect rules it can recognise:
//
//   - Redirect [code|permanent|temp] FROM TO
//   - RedirectMatch [code] REGEX TO
//   - RewriteRule REGEX TARGET [...,R=CODE,...] (only when the R flag is set)
//
// Other directives are ignored. Malformed lines are skipped silently because
// .htaccess files commonly mix in custom or vendor-specific directives that
// the audit tool has no business rejecting.
func ParseHtaccessRedirects(r io.Reader) []RedirectRule {
	var rules []RedirectRule
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		fields := strings.Fields(raw)
		if len(fields) == 0 {
			continue
		}
		directive := strings.ToLower(fields[0])
		var rule (*RedirectRule)
		switch directive {
		case "redirect":
			rule = parseRedirect(fields[1:])
		case "redirectmatch":
			rule = parseRedirectMatch(fields[1:])
		case "rewriterule":
			rule = parseRewriteRule(fields[1:])
		default:
			continue
		}
		if rule == nil {
			continue
		}
		rule.Source = "htaccess"
		rule.Note = fmt.Sprintf("line %d", line)
		rules = append(rules, *rule)
	}
	return rules
}

// parseRedirect handles "Redirect [code|status-keyword] FROM TO".
//
// Default status when omitted is 302 (Apache's behavior for "Redirect").
func parseRedirect(args []string) *RedirectRule {
	code := 302
	switch {
	case len(args) >= 3:
		if c, ok := parseStatusToken(args[0]); ok {
			code = c
			args = args[1:]
		}
	case len(args) < 2:
		return nil
	}
	if len(args) < 2 {
		return nil
	}
	return &RedirectRule{From: args[0], To: args[1], Code: code, Match: "exact"}
}

// parseRedirectMatch handles "RedirectMatch [code] REGEX TO".
func parseRedirectMatch(args []string) *RedirectRule {
	code := 302
	if len(args) >= 3 {
		if c, ok := parseStatusToken(args[0]); ok {
			code = c
			args = args[1:]
		}
	}
	if len(args) < 2 {
		return nil
	}
	return &RedirectRule{From: args[0], To: args[1], Code: code, Match: "regex"}
}

// parseRewriteRule handles "RewriteRule REGEX TARGET [flags]" but only emits
// a rule when the flag list contains "R" or "R=NNN". Pure rewrites without
// redirect intent are not redirects and are skipped.
func parseRewriteRule(args []string) *RedirectRule {
	if len(args) < 2 {
		return nil
	}
	from := args[0]
	to := args[1]
	flags := ""
	if len(args) >= 3 && strings.HasPrefix(args[2], "[") && strings.HasSuffix(args[2], "]") {
		flags = strings.Trim(args[2], "[]")
	}
	if flags == "" {
		return nil
	}
	code, isRedirect := parseRewriteFlags(flags)
	if !isRedirect {
		return nil
	}
	return &RedirectRule{From: from, To: to, Code: code, Match: "regex"}
}

// parseStatusToken interprets numeric or keyword status hints used by Apache.
func parseStatusToken(tok string) (int, bool) {
	switch strings.ToLower(tok) {
	case "permanent":
		return 301, true
	case "temp":
		return 302, true
	case "seeother":
		return 303, true
	case "gone":
		return 410, true
	}
	if n, err := strconv.Atoi(tok); err == nil && n >= 300 && n < 400 {
		return n, true
	}
	return 0, false
}

// parseRewriteFlags returns (code, isRedirect). isRedirect is true only when
// the R flag is present. R defaults to 302 if no value is given.
func parseRewriteFlags(flags string) (int, bool) {
	for _, f := range strings.Split(flags, ",") {
		f = strings.TrimSpace(f)
		if f == "R" {
			return 302, true
		}
		if strings.HasPrefix(strings.ToUpper(f), "R=") {
			if n, err := strconv.Atoi(f[2:]); err == nil {
				return n, true
			}
			return 302, true
		}
	}
	return 0, false
}
