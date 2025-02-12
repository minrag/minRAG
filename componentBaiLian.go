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

// 百度千帆平台,主要适配Reranker模型
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// https://help.aliyun.com/zh/model-studio/developer-reference/text-rerank-api
// BaiLianDocumentChunkReranker  阿里百炼的重排序模型
type BaiLianDocumentChunkReranker struct {
	APIKey         string            `json:"api_key,omitempty"`
	Model          string            `json:"model,omitempty"`
	BaseURL        string            `json:"base_url,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	// Query 需要查询的关键字
	Query string `json:"query,omitempty"`
	// TopN 检索多少条
	TopN int `json:"top_n,omitempty"`
	// Score ranker的score匹配分数
	Score  float32      `json:"score,omitempty"`
	client *http.Client `json:"-"`
}

func (component *BaiLianDocumentChunkReranker) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 180
	}

	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
	}

	if component.APIKey == "" {
		component.APIKey = config.AIAPIkey
	}
	if component.BaseURL == "" {
		if config.AIBaseURL == "" {
			return nil
		}
		index := strings.Index(config.AIBaseURL, "/v1")
		if index <= 0 {
			return nil
		}
		component.BaseURL = config.AIBaseURL[:index] + "/services/rerank/text-rerank/text-rerank"
	}
	if component.DefaultHeaders == nil {
		component.DefaultHeaders = make(map[string]string, 0)
	}
	return nil
}
func (component *BaiLianDocumentChunkReranker) Run(ctx context.Context, input map[string]interface{}) error {
	topN := 0
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

	tId, has := input["topN"]
	if has {
		topN = tId.(int)
	}
	if topN == 0 {
		topN = component.TopN
	}
	if topN == 0 {
		topN = 5
	}
	disId, has := input["score"]
	if has {
		score = disId.(float32)
	}
	if score <= 0 {
		score = component.Score
	}

	documentChunks := dcs.([]DocumentChunk)
	if topN > len(documentChunks) {
		topN = len(documentChunks)
	}
	if len(documentChunks) < 1 { //没有文档,不需要重排
		return nil
	}
	documents := make([]string, 0)
	for i := 0; i < len(documentChunks); i++ {
		documents = append(documents, documentChunks[i].Markdown)
	}

	bodyMap := map[string]interface{}{
		"model": component.Model,
		"input": map[string]interface{}{
			"query":     query,
			"documents": documents,
		},
		"parameters": map[string]interface{}{
			"top_n":            topN,
			"return_documents": true,
		},
	}

	rsStringByte, err := httpPostJsonBody(component.client, component.APIKey, component.BaseURL, component.DefaultHeaders, bodyMap)
	if err != nil {
		input[errorKey] = err
		return err
	}

	rs := struct {
		Output struct {
			Results []struct {
				Document struct {
					Text string `json:"text,omitempty"`
				} `json:"document,omitempty"`
				RelevanceScore float32 `json:"relevance_score,omitempty"`
			} `json:"results,omitempty"`
		} `json:"output,omitempty"`
	}{}

	err = json.Unmarshal(rsStringByte, &rs)
	if err != nil {
		input[errorKey] = err
		return err
	}
	rerankerDCS := make([]DocumentChunk, 0)
	for i := 0; i < len(rs.Output.Results); i++ {
		markdown := rs.Output.Results[i].Document.Text
		for j := 0; j < len(documentChunks); j++ {
			dc := documentChunks[j]
			if markdown == dc.Markdown { //相等
				dc.Score = rs.Output.Results[i].RelevanceScore
				rerankerDCS = append(rerankerDCS, dc)
				break
			}
		}
	}
	rerankerDCS = sortDocumentChunksScore(rerankerDCS, topN, score)
	input["documentChunks"] = rerankerDCS
	return nil
}
