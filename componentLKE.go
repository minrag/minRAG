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

//POST https://lkeap.tencentcloudapi.com/
//Authorization: TC3-HMAC-SHA256 Credential=AKID********************************/2019-02-25/lkeap/tc3_request, SignedHeaders=content-type;host;x-tc-action, Signature=10b1a37a7301a02ca19a647ad722d5e43b4b3cff309d421d85b46093f6ab6c4f
/**
Content-Type: application/json; charset=utf-8
Host: lkeap.tencentcloudapi.com
X-TC-Action: GetEmbedding
X-TC-Version: 2024-05-22
X-TC-Timestamp: 1551113065
X-TC-Region: ap-guangzhou

{"Limit": 1, "Filters": [{"Values": ["\u672a\u547d\u540d"], "Name": "instance-name"}]}

*/
// https://cloud.tencent.com/document/product/1772/115368

var lkeapHost = "lkeap.tencentcloudapi.com"
var lkeAlgorithm = "TC3-HMAC-SHA256"

// LKETextEmbedder  LKE向量化字符串文本
type LKETextEmbedder struct {
	Host      string `json:"Host,omitempty"`        // lkeap.tencentcloudapi.com
	Action    string `json:"X-TC-Action,omitempty"` // GetEmbedding
	Region    string `json:"X-TC-Region,omitempty"` // ap-guangzhou
	Timestamp int    `json:"X-TC-Timestamp,omitempty"`
	Version   string `json:"X-TC-Version,omitempty"` // 2024-05-22

	Model          string            `json:"Model,omitempty"` // lke-text-embedding-v1
	BaseURL        string            `json:"base_url,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	client         *http.Client      `json:"-"`
}

func (component *LKETextEmbedder) Initialization(ctx context.Context, input map[string]interface{}) error {
	component.Host = "lkeap.tencentcloudapi.com"
	component.Region = "ap-guangzhou"
	component.Version = "2024-05-22"

	if component.Timeout == 0 {
		component.Timeout = 180
	}
	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
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
