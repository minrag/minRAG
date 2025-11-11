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
	"encoding/json"

	"gitee.com/chunanyong/zorm"
)

var functionCallingMap = make(map[string]IToolFunctionCalling, 0)

const (
	fcSearchKnowledgeBaseName = "search_knowledge_base"
)

func init() {
	ctx := context.Background()

	//本地知识库检索函数
	fcSearchKnowledgeBase := FCSearchKnowledgeBase{}
	searchKnowledgeBase, err := fcSearchKnowledgeBase.Initialization(ctx, search_knowledge_base_json)
	if err == nil {
		functionCallingMap[fcSearchKnowledgeBaseName] = searchKnowledgeBase
	}
}

// IToolFunctionCalling 函数调用接口
type IToolFunctionCalling interface {
	// Initialization 初始化方法
	Initialization(ctx context.Context, descriptionJson string) (IToolFunctionCalling, error)
	//获取描述的Map
	Description(ctx context.Context) interface{}
	// Run 执行方法
	Run(ctx context.Context, arguments string) (string, error)
}

// search_knowledge_base_json 查询知识库的函数json字符串
var search_knowledge_base_json = `{
	"type": "function",
	"function": {
		"name": "` + fcSearchKnowledgeBaseName + `",
		"description": "根据用户问题和提供的知识库文档结构树,找出所有可能包含答案的知识库文档节点ID,如果可能至少返回5个节点.如果函数返回的节点内容和用户问题关系不紧密,可以多次调用此函数,获取其他的节点内容",
		"parameters": {
			"type": "object",
			"properties": {
				"nodeIds": {
					"type": "array",
                    "items": {"type": "string"},
					"description": "要检索的知识库文档节点ID"
				}
			},
			"required": ["nodeIds"],
			"additionalProperties": false
		}
	}
}`

// FCSearchKnowledgeBase 查询本地知识库的函数
type FCSearchKnowledgeBase struct {
	//接受模型返回的 arguments
	NodeIds        []string               `json:"nodeIds,omitempty"`
	DescriptionMap map[string]interface{} `json:"-"`
}

func (fc FCSearchKnowledgeBase) Initialization(ctx context.Context, descriptionJson string) (IToolFunctionCalling, error) {
	dm := make(map[string]interface{})
	if descriptionJson == "" {
		return fc, nil
	}
	err := json.Unmarshal([]byte(descriptionJson), &dm)
	if err != nil {
		return fc, err
	}
	fc.DescriptionMap = dm
	return fc, nil
}

// 获取描述的Map
func (fc FCSearchKnowledgeBase) Description(ctx context.Context) interface{} {
	return fc.DescriptionMap
}

// Run 执行方法
func (fc FCSearchKnowledgeBase) Run(ctx context.Context, arguments string) (string, error) {
	if arguments == "" {
		return "", nil
	}
	err := json.Unmarshal([]byte(arguments), &fc)
	if err != nil {
		return "", nil
	}
	if len(fc.NodeIds) < 1 {
		return "", nil
	}

	tocChunks := make([]DocumentChunk, 0)
	f_dc := zorm.NewSelectFinder(tableDocumentChunkName, "id,markdown").Append("WHERE id in (?)", fc.NodeIds)
	page := zorm.NewPage()
	page.PageSize = 100
	err = zorm.Query(ctx, f_dc, &tocChunks, page)
	if err != nil {
		return "", nil
	}
	resultByte, err := json.Marshal(tocChunks)
	if err != nil {
		return "", nil
	}
	return string(resultByte), nil
}
