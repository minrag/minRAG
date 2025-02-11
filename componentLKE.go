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
// 使用OpenAI SDK 接入方式: https://console.cloud.tencent.com/lkeap
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
	"io"
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

// LKEDocumentEmbedder  LKE向量化文档字符串
type LKEDocumentEmbedder struct {
	Host   string `json:"Host,omitempty"`   // lkeap.tencentcloudapi.com
	Action string `json:"Action,omitempty"` // GetEmbedding
	Region string `json:"Region,omitempty"` // ap-guangzhou

	Version   string `json:"Version,omitempty"`   // 2024-05-22
	Algorithm string `json:"Algorithm,omitempty"` // TC3-HMAC-SHA256
	Service   string `json:"Service,omitempty"`
	SecretId  string `json:"SecretId,omitempty"`
	SecretKey string `json:"SecretKey,omitempty"`

	Timestamp int `json:"-"`

	Model          string            `json:"model,omitempty"` // lke-text-embedding-v1
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	client         *http.Client      `json:"-"`
}

func (component *LKEDocumentEmbedder) Initialization(ctx context.Context, input map[string]interface{}) error {
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

	if component.Service == "" {
		component.Service = "lkeap"
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

	if component.SecretId == "" {
		component.SecretId = config.AIBaseURL
	}
	if component.SecretKey == "" {
		component.SecretKey = config.AIAPIkey
	}

	return nil
}
func (component *LKEDocumentEmbedder) Run(ctx context.Context, input map[string]interface{}) error {
	documentChunksObj, has := input["documentChunks"]
	if !has {
		return errors.New(funcT("input['documentChunks'] cannot be empty"))
	}
	documentChunks := documentChunksObj.([]DocumentChunk)
	vecDocumentChunks := make([]VecDocumentChunk, 0)
	for i := 0; i < len(documentChunks); i++ {
		documentChunk := documentChunks[i]

		bodyMap := make(map[string]interface{}, 0)
		bodyMap["Inputs"] = []string{documentChunk.Markdown}
		bodyMap["Model"] = component.Model
		bodyByte, err := httpPostLKEBody(component.client, component.SecretId, component.SecretKey, component.Host, component.Algorithm, component.Service, component.Version, component.Action, component.Region, bodyMap)
		if err != nil {
			input[errorKey] = err
			return err
		}

		rs := struct {
			Response struct {
				Data []struct {
					Embedding []float64 `json:"Embedding,omitempty"`
				} `json:"Data,omitempty"`
			} `json:"Response,omitempty"`
		}{}
		err = json.Unmarshal(bodyByte, &rs)
		if err != nil {
			input[errorKey] = err
			return err
		}
		if len(rs.Response.Data) < 1 {
			err := errors.New("httpPostLKEBody data is empty")
			input[errorKey] = err
			return err
		}
		embedding, err := vecSerializeFloat64(rs.Response.Data[0].Embedding)
		if err != nil {
			input[errorKey] = err
			return err
		}
		documentChunks[i].Embedding = embedding

		vecdc := VecDocumentChunk{}
		vecdc.Id = documentChunks[i].Id
		vecdc.DocumentID = documentChunks[i].DocumentID
		vecdc.KnowledgeBaseID = documentChunks[i].KnowledgeBaseID
		vecdc.SortNo = documentChunks[i].SortNo
		vecdc.Status = 2
		vecdc.Embedding = embedding
		vecDocumentChunks = append(vecDocumentChunks, vecdc)
	}
	input["documentChunks"] = documentChunks
	input["vecDocumentChunks"] = vecDocumentChunks
	return nil
}

// LKETextEmbedder  LKE向量化字符串文本
type LKETextEmbedder struct {
	Host   string `json:"Host,omitempty"`   // lkeap.tencentcloudapi.com
	Action string `json:"Action,omitempty"` // GetEmbedding
	Region string `json:"Region,omitempty"` // ap-guangzhou

	Version   string `json:"Version,omitempty"`   // 2024-05-22
	Algorithm string `json:"Algorithm,omitempty"` // TC3-HMAC-SHA256
	Service   string `json:"Service,omitempty"`
	SecretId  string `json:"SecretId,omitempty"`
	SecretKey string `json:"SecretKey,omitempty"`

	Timestamp int `json:"-"`

	Model          string            `json:"model,omitempty"` // lke-text-embedding-v1
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

	if component.Service == "" {
		component.Service = "lkeap"
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

	if component.SecretId == "" {
		component.SecretId = config.AIBaseURL
	}
	if component.SecretKey == "" {
		component.SecretKey = config.AIAPIkey
	}

	return nil
}
func (component *LKETextEmbedder) Run(ctx context.Context, input map[string]interface{}) error {
	queryObj, has := input["query"]
	if !has {
		return errors.New(funcT("input['query'] cannot be empty"))
	}
	bodyMap := make(map[string]interface{}, 0)
	bodyMap["Inputs"] = []string{queryObj.(string)}
	bodyMap["Model"] = component.Model
	bodyByte, err := httpPostLKEBody(component.client, component.SecretId, component.SecretKey, component.Host, component.Algorithm, component.Service, component.Version, component.Action, component.Region, bodyMap)
	if err != nil {
		input[errorKey] = err
		return err
	}

	rs := struct {
		Response struct {
			Data []struct {
				Embedding []float64 `json:"Embedding,omitempty"`
			} `json:"Data,omitempty"`
		} `json:"Response,omitempty"`
	}{}
	err = json.Unmarshal(bodyByte, &rs)
	if err != nil {
		input[errorKey] = err
		return err
	}
	if len(rs.Response.Data) < 1 {
		err := errors.New("httpPostLKEBody data is empty")
		input[errorKey] = err
		return err
	}
	input["embedding"] = rs.Response.Data[0].Embedding
	return nil
}

// LKEDocumentChunkReranker  LKE对DocumentChunks进行重新排序
type LKEDocumentChunkReranker struct {
	Host   string `json:"Host,omitempty"`   // lkeap.tencentcloudapi.com
	Action string `json:"Action,omitempty"` // GetEmbedding
	Region string `json:"Region,omitempty"` // ap-guangzhou

	Version   string `json:"Version,omitempty"`   // 2024-05-22
	Algorithm string `json:"Algorithm,omitempty"` // TC3-HMAC-SHA256
	Service   string `json:"Service,omitempty"`
	SecretId  string `json:"SecretId,omitempty"`
	SecretKey string `json:"SecretKey,omitempty"`

	Timestamp int `json:"-"`

	Model          string            `json:"model,omitempty"` // lke-text-embedding-v1
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`

	// TopK 检索多少条
	TopK int `json:"topK,omitempty"`
	// Score ranker的score匹配分数
	Score  float32      `json:"score,omitempty"`
	client *http.Client `json:"-"`
}

func (component *LKEDocumentChunkReranker) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Host == "" {
		component.Host = "lkeap.tencentcloudapi.com"
	}
	if component.Action == "" {
		component.Action = "RunRerank"
	}
	if component.Region == "" {
		component.Region = "ap-guangzhou"
	}
	if component.Version == "" {
		component.Version = "2024-05-22"
	}

	if component.Model == "" {
		component.Model = "lke-reranker-base"
	}

	if component.Algorithm == "" {
		component.Algorithm = "TC3-HMAC-SHA256"
	}

	if component.Service == "" {
		component.Service = "lkeap"
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

	if component.SecretId == "" {
		component.SecretId = config.AIBaseURL
	}
	if component.SecretKey == "" {
		component.SecretKey = config.AIAPIkey
	}

	return nil
}
func (component *LKEDocumentChunkReranker) Run(ctx context.Context, input map[string]interface{}) error {
	topK := 0
	var score float32 = 0.0
	dcs, has := input["documentChunks"]
	if !has || dcs == nil {
		err := errors.New(funcT("input['documentChunks'] cannot be empty"))
		input[errorKey] = err
		return err
	}
	queryObj, has := input["query"]
	if !has {
		return errors.New(funcT("input['query'] cannot be empty"))
	}
	query := queryObj.(string)
	if query == "" {
		return errors.New(funcT("input['query'] cannot be empty"))
	}

	tId, has := input["topK"]
	if has {
		topK = tId.(int)
	}
	if topK == 0 {
		topK = component.TopK
	}
	if topK == 0 {
		topK = 5
	}
	disId, has := input["score"]
	if has {
		score = disId.(float32)
	}
	if score <= 0 {
		score = component.Score
	}

	documentChunks := dcs.([]DocumentChunk)
	if topK > len(documentChunks) {
		topK = len(documentChunks)
	}
	if len(documentChunks) < 1 { //没有文档,不需要重排
		return nil
	}
	documents := make([]string, 0)
	for i := 0; i < len(documentChunks); i++ {
		documents = append(documents, documentChunks[i].Markdown)
	}

	bodyMap := map[string]interface{}{
		"Query": query,
		"Docs":  documents,
		"Model": component.Model,
	}
	bodyByte, err := httpPostLKEBody(component.client, component.SecretId, component.SecretKey, component.Host, component.Algorithm, component.Service, component.Version, component.Action, component.Region, bodyMap)
	if err != nil {
		input[errorKey] = err
		return err
	}

	rs := struct {
		Response struct {
			ScoreList []float32 `json:"ScoreList,omitempty"`
		} `json:"Response,omitempty"`
	}{}
	err = json.Unmarshal(bodyByte, &rs)
	if err != nil {
		input[errorKey] = err
		return err
	}
	if len(rs.Response.ScoreList) != len(documentChunks) {
		err := errors.New("httpPostLKEBody ScoreList is error")
		input[errorKey] = err
		return err
	}
	for i := 0; i < len(documentChunks); i++ {
		documentChunks[i].Score = rs.Response.ScoreList[i]
	}

	documentChunks = sortDocumentChunksScore(documentChunks, topK, score)
	input["documentChunks"] = documentChunks

	return nil
}

// https://github.com/TencentCloud/signature-process-demo/blob/main/signature-v3/golang/demo.go
func httpPostLKEBody(client *http.Client, secretId, secretKey, host, algorithm, service, version, action, region string, bodyMap map[string]interface{}) ([]byte, error) {
	// 需要设置环境变量 TENCENTCLOUD_SECRET_ID，值为示例的 AKIDz8krbsJ5yKBZQpn74WFkmLPx3*******
	var timestamp int64 = time.Now().Unix()
	// step 1: build canonical request string
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		"application/json; charset=utf-8", host, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"
	// 序列化请求体
	payloadBytes, err := json.Marshal(bodyMap)
	// hash 请求体
	hashedRequestPayload := sha256hex(string(payloadBytes))
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)
	//fmt.Println(canonicalRequest)

	// step 2: build string to sign
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := sha256hex(canonicalRequest)
	string2sign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest)
	//fmt.Println(string2sign)

	// step 3: sign string
	secretDate := hmacsha256(date, "TC3"+secretKey)
	secretService := hmacsha256(service, secretDate)
	secretSigning := hmacsha256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(hmacsha256(string2sign, secretSigning)))
	//fmt.Println(signature)

	// step 4: build authorization
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		secretId,
		credentialScope,
		signedHeaders,
		signature)
	//fmt.Println(authorization)

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
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Region", region)

	/*
		curl := fmt.Sprintf(`curl -X POST https://%s\
		-H "Authorization: %s"\
		-H "Content-Type: application/json; charset=utf-8"\
		-H "Host: %s"
		-H "X-TC-Action: %s"\
		-H "X-TC-Timestamp: %d"\
		-H "X-TC-Version: %s"\
		-H "X-TC-Region: %s"\
		-d '%s'`, host, authorization, host, action, timestamp, version, region, payload)
		fmt.Println(curl)
	*/
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyByte, err := io.ReadAll(resp.Body)
	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(bodyByte))
	}

	return bodyByte, err
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
