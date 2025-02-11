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
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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

// LKETextEmbedder  LKE向量化字符串文本
type LKETextEmbedder struct {
	Host      string `json:"Host,omitempty"`        // lkeap.tencentcloudapi.com
	Action    string `json:"X-TC-Action,omitempty"` // GetEmbedding
	Region    string `json:"X-TC-Region,omitempty"` // ap-guangzhou
	Timestamp int    `json:"X-TC-Timestamp,omitempty"`
	Version   string `json:"X-TC-Version,omitempty"` // 2024-05-22

	Algorithm string `json:"Algorithm"` // TC3-HMAC-SHA256

	SecretId  string `json:"SecretId"`
	SecretKey string `json:"SecretKey"`

	Model          string            `json:"Model,omitempty"` // lke-text-embedding-v1
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	client         *http.Client      `json:"-"`
}

func (component *LKETextEmbedder) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Host == "" {
		component.Host = "lkeap.tencentcloudapi.com"
	}
	if component.Action == "" {
		component.Action = "GetEmbedding"
	}
	if component.Region == "" {
		component.Region = "ap-guangzhou"
	}
	if component.Version == "" {
		component.Version = "2024-05-22"
	}

	if component.Model == "" {
		component.Model = "lke-text-embedding-v1"
	}

	if component.Algorithm == "" {
		component.Algorithm = "TC3-HMAC-SHA256"
	}

	if component.DefaultHeaders == nil {
		component.DefaultHeaders = make(map[string]string, 0)
	}

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
	bodyByte, err := httpPostJsonBody(component.client, "Authorization", "/embeddings", component.DefaultHeaders, bodyMap)
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

func genAuthorization(ctx context.Context, secretId, secretKey, host, algorithm, service, version, action, region string, bodyMap map[string]interface{}) ([]byte, error) {
	// 需要设置环境变量 TENCENTCLOUD_SECRET_ID，值为示例的 AKIDz8krbsJ5yKBZQpn74WFkmLPx3*******
	var timestamp int64 = time.Now().Unix()
	// step 1: build canonical request string
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		"application/json; charset=utf-8", host, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"
	payload := `{"Limit": 1, "Filters": [{"Values": ["\u672a\u547d\u540d"], "Name": "instance-name"}]}`
	hashedRequestPayload := sha256hex(payload)
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)
	fmt.Println(canonicalRequest)

	// step 2: build string to sign
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := sha256hex(canonicalRequest)
	string2sign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest)
	fmt.Println(string2sign)

	// step 3: sign string
	secretDate := hmacsha256(date, "TC3"+secretKey)
	secretService := hmacsha256(service, secretDate)
	secretSigning := hmacsha256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(hmacsha256(string2sign, secretSigning)))
	fmt.Println(signature)

	// step 4: build authorization
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		secretId,
		credentialScope,
		signedHeaders,
		signature)
	fmt.Println(authorization)
	// 序列化请求体
	payloadBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}
	// 创建HTTP请求
	req, err := http.NewRequest(httpRequestMethod, "https://"+host, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Region", region)

	curl := fmt.Sprintf(`curl -X POST https://%s\
	-H "Authorization: %s"\
	-H "Content-Type: application/json; charset=utf-8"\
	-H "Host: %s" -H "X-TC-Action: %s"\
	-H "X-TC-Timestamp: %d"\
	-H "X-TC-Version: %s"\
	-H "X-TC-Region: %s"\
	-d '%s'`, host, authorization, host, action, timestamp, version, region, payload)
	fmt.Println(curl)

	return nil, nil
}

func sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func hmacsha256(s, key string) string {
	hashed := hmac.New(sha256.New, []byte(key))
	hashed.Write([]byte(s))
	return string(hashed.Sum(nil))
}
