package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMCP(t *testing.T) {
	client := NewMCPClient("http://localhost:8080")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 分块流式调用示例 (如AI文本生成)
	fmt.Println("=== 分块流式调用 ===")
	err := client.Call(ctx, "POST", "text_generation", map[string]any{
		"prompt": "Go语言如何实现流式HTTP?",
	}, func(data []byte) error {
		fmt.Printf("收到数据块: %s\n", data)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	// 标准HTTP调用示例
	fmt.Println("\n=== 标准HTTP调用 ===")
	err = client.Call(ctx, "POST", "sql_query", map[string]any{
		"query": "SELECT * FROM users",
	}, func(data []byte) error {
		fmt.Printf("收到完整响应: %s\n", data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
