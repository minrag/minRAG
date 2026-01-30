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

package main

import (
	"context"
	"fmt"
	"testing"

	"gitee.com/chunanyong/zorm"
)

func TestVecLikeQuery(t *testing.T) {
	finder := zorm.NewSelectFinder(tableVecDocumentChunkName).Append("WHERE knowledgeBaseID like ?  LIMIT 5", "%")
	list, _ := zorm.QueryMap(context.Background(), finder, nil)
	fmt.Println(len(list))
	fmt.Println(list)
}

func TestVecQuery(t *testing.T) {
	ctx := context.Background()
	embedder := baseComponentMap["OpenAITextEmbedder"]
	input := map[string]any{"query": "I am a technical developer from China, primarily using Java, Go, and Python as my development languages."}
	err := embedder.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	//需要使用Qwen3-Embedding-8B模型进行embedding
	embedding := input["embedding"].([]float64)
	query, _ := vecSerializeFloat64(embedding)
	finder := zorm.NewSelectFinder(tableVecDocumentChunkName, "rowid,distance as score,*").Append("WHERE embedding MATCH ? ORDER BY score LIMIT 5", query)
	datas := make([]DocumentChunk, 0)
	zorm.Query(ctx, finder, &datas, nil)

	for i := 0; i < len(datas); i++ {
		data := datas[i]
		fmt.Println(data.DocumentID, data.Score)
	}

}
