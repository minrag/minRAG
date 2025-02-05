// Copyright (c) 2025 minrag Authors.
//
// This file is part of minrag.
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
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// httpPostJsonSlice0 使用Post发送Json请求,并把返回值中,指定key的值变成数组,然后取值第一个返回
func httpPostJsonSlice0(client *http.Client, authorization string, url string, header map[string]string, bodyMap map[string]interface{}, resultKey string) (interface{}, error) {
	if client == nil {
		client = &http.Client{
			Timeout: time.Second * time.Duration(60),
		}
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
	req.Header.Set("Authorization", "Bearer "+authorization)
	req.Header.Set("Content-Type", "application/json")
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
	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("http post error")
	}

	// 读取响应体内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 将 JSON 数据解析为 map[string]interface{}
	var resultMap map[string]interface{}
	if err := json.Unmarshal(body, &resultMap); err != nil {
		return nil, err
	}
	if resultKey == "" {
		return resultMap, nil
	}
	resultSlice := resultMap[resultKey]
	if resultSlice == nil {
		return resultSlice, nil
	}
	rs, ok := resultSlice.([]interface{})
	if !ok || len(rs) < 1 {
		return resultSlice, nil
	}
	return rs[0], nil
}
