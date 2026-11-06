package engine

import (
	"strings"
	"testing"
)

func TestCompiler_RenderSimpleVariables(t *testing.T) {
	c := NewCompiler()
	data := map[string]any{
		"user": map[string]any{
			"name": "Alexandr",
			"age":  28,
		},
		"order_id": "ORD-123",
	}

	tmpl := "Hello {{user.name}}! Your order {{order_id}} is confirmed."
	res, err := c.Render(tmpl, data, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello Alexandr! Your order ORD-123 is confirmed."
	if res != expected {
		t.Errorf("expected %q, got %q", expected, res)
	}
}

func TestCompiler_RenderConditionals(t *testing.T) {
	c := NewCompiler()
	tmpl := "{{#if is_premium}}Welcome VIP!{{else}}Welcome Standard!{{/if}}"

	// True branch
	resTrue, _ := c.Render(tmpl, map[string]any{"is_premium": true}, false)
	if resTrue != "Welcome VIP!" {
		t.Errorf("expected true branch, got %q", resTrue)
	}

	// False branch
	resFalse, _ := c.Render(tmpl, map[string]any{"is_premium": false}, false)
	if resFalse != "Welcome Standard!" {
		t.Errorf("expected false branch, got %q", resFalse)
	}

	// Missing branch
	resMissing, _ := c.Render(tmpl, map[string]any{}, false)
	if resMissing != "Welcome Standard!" {
		t.Errorf("expected false branch for missing key, got %q", resMissing)
	}
}

func TestCompiler_RenderUnless(t *testing.T) {
	c := NewCompiler()
	tmpl := "{{#unless muted}}Chime!{{/unless}}"

	resNotMuted, _ := c.Render(tmpl, map[string]any{"muted": false}, false)
	if resNotMuted != "Chime!" {
		t.Errorf("expected body, got %q", resNotMuted)
	}

	resMuted, _ := c.Render(tmpl, map[string]any{"muted": true}, false)
	if resMuted != "" {
		t.Errorf("expected empty, got %q", resMuted)
	}
}

func TestCompiler_RenderLoops(t *testing.T) {
	c := NewCompiler()
	data := map[string]any{
		"items": []any{
			map[string]any{"name": "Apple", "price": 1.5},
			map[string]any{"name": "Banana", "price": 0.8},
		},
	}

	tmpl := "Items:{{#each items}} [{{@index}}: {{name}} ${{price}}]{{/each}}"
	res, err := c.Render(tmpl, data, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Items: [0: Apple $1.5] [1: Banana $0.8]"
	if res != expected {
		t.Errorf("expected %q, got %q", expected, res)
	}
}

func TestCompiler_HTMLEscaping(t *testing.T) {
	c := NewCompiler()
	data := map[string]any{
		"script": "<script>alert(1)</script>",
	}

	// Escaped
	resEscaped, _ := c.Render("Clean: {{script}}", data, true)
	if strings.Contains(resEscaped, "<script>") {
		t.Errorf("expected escaped HTML, got %q", resEscaped)
	}

	// Raw {{{raw}}}
	resRaw, _ := c.Render("Raw: {{{script}}}", data, true)
	if !strings.Contains(resRaw, "<script>alert(1)</script>") {
		t.Errorf("expected unescaped raw HTML, got %q", resRaw)
	}
}

func TestMarkdownToHTML(t *testing.T) {
	md := `# Order Confirmation
Thank you for your order **#8942**.

- Item 1: Keyboard
- Item 2: Mouse

[Track Order](https://example.com/track)`

	htmlRes := MarkdownToHTML(md)
	if !strings.Contains(htmlRes, "<h1") || !strings.Contains(htmlRes, "Order Confirmation") {
		t.Errorf("expected h1 tag, got %s", htmlRes)
	}
	if !strings.Contains(htmlRes, "<strong") || !strings.Contains(htmlRes, "#8942") {
		t.Errorf("expected strong tag, got %s", htmlRes)
	}
	if !strings.Contains(htmlRes, "<ul") || !strings.Contains(htmlRes, "<li") {
		t.Errorf("expected list tags, got %s", htmlRes)
	}
	if !strings.Contains(htmlRes, `<a href="https://example.com/track"`) {
		t.Errorf("expected link tag, got %s", htmlRes)
	}

	wrapped := WrapEmailHTML(htmlRes, "Order Confirmation")
	if !strings.Contains(wrapped, "<!DOCTYPE html>") || !strings.Contains(wrapped, "Sent via Notiflux") {
		t.Errorf("expected wrapped email layout, got %s", wrapped)
	}
}
