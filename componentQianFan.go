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

// 百度千帆平台,主要适配Reranker模型
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// https://cloud.baidu.com/doc/qianfan-api/s/2m7u4zt74
// QianFanDocumentChunkReranker  百度千帆的重排序模型
type QianFanDocumentChunkReranker struct {
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

func (component *QianFanDocumentChunkReranker) Initialization(ctx context.Context, input map[string]interface{}) error {
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
		component.BaseURL = config.AIBaseURL + "/reranker"
	}
	if component.DefaultHeaders == nil {
		component.DefaultHeaders = make(map[string]string, 0)
	}

	return nil
}
func (component *QianFanDocumentChunkReranker) Run(ctx context.Context, input map[string]interface{}) error {
	topN := 0
	var score float32 = 0.0
	documentChunks, has := input["documentChunks"].([]DocumentChunk)
	if !has || documentChunks == nil {
		err := errors.New(funcT("input['documentChunks'] cannot be empty"))
		input[errorKey] = err
		return err
	}
	query, has := input["query"].(string)
	if !has || query == "" {
		return errors.New(funcT("input['query'] cannot be empty"))
	}

	topN = input["topN"].(int)
	if topN == 0 {
		topN = component.TopN
	}
	if topN == 0 {
		topN = 5
	}
	score = input["score"].(float32)

	if score <= 0 {
		score = component.Score
	}
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
		"model":     component.Model,
		"query":     query,
		"top_n":     topN,
		"documents": documents,
	}

	rsStringByte, err := httpPostJsonBody(component.client, component.APIKey, component.BaseURL, component.DefaultHeaders, bodyMap)
	if err != nil {
		input[errorKey] = err
		return err
	}

	rs := struct {
		Results []struct {
			Document       string  `json:"document,omitempty"`
			RelevanceScore float32 `json:"relevance_score,omitempty"`
			Index          int     `json:"index,omitempty"`
		} `json:"results,omitempty"`
	}{}

	err = json.Unmarshal(rsStringByte, &rs)
	if err != nil {
		input[errorKey] = err
		return err
	}
	rerankerDCS := make([]DocumentChunk, 0)
	for i := 0; i < len(rs.Results); i++ {
		markdown := rs.Results[i].Document
		for j := 0; j < len(documentChunks); j++ {
			dc := documentChunks[j]
			if markdown == dc.Markdown { //相等
				dc.Score = rs.Results[i].RelevanceScore
				rerankerDCS = append(rerankerDCS, dc)
				break
			}
		}
	}
	rerankerDCS = sortDocumentChunksScore(rerankerDCS, topN, score)
	input["documentChunks"] = rerankerDCS
	return nil
}
