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

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

// httpPostJsonBody 使用Post发送Json请求
func httpPostJsonBody(client *http.Client, authorization string, url string, header map[string]string, bodyMap map[string]interface{}) ([]byte, error) {
	resp, err := httpPostJsonResponse(client, authorization, url, header, bodyMap)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(body) < 1 {
		return nil, errors.New("body is empty")
	}

	return body, nil
}

// httpPostJsonResponse post请求的response
func httpPostJsonResponse(client *http.Client, authorization string, url string, header map[string]string, bodyMap map[string]interface{}) (*http.Response, error) {
	if client == nil {
		return nil, errors.New("httpClient is nil")
	}
	// 序列化请求体
	payloadBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	if authorization != "" {
		req.Header.Set("Authorization", "Bearer "+authorization)
	}
	req.Header.Set("Content-Type", "application/json")
	if len(header) > 0 {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}
	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		bodyByte, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, errors.New(string(bodyByte))
	}
	return resp, err
}

// httpUploadFile http上传附件
func httpUploadFile(client *http.Client, method string, url string, filePath string, header map[string]string) ([]byte, error) {
	if client == nil {
		return nil, errors.New("httpClient is nil")
	}
	if filePath == "" {
		return nil, errors.New("filePath is empty")
	}
	if method == "" {
		method = "POST"
	}
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 创建请求体
	bodyBuffer := &bytes.Buffer{}
	if _, err := io.Copy(bodyBuffer, file); err != nil {
		return nil, fmt.Errorf("read file error: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest(method, url, bodyBuffer)
	if err != nil {
		return nil, err
	}

	if len(header) > 0 {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}
	// 读取响应体内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(body) < 1 {
		return nil, errors.New("body is empty")
	}

	return body, nil
}
