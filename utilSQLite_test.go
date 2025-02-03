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
	finder := zorm.NewSelectFinder(tableVecDocumentChunkName, "rowid,distance as score,*").Append("WHERE embedding MATCH ? ORDER BY score LIMIT 5", query)
	datas := make([]DocumentChunk, 0)
	zorm.Query(ctx, finder, &datas, nil)
	fmt.Println(len(datas))
	for i := 0; i < len(datas); i++ {
		data := datas[i]
		fmt.Println(data.DocumentID, data.Score)
	}

}

func TestDocumentSplitter(t *testing.T) {
	ctx := context.Background()
	documentSplitter := componentMap["DocumentSplitter"]
	input := make(map[string]interface{}, 0)
	input["document"] = &Document{Markdown: "我是中国人,我爱中国。圣诞节,了大家安康金发傲娇考虑实际得分拉萨放假啊十六分是。1。2。3。"}
	output, err := documentSplitter.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	ds, _ := output["documentChunks"]
	documentChunks := ds.([]DocumentChunk)
	for i := 0; i < len(documentChunks); i++ {
		documentChunk := documentChunks[i]
		fmt.Println(documentChunk.Markdown)
	}

}

func TestFtsKeywordRetriever(t *testing.T) {
	ctx := context.Background()
	ftsKeywordRetriever := componentMap["FtsKeywordRetriever"]
	input := make(map[string]interface{}, 0)
	input["query"] = "马斯克"
	output, err := ftsKeywordRetriever.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	ds, _ := output["documentChunks"]
	documentChunks := ds.([]DocumentChunk)
	for i := 0; i < len(documentChunks); i++ {
		documentChunk := documentChunks[i]
		fmt.Println(documentChunk)
	}
}

func TestDocumentChunksRanker(t *testing.T) {
	ctx := context.Background()
	documentChunksRanker := componentMap["DocumentChunksRanker"]
	input := make(map[string]interface{}, 0)
	input["query"] = "你在哪里?"
	documentChunks := make([]DocumentChunk, 3)
	documentChunks[0] = DocumentChunk{Markdown: "我在郑州"}
	documentChunks[1] = DocumentChunk{Markdown: "今天晴天"}
	documentChunks[2] = DocumentChunk{Markdown: "我明天去旅游"}
	input["documentChunks"] = documentChunks
	output, err := documentChunksRanker.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	ds, _ := output["documentChunks"]
	documentChunks = ds.([]DocumentChunk)
	for i := 0; i < len(documentChunks); i++ {
		documentChunk := documentChunks[i]
		fmt.Println(documentChunk)
	}
}

func TestPromptBuilder(t *testing.T) {
	ctx := context.Background()
	promptBuilder := componentMap["PromptBuilder"]
	input := make(map[string]interface{}, 0)
	input["query"] = "你在哪里?"
	documentChunks := make([]DocumentChunk, 3)
	documentChunks[0] = DocumentChunk{Markdown: "我在郑州"}
	documentChunks[1] = DocumentChunk{Markdown: "今天晴天"}
	documentChunks[2] = DocumentChunk{Markdown: "我明天去旅游"}
	input["documentChunks"] = documentChunks
	output, err := promptBuilder.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(output["prompt"])

	openAIChatMessageMemory := componentMap["OpenAIChatMessageMemory"]
	output, err = openAIChatMessageMemory.Run(ctx, output)

	openAIChatCompletion := componentMap["OpenAIChatCompletion"]
	fmt.Println(openAIChatCompletion)
	output, err = openAIChatCompletion.Run(ctx, output)

}
