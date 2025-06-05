// Copyright (c) 2025 minRAG Authors.
//
// This file is part of minRAG.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses>.

// MCP组件库
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ToolDefinition 表示 MCP Server 提供的工具定义
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"` // 参数模式 (JSON Schema)
}

// MCPClient 实现 MCP Streamable HTTP 协议客户端
type MCPClient struct {
	BaseURL    string
	HTTPClient *http.Client
	SessionID  string           // 会话状态保持
	Tools      []ToolDefinition // 缓存的工具列表
}

func NewMCPClient(baseURL string) *MCPClient {
	return &MCPClient{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				DisableCompression: true,
			},
		},
	}
}

// DiscoverTools 发现 MCP Server 提供的工具列表和参数
func (c *MCPClient) DiscoverTools(ctx context.Context) error {
	// 调用标准的 list_tools 接口获取工具信息
	respData, err := c.callRPC(ctx, "GET", "list_tools", nil)
	if err != nil {
		return err
	}

	// 解析工具定义
	var tools []ToolDefinition
	if err := json.Unmarshal(respData, &tools); err != nil {
		return fmt.Errorf("解析工具列表失败: %w", err)
	}

	c.Tools = tools
	return nil
}

// GetTools 返回缓存的工具列表
func (c *MCPClient) GetTools() []ToolDefinition {
	return c.Tools
}

// Call 发送请求并处理响应
func (c *MCPClient) Call(
	ctx context.Context,
	method, toolName string,
	params interface{},
	callback func(data []byte) error,
) error {
	// 1. 构建 MCP 协议请求体
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  toolName,
		"params":  params,
		"id":      time.Now().UnixNano(),
	}

	return c.callRPCWithCallback(ctx, method, toolName, reqBody, callback)
}

// callRPC 调用 RPC 方法并返回完整响应
func (c *MCPClient) callRPC(
	ctx context.Context,
	method, procedure string,
	params interface{},
) ([]byte, error) {
	var respData []byte
	err := c.callRPCWithCallback(ctx, method, procedure, params, func(data []byte) error {
		respData = append(respData, data...)
		return nil
	})
	return respData, err
}

// callRPCWithCallback 核心 RPC 调用方法
func (c *MCPClient) callRPCWithCallback(
	ctx context.Context,
	method, procedure string,
	params interface{},
	callback func(data []byte) error,
) error {
	jsonBody, _ := json.Marshal(params)

	// 2. 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+"/mcp", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}

	// 3. 设置 MCP 协议头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if c.SessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.SessionID)
	}

	// 4. 发送请求
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 5. 保存会话ID
	if sessionID := resp.Header.Get("Mcp-Session-Id"); sessionID != "" {
		c.SessionID = sessionID
	}

	// 6. 根据内容类型选择处理模式
	contentType := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "text/event-stream"):
		return c.handleSSEStream(resp.Body, callback)
	case strings.Contains(resp.Header.Get("Transfer-Encoding"), "chunked"):
		return c.handleChunkedStream(resp.Body, callback)
	default:
		return c.handleHTTP(resp, callback)
	}
}

// handleSSEStream 处理Server-Sent Events流
func (c *MCPClient) handleSSEStream(body io.ReadCloser, callback func([]byte) error) error {
	defer body.Close()
	reader := bufio.NewReader(body)

	var eventData bytes.Buffer
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		line = bytes.TrimSpace(line)

		if len(line) == 0 {
			if eventData.Len() > 0 {
				if err := callback(eventData.Bytes()); err != nil {
					return err
				}
				eventData.Reset()
			}
			continue
		}

		switch {
		case bytes.HasPrefix(line, []byte("data:")):
			data := bytes.TrimSpace(line[5:])
			if eventData.Len() > 0 {
				eventData.WriteByte('\n')
			}
			eventData.Write(data)
		}
	}

	if eventData.Len() > 0 {
		return callback(eventData.Bytes())
	}

	return nil
}

// handleChunkedStream 处理分块传输编码
func (c *MCPClient) handleChunkedStream(body io.ReadCloser, callback func([]byte) error) error {
	defer body.Close()
	reader := bufio.NewReader(body)

	for {
		chunkSizeLine, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		chunkSizeLine = strings.TrimSpace(chunkSizeLine)
		if chunkSizeLine == "0" {
			break
		}

		var chunkSize int
		_, err = fmt.Sscanf(chunkSizeLine, "%x", &chunkSize)
		if err != nil {
			return err
		}

		data := make([]byte, chunkSize)
		if _, err := io.ReadFull(reader, data); err != nil {
			return err
		}

		if err := callback(data); err != nil {
			return err
		}

		if _, err := reader.Discard(2); err != nil {
			return err
		}
	}
	return nil
}

// handleHTTP 处理标准 HTTP 响应
func (c *MCPClient) handleHTTP(resp *http.Response, callback func([]byte) error) error {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return callback(data)
}
