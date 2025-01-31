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

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// IComponent 组件的接口
type IComponent interface {
	Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

// BaseComponent 基础组件属性
type BaseComponent struct {
	APIKey     string
	Model      string
	APIBaseURL string
	Header     map[string]interface{}
	Timeout    int
	MaxRetries int
}

type OpenAITextEmbedder struct {
	BaseComponent
	Organization string
	Dimensions   int
}

func (component *OpenAITextEmbedder) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	client := openai.NewClient(
		option.WithAPIKey(component.APIKey),
		option.WithBaseURL(component.APIBaseURL),
		option.WithMaxRetries(component.MaxRetries),
	)
	headerOpention := make([]option.RequestOption, 0)
	if len(component.Header) > 0 {
		for k, v := range component.Header {
			headerOpention = append(headerOpention, option.WithHeader(k, v.(string)))
		}
	}
	response, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{Model: openai.String(component.Model)}, headerOpention...)
	if err != nil {
		return input, err
	}
	fmt.Println(response.JSON.RawJSON())
	return input, err
}
