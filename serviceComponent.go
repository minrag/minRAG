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

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

// componentTypeMap 组件类型对照,key是类型名称,value是组件实例
var componentTypeMap = map[string]IComponent{
	"OpenAITextEmbedder": OpenAITextEmbedder{},
}

// IComponent 组件的接口
type IComponent interface {
	Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

// Pipeline 流水线也是组件
type Pipeline struct {
}

func (component Pipeline) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return input, nil
}

// FileToDocument 把文件转成Document
type FileToDocument struct {
	FilePath string `json:"filePath,omitempty"`
}

func (component FileToDocument) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return input, nil
}

// DocumentCleaner 清理文档的html标签等
type DocumentCleaner struct {
}

func (component DocumentCleaner) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return input, nil
}

// DocumentSplitter 分割文档
type DocumentSplitter struct {
	SplitBy      []string `json:"splitBy,omitempty"`
	SplitLength  int      `json:"splitLength,omitempty"`
	SplitOverlap int      `json:"splitOverlap,omitempty"`
}

func (component DocumentSplitter) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return input, nil
}

// DocumentWriter 将文档向量化保存
type DocumentWriter struct {
}

func (component DocumentWriter) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return input, nil
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

func (component OpenAITextEmbedder) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
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

// OpenAIDocumentEmbedder 向量化文档
type OpenAIDocumentEmbedder struct {
}

func (component OpenAIDocumentEmbedder) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return input, nil
}
