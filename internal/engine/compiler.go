package engine

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Compiler provides Mustache-like variable interpolation and Markdown-to-HTML rendering.
type Compiler struct{}

func NewCompiler() *Compiler {
	return &Compiler{}
}

// Render evaluates variables, conditionals, and loops within templateStr against data.
func (c *Compiler) Render(templateStr string, data map[string]any, escapeHTML bool) (string, error) {
	if templateStr == "" {
		return "", nil
	}

	result := templateStr

	// 1. Process {{#each list}}...{{/each}} loops
	result = c.renderLoops(result, data, escapeHTML)

	// 2. Process {{#if cond}}...{{else}}...{{/if}} and {{#if cond}}...{{/if}}
	result = c.renderConditionals(result, data)

	// 3. Process {{#unless cond}}...{{/unless}}
	result = c.renderUnless(result, data)

	// 4. Process {{{raw}}} unescaped variables
	result = c.renderRawVariables(result, data)

	// 5. Process {{variable}} standard variables
	result = c.renderVariables(result, data, escapeHTML)

	return result, nil
}

// renderVariables replaces {{path.to.val}}
var varRegex = regexp.MustCompile(`\{\{\s*([@a-zA-Z0-9_\.]+)\s*\}\}`)

func (c *Compiler) renderVariables(tmpl string, data map[string]any, escapeHTML bool) string {
	return varRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		sub := varRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		path := sub[1]
		val, found := resolvePath(data, path)
		if !found || val == nil {
			return ""
		}
		str := fmt.Sprintf("%v", val)
		if escapeHTML {
			return html.EscapeString(str)
		}
		return str
	})
}

// renderRawVariables replaces {{{path.to.val}}}
var rawVarRegex = regexp.MustCompile(`\{\{\{\s*([@a-zA-Z0-9_\.]+)\s*\}\}\}`)

func (c *Compiler) renderRawVariables(tmpl string, data map[string]any) string {
	return rawVarRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		sub := rawVarRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		path := sub[1]
		val, found := resolvePath(data, path)
		if !found || val == nil {
			return ""
		}
		return fmt.Sprintf("%v", val)
	})
}

// renderConditionals processes {{#if cond}}...{{else}}...{{/if}} and {{#if cond}}...{{/if}}
var ifRegex = regexp.MustCompile(`(?s)\{\{#if\s+([a-zA-Z0-9_\.]+)\s*\}\}(.*?)(?:\{\{else\}\}(.*?))?\{\{/if\}\}`)

func (c *Compiler) renderConditionals(tmpl string, data map[string]any) string {
	return ifRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		sub := ifRegex.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		condKey := sub[1]
		trueBranch := sub[2]
		falseBranch := ""
		if len(sub) > 3 {
			falseBranch = sub[3]
		}

		val, found := resolvePath(data, condKey)
		if found && isTruthy(val) {
			return trueBranch
		}
		return falseBranch
	})
}

// renderUnless processes {{#unless cond}}...{{/unless}}
var unlessRegex = regexp.MustCompile(`(?s)\{\{#unless\s+([a-zA-Z0-9_\.]+)\s*\}\}(.*?)\{\{/unless\}\}`)

func (c *Compiler) renderUnless(tmpl string, data map[string]any) string {
	return unlessRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		sub := unlessRegex.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		condKey := sub[1]
		body := sub[2]

		val, found := resolvePath(data, condKey)
		if !found || !isTruthy(val) {
			return body
		}
		return ""
	})
}

// renderLoops processes {{#each list}}...{{/each}}
var loopRegex = regexp.MustCompile(`(?s)\{\{#each\s+([a-zA-Z0-9_\.]+)\s*\}\}(.*?)\{\{/each\}\}`)

func (c *Compiler) renderLoops(tmpl string, data map[string]any, escapeHTML bool) string {
	return loopRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		sub := loopRegex.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		listKey := sub[1]
		blockTmpl := sub[2]

		val, found := resolvePath(data, listKey)
		if !found || val == nil {
			return ""
		}

		slice, ok := toSlice(val)
		if !ok || len(slice) == 0 {
			return ""
		}

		var sb strings.Builder
		for i, item := range slice {
			itemData := make(map[string]any)
			// copy parent data for inherited context
			for k, v := range data {
				itemData[k] = v
			}
			itemData["@index"] = i
			itemData["this"] = item

			if itemMap, ok := item.(map[string]any); ok {
				for k, v := range itemMap {
					itemData[k] = v
				}
			}

			renderedItem, _ := c.Render(blockTmpl, itemData, escapeHTML)
			sb.WriteString(renderedItem)
		}
		return sb.String()
	})
}

func resolvePath(data map[string]any, path string) (any, bool) {
	if data == nil {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}
	return current, true
}

func isTruthy(val any) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "false" && v != "0"
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case []any:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	default:
		return true
	}
}

func toSlice(val any) ([]any, bool) {
	switch v := val.(type) {
	case []any:
		return v, true
	case []string:
		res := make([]any, len(v))
		for i, s := range v {
			res[i] = s
		}
		return res, true
	case []map[string]any:
		res := make([]any, len(v))
		for i, m := range v {
			res[i] = m
		}
		return res, true
	default:
		return nil, false
	}
}

// ==================== MARKDOWN TO EMAIL HTML ====================

// MarkdownToHTML converts a markdown body into clean, responsive HTML with inline CSS.
func MarkdownToHTML(md string) string {
	lines := strings.Split(md, "\n")
	var htmlLines []string
	inList := false

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			if inList {
				htmlLines = append(htmlLines, "</ul>")
				inList = false
			}
			continue
		}

		// Bullet lists: - item or * item
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if !inList {
				htmlLines = append(htmlLines, `<ul style="margin: 0 0 16px 20px; padding: 0; color: #374151; font-size: 15px; line-height: 24px;">`)
				inList = true
			}
			itemContent := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
			htmlLines = append(htmlLines, fmt.Sprintf(`<li style="margin-bottom: 6px;">%s</li>`, formatInline(itemContent)))
			continue
		}

		if inList {
			htmlLines = append(htmlLines, "</ul>")
			inList = false
		}

		// Headings
		if strings.HasPrefix(trimmed, "# ") {
			text := strings.TrimPrefix(trimmed, "# ")
			htmlLines = append(htmlLines, fmt.Sprintf(`<h1 style="margin: 0 0 16px 0; color: #111827; font-size: 24px; font-weight: 700; line-height: 32px;">%s</h1>`, formatInline(text)))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			text := strings.TrimPrefix(trimmed, "## ")
			htmlLines = append(htmlLines, fmt.Sprintf(`<h2 style="margin: 20px 0 12px 0; color: #111827; font-size: 18px; font-weight: 600; line-height: 26px;">%s</h2>`, formatInline(text)))
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			text := strings.TrimPrefix(trimmed, "### ")
			htmlLines = append(htmlLines, fmt.Sprintf(`<h3 style="margin: 16px 0 8px 0; color: #1f2937; font-size: 16px; font-weight: 600; line-height: 24px;">%s</h3>`, formatInline(text)))
			continue
		}

		// Horizontal rule: ---
		if trimmed == "---" || trimmed == "***" {
			htmlLines = append(htmlLines, `<hr style="border: none; border-top: 1px solid #e5e7eb; margin: 24px 0;" />`)
			continue
		}

		// Paragraph
		htmlLines = append(htmlLines, fmt.Sprintf(`<p style="margin: 0 0 16px 0; color: #374151; font-size: 15px; line-height: 24px;">%s</p>`, formatInline(line)))
	}

	if inList {
		htmlLines = append(htmlLines, "</ul>")
	}

	return strings.Join(htmlLines, "\n")
}

var boldRegex = regexp.MustCompile(`\*\*(.*?)\*\*`)
var italicRegex = regexp.MustCompile(`\*(.*?)\*`)
var linkRegex = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
var codeRegex = regexp.MustCompile("`" + `(.*?)` + "`")

func formatInline(text string) string {
	escaped := html.EscapeString(text)

	// Code: `code`
	escaped = codeRegex.ReplaceAllString(escaped, `<code style="background-color: #f3f4f6; color: #111827; padding: 2px 6px; border-radius: 4px; font-family: monospace; font-size: 13px;">$1</code>`)

	// Bold: **text**
	escaped = boldRegex.ReplaceAllString(escaped, `<strong style="color: #111827; font-weight: 600;">$1</strong>`)

	// Italic: *text*
	escaped = italicRegex.ReplaceAllString(escaped, `<em style="color: #4b5563;">$1</em>`)

	// Links: [title](url)
	escaped = linkRegex.ReplaceAllString(escaped, `<a href="$2" style="color: #2563eb; text-decoration: underline; font-weight: 500;">$1</a>`)

	return escaped
}

// WrapEmailHTML wraps body HTML with a clean responsive transactional email container.
func WrapEmailHTML(bodyHTML, title string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
</head>
<body style="margin: 0; padding: 0; background-color: #f9fafb; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">
  <table width="100%%" border="0" cellspacing="0" cellpadding="0" style="background-color: #f9fafb; padding: 32px 16px;">
    <tr>
      <td align="center">
        <table width="100%%" border="0" cellspacing="0" cellpadding="0" style="max-width: 560px; background-color: #ffffff; border-radius: 8px; border: 1px solid #e5e7eb; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.05);">
          <tr>
            <td style="padding: 32px 32px 24px 32px;">
              %s
            </td>
          </tr>
          <tr>
            <td style="padding: 16px 32px; background-color: #f3f4f6; border-top: 1px solid #e5e7eb; text-align: center; color: #9ca3af; font-size: 12px;">
              Sent via Notiflux
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, html.EscapeString(title), bodyHTML)
}
