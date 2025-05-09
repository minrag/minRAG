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
	ctx := context.Background()
	hc.Initialization(ctx, input)
	err := hc.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(document.Markdown)
}

func TestWebScraper(t *testing.T) {
	ws := WebScraper{QuerySelector: []string{"#s-top-left"}}
	ws.Depth = 2
	input := make(map[string]interface{}, 0)
	document := &Document{Markdown: ""}
	input["document"] = document
	ws.WebURL = "https://www.baidu.com"
	ctx := context.Background()
	ws.Initialization(ctx, input)
	herf, err := ws.FetchPage(ctx, document, input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(document)
	fmt.Println("-------------------")
	fmt.Println(herf)

}
