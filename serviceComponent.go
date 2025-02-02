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
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const (
	errorKey      string = "__error__"
	nextComponent string = "__next__"
	endValue      string = "__end__"
)

// componentTypeMap 组件类型对照,key是类型名称,value是组件实例
var componentTypeMap = map[string]IComponent{
	"DocumentSplitter":   &DocumentSplitter{},
	"OpenAITextEmbedder": &OpenAITextEmbedder{},
}

// componentMap 组件的Map,从数据查询拼装参数
var componentMap = make(map[string]IComponent, 0)

// IComponent 组件的接口
type IComponent interface {
	Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

func init() {
	initComponentMap()
}

// initComponentMap 初始化componentMap
func initComponentMap() {
	componentMap = make(map[string]IComponent, 0)
	finder := zorm.NewSelectFinder(tableComponentName).Append("WHERE status=1")
	cs := make([]Component, 0)
	ctx := context.Background()
	zorm.Query(ctx, finder, &cs, nil)
	for i := 0; i < len(cs); i++ {
		c := cs[i]
		component, has := componentTypeMap[c.Id]
		if component == nil || (!has) {
			continue
		}
		if c.Parameter == "" {
			componentMap[c.Id] = component
			continue
		}
		err := json.Unmarshal([]byte(c.Parameter), component)
		if err != nil {
			FuncLogError(ctx, err)
			continue
		}
		componentMap[c.Id] = component
	}
}

// Pipeline 流水线也是组件
type Pipeline struct {
}

func (component *Pipeline) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return input, nil
}

type DocumentSplitter struct {
	SplitBy      []string `json:"splitBy,omitempty"`
	SplitLength  int      `json:"splitLength,omitempty"`
	SplitOverlap int      `json:"splitOverlap,omitempty"`
}

func (component *DocumentSplitter) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	document, has := input["document"].(*Document)
	if document == nil || (!has) {
		err := errors.New(funcT("The document of DocumentSplitter cannot be empty"))
		input[errorKey] = err
		return input, err
	}
	if len(component.SplitBy) < 1 {
		component.SplitBy = []string{"\f", "\n\n", "\n", "。", "!", ".", ";", "，", ",", " "}
	}
	if component.SplitLength == 0 {
		component.SplitLength = 500
	}
	if component.SplitOverlap == 0 {
		component.SplitOverlap = 30
	}
	// 递归分割
	chunks := component.recursiveSplit(document.Markdown, 0)

	if len(chunks) < 1 {
		return input, nil
	}

	// 最多合并3个短文本
	for j := 0; j < 3; j++ {
		chunks = component.mergeChunks(chunks)
	}

	// 处理文本重叠,感觉没有必要了,还会破坏文本的连续性
	now := time.Now().Format("2006-01-02 15:04:05")
	documents := make([]Document, 0)
	for i := 0; i < len(chunks); i++ {
		chunk := chunks[i]
		temp := *document
		temp.Id = FuncGenerateStringID()
		temp.Markdown = chunk
		temp.CreateTime = now
		temp.UpdateTime = now
		temp.DocumentID = document.Id
		documents = append(documents, temp)
	}
	input["documents"] = documents
	return input, nil
}

// recursiveSplit 递归分割实现
func (component *DocumentSplitter) recursiveSplit(text string, depth int) []string {
	chunks := make([]string, 0)
	// 终止条件：处理完所有分隔符
	if depth >= len(component.SplitBy) {
		if text != "" {
			return append(chunks, text)
		}
		return chunks
	}

	currentSep := component.SplitBy[depth]
	parts := strings.Split(text, currentSep)
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		partContent := part
		if i < len(parts)-1 { //不是最后一个
			partContent = partContent + currentSep
		}

		// 处理超长内容
		if len(part) >= component.SplitLength {
			partLeaf := component.recursiveSplit(partContent, depth+1)
			if len(partLeaf) > 0 {
				chunks = append(chunks, partLeaf...)
			}
			continue
		} else {
			chunks = append(chunks, partContent)
		}
	}
	return chunks
}

// mergeChunks 合并短内容
func (component *DocumentSplitter) mergeChunks(chunks []string) []string {
	// 合并短内容
	for i := 0; i < len(chunks); i++ {
		chunk := chunks[i]
		if len(chunk) >= component.SplitLength || i+1 >= len(chunks) {
			continue
		}
		nextChunk := chunks[i+1]

		// 汉字字符占位3个长度
		if (len(chunk) + len(nextChunk)) > (component.SplitLength*18)/10 {
			continue
		}
		chunks[i] = chunk + nextChunk
		if i+2 >= len(chunks) { //倒数第二个元素,去掉最后一个
			chunks = chunks[:len(chunks)-1]
		} else { // 去掉 i+1 索引元素,合并到了 i 索引
			chunks = append(chunks[:i+1], chunks[i+2:]...)
		}
	}
	return chunks
}

// OpenAITextEmbedder 向量化字符串文本
type OpenAITextEmbedder struct {
	APIKey         string            `json:"apikey,omitempty"`
	Model          string            `json:"model,omitempty"`
	APIBaseURL     string            `json:"apiBaseURL,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	Organization   string            `json:"organization,omitempty"`
	Dimensions     int               `json:"dimensions,omitempty"`
	Client         openai.Client
}

func (component *OpenAITextEmbedder) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	client := openai.NewClient(
		option.WithAPIKey(component.APIKey),
		option.WithBaseURL(component.APIBaseURL),
		option.WithMaxRetries(component.MaxRetries),
	)
	headerOpention := make([]option.RequestOption, 0)
	if len(component.DefaultHeaders) > 0 {
		for k, v := range component.DefaultHeaders {
			headerOpention = append(headerOpention, option.WithHeader(k, v))
		}
	}
	query := input["query"].(string)
	response, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model:          openai.F(component.Model),
		EncodingFormat: openai.F(openai.EmbeddingNewParamsEncodingFormatFloat),
		Input:          openai.F[openai.EmbeddingNewParamsInputUnion](shared.UnionString(query))}, headerOpention...)
	if err != nil {
		return input, err
	}
	input["embedding"] = response.Data[0].Embedding
	return input, err
}

// findAllComponentList 查询所有的组件
func findAllComponentList(ctx context.Context) ([]Component, error) {
	finder := zorm.NewSelectFinder(tableComponentName).Append("order by sortNo desc")
	list := make([]Component, 0)
	err := zorm.Query(ctx, finder, &list, nil)
	return list, err
}
