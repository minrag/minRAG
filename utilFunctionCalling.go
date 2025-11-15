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
	"strconv"

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
		"description": "根据用户问题和提供的知识库文档结构树,找出所有可能包含答案的知识库文档节点ID,也可以使用query在文档中进行全文检索,如果可能至少返回5个节点.如果函数返回的节点内容和用户问题关系不紧密,可以多次调用此函数,获取其他的节点内容",
		"parameters": {
			"type": "object",
			"properties": {
			    "documentIds": {
					"type": "array",
                    "items": {"type": "string"},
					"description": "要检索的知识库文档ID,可以配合 query 在文档中全文检索关联的节点ID"
				},
				"query": {
					"type": "string",
					"description": "在documentIds文档中全文检索关联的节点ID,需要同时传递documentIds"
				},
				"nodeIds": {
					"type": "array",
                    "items": {"type": "string"},
					"description": "要检索的知识库文档节点ID"
				}
			},
			"additionalProperties": false
		}
	}
}`

// FCSearchKnowledgeBase 查询本地知识库的函数
type FCSearchKnowledgeBase struct {
	//接受模型返回的 arguments
	DocumentIds []string `json:"documentIds,omitempty"`
	Query       string   `json:"query,omitempty"`
	NodeIds     []string `json:"nodeIds,omitempty"`

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
	if len(fc.NodeIds) < 1 && len(fc.DocumentIds) < 1 && len(fc.Query) < 1 {
		return "", nil
	}
	knowledgeBaseID := ""
	if ctx.Value("knowledgeBaseID") != nil {
		knowledgeBaseID = ctx.Value("knowledgeBaseID").(string)
	}
	pageSize := 100
	tocChunks := make([]DocumentChunk, 0)
	f_dc := zorm.NewSelectFinder(tableDocumentChunkName, "id,markdown").Append("WHERE 1=1 ")
	f_dc.SelectTotalCount = false
	if knowledgeBaseID != "" {
		f_dc.Append(" and knowledgeBaseID like ?", knowledgeBaseID+"%")
	}
	if len(fc.NodeIds) > 0 {
		f_dc.Append("and id in (?)", fc.NodeIds)
	}
	if len(fc.DocumentIds) > 0 {
		f_dc.Append("and documentID in (?)", fc.DocumentIds)
	}
	if len(fc.Query) > 0 {
		// BM25的FTS5实现在返回结果之前将结果乘以-1,得分越小(数值上更负),表示匹配越好
		f_fts := zorm.NewFinder().Append("SELECT id from fts_document_chunk where fts_document_chunk match jieba_query(?)", fc.Query)
		if len(fc.DocumentIds) > 0 {
			f_fts.Append("and documentID in (?)", fc.DocumentIds)
		}
		// BM25的FTS5实现在返回结果之前将结果乘以-1,查询时再乘以-1
		f_fts.Append("and -1*rank >= ?", 0.3)
		f_fts.Append("ORDER BY -1*rank DESC LIMIT " + strconv.Itoa(pageSize))

		f_dc.Append("and id in (").AppendFinder(f_fts)
		f_dc.Append(")")
	}
	page := zorm.NewPage()
	page.PageSize = pageSize
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
