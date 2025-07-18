package parser

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

func templateFunctionExpand(s string) (string, error) {
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid template: %s. - must join two integers", s)
		}
		nFrom, err := strconv.Atoi(parts[0])
		if err != nil {
			return "", fmt.Errorf("invalid template: %s. First part must be an integer", s)
		}
		nTo, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", fmt.Errorf("invalid template: %s. Second part must be an integer", s)
		}
		if nFrom > nTo {
			nFrom, nTo = nTo, nFrom
		}
		if nFrom < 0 || nTo > 65535 {
			return "", fmt.Errorf("invalid template: %s. Integers must be in range [0, 65535]", s)
		}
		// expand the range to [nFrom, nTo]
		var result []string
		for i := nFrom; i <= nTo; i++ {
			result = append(result, strconv.Itoa(i))
		}
		return strings.Join(result, ","), nil
	} else {
		return "", fmt.Errorf("invalid template: %s. Expected: expand \"[int]-[int]\"", s)
	}
}

var templateFunctionMap = template.FuncMap{
	"expand": templateFunctionExpand,
}

// ApplyTemplate processes the configuration content with template functions
func ApplyTemplate(content []byte) ([]byte, error) {
	tmpl, err := template.New("config").Funcs(templateFunctionMap).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	err = tmpl.Execute(&result, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return []byte(result.String()), nil
}
