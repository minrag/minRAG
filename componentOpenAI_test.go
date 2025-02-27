package main

import (
	"context"
	"fmt"
	"testing"
)

func TestHtmlCleaner(t *testing.T) {
	mk := `
<html><body><p>这是一个示例文本。</p><a href="https://minrag.com">链接</a></body></html>
`
	hc := HtmlCleaner{}
	input := make(map[string]interface{}, 0)
	document := &Document{Markdown: mk}
	input["document"] = document
	err := hc.Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(document.Markdown)
}
