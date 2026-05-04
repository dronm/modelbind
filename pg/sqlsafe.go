package pg

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	identRE          = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	jsonKeyTokenRE   = regexp.MustCompile(`^->>?'[A-Za-z0-9_:-]+'`)
	jsonIndexTokenRE = regexp.MustCompile(`^->>?[0-9]+`)
)

func SanitizeSQLFieldRef(field string) (string, error) {
	return sanitizeSQLFieldRef(field)
}

func sanitizeSQLFieldRef(field string) (string, error) {
	field = strings.TrimSpace(field)
	if field == "" {
		return "", fmt.Errorf("empty field reference")
	}

	parts := strings.Split(field, ".")
	for i, part := range parts {
		if part == "" {
			return "", fmt.Errorf("invalid field reference %q", field)
		}
		safePart, err := sanitizeFieldSegment(part)
		if err != nil {
			return "", err
		}
		parts[i] = safePart
	}

	return strings.Join(parts, "."), nil
}

func sanitizeFieldSegment(segment string) (string, error) {
	base := segment
	jsonExpr := ""
	if idx := strings.Index(segment, "->"); idx >= 0 {
		base = segment[:idx]
		jsonExpr = segment[idx:]
	}

	if !identRE.MatchString(base) {
		return "", fmt.Errorf("invalid field reference %q", segment)
	}

	for jsonExpr != "" {
		if match := jsonKeyTokenRE.FindString(jsonExpr); match != "" {
			jsonExpr = jsonExpr[len(match):]
			continue
		}
		if match := jsonIndexTokenRE.FindString(jsonExpr); match != "" {
			jsonExpr = jsonExpr[len(match):]
			continue
		}
		return "", fmt.Errorf("invalid json path in field reference %q", segment)
	}

	return segment, nil
}

func joinSafeFieldRefs(fieldIDs []string) string {
	parts := make([]string, 0, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		safeFieldID, err := sanitizeSQLFieldRef(fieldID)
		if err != nil {
			panic(err)
		}
		parts = append(parts, safeFieldID)
	}
	return strings.Join(parts, ",")
}
