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

// 阿里百炼平台,主要适配Reranker模型
package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// https://help.aliyun.com/zh/model-studio/developer-reference/text-rerank-api
// BaiLianDocumentChunkReranker  阿里百炼的重排序模型
type BaiLianDocumentChunkReranker struct {
	DocumentChunkReranker
}

func (component *BaiLianDocumentChunkReranker) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Model == "" {
		return errors.New("Initialization BaiLianDocumentChunkReranker error:Model is empty")
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
	component.DocumentChunkReranker.Initialization(ctx, input)
	return nil
}
func (component *BaiLianDocumentChunkReranker) Run(ctx context.Context, input map[string]interface{}) error {
	query, topN, score, documentChunks, documents, err := component.checkRerankParameter(ctx, input)
	if err != nil {
		input[errorKey] = err
		return err
	}
	if documentChunks == nil {
		return nil
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
