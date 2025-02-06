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
)

// httpPostJsonSlice0 使用Post发送Json请求,并把返回值中,指定key的值变成数组,然后取值第一个返回
func httpPostJsonSlice0(client *http.Client, authorization string, url string, header map[string]string, bodyMap map[string]interface{}, resultKey string) (interface{}, error) {
	resp, err := httpPostResponse(client, authorization, url, header, bodyMap)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resultKey == "" {
		return body, nil
	}
	return bodyJsonKeyValue(body, resultKey)
}

// httpPostResponse post请求的response
func httpPostResponse(client *http.Client, authorization string, url string, header map[string]string, bodyMap map[string]interface{}) (*http.Response, error) {
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
	req.Header.Set("Authorization", "Bearer "+authorization)
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
	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, errors.New("http post error")
	}
	return resp, err
}

// bodyJsonKeyValue 从body中json对象中,获取指定的key
func bodyJsonKeyValue(body []byte, key string) (interface{}, error) {

	// 将 JSON 数据解析为 map[string]interface{}
	var resultMap map[string]interface{}
	if err := json.Unmarshal(body, &resultMap); err != nil {
		return nil, err
	}
	if key == "" {
		return resultMap, nil
	}
	resultSlice := resultMap[key]
	if resultSlice == nil {
		return resultSlice, nil
	}
	rs, ok := resultSlice.([]interface{})
	if !ok || len(rs) < 1 {
		return resultSlice, nil
	}
	return rs[0], nil
}
