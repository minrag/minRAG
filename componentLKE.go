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

// 腾讯云大模型知识引擎LKE https://cloud.tencent.com/product/lke ,适配Embedding和Reranker模型
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// https://cloud.tencent.com/document/product/1772/115368

// LKETextEmbedder  LKE向量化字符串文本
type LKETextEmbedder struct {
	Action         string            `json:"Action,omitempty"`
	Region         string            `json:"Region,omitempty"`
	Model          string            `json:"Model,omitempty"`
	Version        string            `json:"Version,omitempty"`
	Timestamp      int               `json:"-"`
	BaseURL        string            `json:"base_url,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	client         *http.Client      `json:"-"`
}

func (component *LKETextEmbedder) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 180
	}
	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
	}
	if component.BaseURL == "" {
		component.BaseURL = config.AIBaseURL
	}

	return nil
}
func (component *LKETextEmbedder) Run(ctx context.Context, input map[string]interface{}) error {
	queryObj, has := input["query"]
	if !has {
		return errors.New(funcT("input['query'] cannot be empty"))
	}
	bodyMap := make(map[string]interface{}, 0)
	bodyMap["input"] = queryObj.(string)
	bodyMap["model"] = component.Model
	bodyMap["encoding_format"] = "float"
	bodyByte, err := httpPostJsonBody(component.client, "Authorization", component.BaseURL+"/embeddings", component.DefaultHeaders, bodyMap)
	if err != nil {
		input[errorKey] = err
		return err
	}
	rs := struct {
		Data []struct {
			Embedding []float64 `json:"embedding,omitempty"`
		} `json:"data,omitempty"`
	}{}
	err = json.Unmarshal(bodyByte, &rs)
	if err != nil {
		input[errorKey] = err
		return err
	}
	if len(rs.Data) < 1 {
		err := errors.New("httpPostJsonBody data is empty")
		input[errorKey] = err
		return err
	}
	input["embedding"] = rs.Data[0].Embedding
	return nil
}
