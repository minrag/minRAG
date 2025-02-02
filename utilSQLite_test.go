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
	"context"
	"fmt"
	"testing"

	"gitee.com/chunanyong/zorm"
)

func TestVecQuery(t *testing.T) {
	ctx := context.Background()
	embedder := componentMap["OpenAITextEmbedder"]
	output, err := embedder.Run(ctx, map[string]interface{}{"query": "I am a technical developer from China, primarily using Java, Go, and Python as my development languages."})
	if err != nil {
		t.Fatal(err)
	}
	//需要使用bge-m3模型进行embedding
	embedding := output["embedding"].([]float64)
	query, _ := vecSerializeFloat64(embedding)
	finder := zorm.NewSelectFinder(tableVecDocumentChunkName, "rowid,distance,*").Append("WHERE embedding MATCH ? ORDER BY distance LIMIT 5", query)
	datas := make([]Document, 0)
	zorm.Query(ctx, finder, &datas, nil)
	fmt.Println(len(datas))
	for i := 0; i < len(datas); i++ {
		data := datas[i]
		fmt.Println(data.DocumentID, data.Distance)
	}

}

func TestDocumentSplitter(t *testing.T) {
	ctx := context.Background()
	documentSplitter := componentMap["DocumentSplitter"]
	input := make(map[string]interface{}, 0)
	input["document"] = &Document{Markdown: "我是中国人,我爱中国."}
	output, err := documentSplitter.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(output["documents"])

}
