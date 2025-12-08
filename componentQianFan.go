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
)

// https://cloud.baidu.com/doc/qianfan-api/s/2m7u4zt74
// QianFanDocumentChunkReranker  百度千帆的重排序模型
type QianFanDocumentChunkReranker struct {
	DocumentChunkReranker
}

func (component *QianFanDocumentChunkReranker) Initialization(ctx context.Context, input map[string]any) error {
	if component.Model == "" {
		return errors.New("Initialization QianFanDocumentChunkReranker error:Model is empty")
	}
	if component.BaseURL == "" {
		// 兼容 Jina
		component.BaseURL = config.AIBaseURL + "/rerank"
	}

	component.DocumentChunkReranker.Initialization(ctx, input)

	return nil
}
func (component *QianFanDocumentChunkReranker) Run(ctx context.Context, input map[string]any) error {
	query, topN, score, documentChunks, documents, err := component.checkRerankParameter(ctx, input)
	if err != nil {
		input[errorKey] = err
		return err
	}
	if documentChunks == nil {
		return nil
	}

	bodyMap := map[string]any{
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
