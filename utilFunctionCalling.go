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
	"sync"

	"gitee.com/chunanyong/zorm"
)

var functionCallingMap = make(map[string]IToolFunctionCalling, 0)

const (
	// fcSearchDocumentTOCByIDName 检索文档目录数据函数
	fcSearchDocumentTOCByIDName = "search_document_toc_by_id"
	// fcSearchContentByNodeName 根据节点检索文档内容
	fcSearchContentByNodeName = "search_content_by_node"
	// fcSearchContentByKeywordName 根据关键字检索文档内容
	fcSearchContentByKeywordName = "search_content_by_keyword"
	// fcWebSearchName 联网搜索
	fcWebSearchName = "web_search"
)

func init() {
	ctx := context.Background()

	//本地知识库检索文档目录数据函数
	fcSearchDocumentTOCById := FCSearchDocumentTOCById{}
	searchDocumentTOCById, err := fcSearchDocumentTOCById.Initialization(ctx, search_document_toc_by_id_json)
	if err == nil {
		functionCallingMap[fcSearchDocumentTOCByIDName] = searchDocumentTOCById
	}

	//本地知识库检索节点函数
	fcSearchDocumentByNode := FCSearchContentByNode{}
	searchDocumentByNode, err := fcSearchDocumentByNode.Initialization(ctx, search_content_by_node_json)
	if err == nil {
		functionCallingMap[fcSearchContentByNodeName] = searchDocumentByNode
	}

	//本地知识库检索关键字函数
	fcSearchDocumentByKeyword := FCSearchDocumentByKeyword{}
	searchDocumentByKeyword, err := fcSearchDocumentByKeyword.Initialization(ctx, search_content_by_keyword_json)
	if err == nil {
		functionCallingMap[fcSearchContentByKeywordName] = searchDocumentByKeyword
	}

	//联网搜索函数
	fcWebSearch := FCWebSearch{}
	webSearch, err := fcWebSearch.Initialization(ctx, web_search_json)
	if err == nil {
		functionCallingMap[fcWebSearchName] = webSearch
	}
}

// IToolFunctionCalling 函数调用接口
type IToolFunctionCalling interface {
	// Initialization 初始化方法
	Initialization(ctx context.Context, descriptionJson string) (IToolFunctionCalling, error)
	//获取描述的Map
	Description(ctx context.Context) interface{}
	// Run 执行方法
	Run(ctx context.Context, arguments string, intput map[string]any) (string, error)
}

// search_document_toc_by_id_json 查询文档目录的结构树函数json字符串
var search_document_toc_by_id_json = `{
	"type": "function",
	"function": {
		"name": "` + fcSearchDocumentTOCByIDName + `",
		"description": "根据用户问题和提供的文档列表,找出所有可能包含答案的文档目录结构数据,再配合使用` + fcSearchContentByNodeName + `函数,查找具体目录节点的内容.如果函数返回的文档目录数据和用户问题关系不紧密,可以多次调用此函数,获取其他的文档目录数据.可以和其他检索搜索函数配合使用",
		"parameters": {
			"type": "object",
			"properties": {
				"documentIds": {
					"type": "array",
                    "items": {"type": "string"},
					"description": "要检索的知识库文档节点ID"
				}
			},
			"required": ["documentIds"],
			"additionalProperties": false
		}
	}
}`

// FCSearchDocumentById 查询本地知识库的文档目录
type FCSearchDocumentTOCById struct {
	DocumentIds    []string       `json:"documentIds,omitempty"`
	DescriptionMap map[string]any `json:"-"`
}

func (fc FCSearchDocumentTOCById) Initialization(ctx context.Context, descriptionJson string) (IToolFunctionCalling, error) {
	dm := make(map[string]any)
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
func (fc FCSearchDocumentTOCById) Description(ctx context.Context) interface{} {
	return fc.DescriptionMap
}

// Run 执行方法
func (fc FCSearchDocumentTOCById) Run(ctx context.Context, arguments string, intput map[string]any) (string, error) {
	if arguments == "" {
		return "", nil
	}
	err := json.Unmarshal([]byte(arguments), &fc)
	if err != nil {
		return "", nil
	}
	if len(fc.DocumentIds) < 1 {
		return "", nil
	}
	knowledgeBaseID := ""
	if intput["knowledgeBaseID"] != nil {
		knowledgeBaseID = intput["knowledgeBaseID"].(string)
	}

	documentTOCs := make([]Document, 0)
	f_dc := zorm.NewSelectFinder(tableDocumentName, "id,name,toc").Append("WHERE 1=1 ")
	f_dc.SelectTotalCount = false
	if knowledgeBaseID != "" {
		f_dc.Append(" and knowledgeBaseID like ?", knowledgeBaseID+"%")
	}
	if len(fc.DocumentIds) > 0 {
		f_dc.Append("and id in (?)", fc.DocumentIds)
	}

	page := zorm.NewPage()
	page.PageSize = len(fc.DocumentIds)
	err = zorm.Query(ctx, f_dc, &documentTOCs, page)
	if err != nil {
		return "", nil
	}
	resultByte, err := json.Marshal(documentTOCs)
	if err != nil {
		return "", nil
	}
	return string(resultByte), nil
}

// search_content_by_node_json 查询知识库的函数json字符串
var search_content_by_node_json = `{
	"type": "function",
	"function": {
		"name": "` + fcSearchContentByNodeName + `",
		"description": "根据用户问题,先使用` + fcSearchDocumentTOCByIDName + `函数找出所有可能包含答案的文档目录,再使用` + fcSearchContentByNodeName + `函数查找找出所有可能包含答案的节点ID内容.如果函数返回的节点内容和用户问题关系不紧密,可以多次调用此函数,获取其他的节点内容.可以和其他检索搜索函数配合使用",
		"parameters": {
			"type": "object",
			"properties": {
				"nodeIds": {
					"type": "array",
                    "items": {"type": "string"},
					"description": "要检索的文档节点ID"
				}
			},
			"required": ["nodeIds"],
			"additionalProperties": false
		}
	}
}`

// FCSearchContentByNode 查询本地知识库的函数
type FCSearchContentByNode struct {
	NodeIds        []string       `json:"nodeIds,omitempty"`
	DescriptionMap map[string]any `json:"-"`
}

func (fc FCSearchContentByNode) Initialization(ctx context.Context, descriptionJson string) (IToolFunctionCalling, error) {
	dm := make(map[string]any)
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
func (fc FCSearchContentByNode) Description(ctx context.Context) interface{} {
	return fc.DescriptionMap
}

// Run 执行方法
func (fc FCSearchContentByNode) Run(ctx context.Context, arguments string, intput map[string]any) (string, error) {
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
	knowledgeBaseID := ""
	if intput["knowledgeBaseID"] != nil {
		knowledgeBaseID = intput["knowledgeBaseID"].(string)
	}

	tocChunks := make([]DocumentChunk, 0)
	f_dc := zorm.NewSelectFinder(tableDocumentChunkName, "id,markdown").Append("WHERE 1=1 ")
	f_dc.SelectTotalCount = false
	if knowledgeBaseID != "" {
		f_dc.Append(" and knowledgeBaseID like ?", knowledgeBaseID+"%")
	}
	if len(fc.NodeIds) > 0 {
		f_dc.Append("and id in (?)", fc.NodeIds)
	}

	page := zorm.NewPage()
	page.PageSize = len(fc.NodeIds)
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

// search_content_by_keyword_json 查询知识库的函数json字符串
var search_content_by_keyword_json = `{
	"type": "function",
	"function": {
		"name": "` + fcSearchContentByKeywordName + `",
		"description": "根据用户问题和提供的知识库文档结构树,全文检索query关键字,如果函数返回的内容和用户问题关系不紧密,可以多次调用此函数,获取其他的内容.可以和web_search网络联网搜索配合使用",
		"parameters": {
			"type": "object",
			"properties": {
			"query": {
					"type": "string",
					"description": "全文检索关联的内容"
				},
			    "documentIds": {
					"type": "array",
                    "items": {"type": "string"},
					"description": "要检索的知识库文档ID,可以配合 query 在文档中全文检索关联内容,可以为空,非必要字段"
				}
			},
			"required": ["query"],
			"additionalProperties": false
		}
	}
}`

// FCSearchDocumentByKeyword 根据关键字全文检索本地知识库的函数
type FCSearchDocumentByKeyword struct {
	//接受模型返回的 arguments
	DocumentIds []string `json:"documentIds,omitempty"`
	Query       string   `json:"query,omitempty"`

	DescriptionMap map[string]any `json:"-"`
}

func (fc FCSearchDocumentByKeyword) Initialization(ctx context.Context, descriptionJson string) (IToolFunctionCalling, error) {
	dm := make(map[string]any)
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
func (fc FCSearchDocumentByKeyword) Description(ctx context.Context) interface{} {
	return fc.DescriptionMap
}

// Run 执行方法
func (fc FCSearchDocumentByKeyword) Run(ctx context.Context, arguments string, intput map[string]any) (string, error) {
	if arguments == "" {
		return "", nil
	}
	err := json.Unmarshal([]byte(arguments), &fc)
	if err != nil {
		return "", nil
	}
	if len(fc.Query) < 1 {
		return "", nil
	}
	knowledgeBaseID := ""
	if intput["knowledgeBaseID"] != nil {
		knowledgeBaseID = intput["knowledgeBaseID"].(string)
	}
	var score float32 = 0.3
	if intput["score"] != nil {
		score = intput["score"].(float32)
	}

	topN := 5
	if intput["topN"] != nil {
		topN = intput["topN"].(int)
	}

	// BM25的FTS5实现在返回结果之前将结果乘以-1,得分越小(数值上更负),表示匹配越好
	finder := zorm.NewFinder().Append("SELECT id,markdown from fts_document_chunk where fts_document_chunk match jieba_query(?)", fc.Query)
	finder.SelectTotalCount = false
	finder.Append(" and markdown !=?  and markdown is not null", "")
	if len(fc.DocumentIds) > 0 {
		finder.Append(" and documentID in (?)", fc.DocumentIds)
	}
	if knowledgeBaseID != "" {
		finder.Append(" and knowledgeBaseID like ?", knowledgeBaseID+"%")
	}
	if score > 0.0 { // BM25的FTS5实现在返回结果之前将结果乘以-1,查询时再乘以-1
		finder.Append("and -1*rank >= ?", score)
	}
	// BM25的FTS5实现在返回结果之前将结果乘以-1,查询时再乘以-1

	finder.Append("ORDER BY -1*rank DESC LIMIT " + strconv.Itoa(topN))

	documentChunks := make([]DocumentChunk, 0)
	err = zorm.Query(ctx, finder, &documentChunks, nil)
	if err != nil {
		//input[errorKey] = err
		return "", err
	}
	resultByte, _ := json.Marshal(documentChunks)
	return string(resultByte), nil
}

// web_search_json 网络搜索json字符串
var web_search_json = `{
	"type": "function",
	"function": {
		"name": "` + fcWebSearchName + `",
		"description": "用于信息检索的网络联网搜索,可以多次调用.可以和search_content_by_node本地知识库检索配合使用",
		"parameters": {
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "要网络联网搜索的内容"
				}
			},
			"required": ["query"],
			"additionalProperties": false
		}
	}
}`

// FCWebSearch 网络搜索的函数
type FCWebSearch struct {
	//接受模型返回的 arguments
	Query string `json:"query,omitempty"`

	DescriptionMap map[string]any `json:"-"`
}

func (fc FCWebSearch) Initialization(ctx context.Context, descriptionJson string) (IToolFunctionCalling, error) {
	dm := make(map[string]any)
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
func (fc FCWebSearch) Description(ctx context.Context) interface{} {
	return fc.DescriptionMap
}

// Run 执行方法
func (fc FCWebSearch) Run(ctx context.Context, arguments string, intput map[string]any) (string, error) {
	if arguments == "" {
		return "", nil
	}
	err := json.Unmarshal([]byte(arguments), &fc)
	if err != nil {
		return "", nil
	}
	if len(fc.Query) < 1 {
		return "", nil
	}
	webSearch := WebSearch{}

	webSearch.Depth = 2
	webSearch.QuerySelector = []string{"li.b_algo div.b_tpcn"}
	webURL := "https://www.bing.com/search?q=" + fc.Query
	input1 := make(map[string]any, 0)
	webSearch.Initialization(ctx, input1)

	document := &Document{}
	input1["document"] = document
	input1["webScraper_webURL"] = webURL
	hrefSlice, _ := webSearch.FetchPage(ctx, document, input1)
	if len(hrefSlice) < 1 {
		return "", nil
	}
	tokN := webSearch.TopN
	if len(hrefSlice) < tokN {
		tokN = len(hrefSlice)
	}
	hrefWS := &WebSearch{}
	hrefWS.WebScraper.Depth = 1
	hrefWS.WebScraper.QuerySelector = []string{"body"}
	hrefWS.WebScraper.Initialization(ctx, nil)
	webSerachDocuments := make([]Document, 0)
	// 使用WaitGroup和Mutex的基本异步方案
	var wg sync.WaitGroup
	var mu sync.Mutex
	for j := 0; j < tokN; j++ {
		wg.Add(1)
		go func(href string) {
			defer wg.Done()
			document := &Document{}
			document.Id = href
			hrefInput := make(map[string]any, 0)
			hrefInput["document"] = document
			hrefInput["webScraper_webURL"] = href
			hrefWS.WebScraper.FetchPage(ctx, document, hrefInput)
			if document.Markdown != "" {
				mu.Lock()
				webSerachDocuments = append(webSerachDocuments, *document)
				mu.Unlock()
			}
		}(hrefSlice[j])

	}
	wg.Wait() // 等待所有goroutine完成
	resultByte, _ := json.Marshal(webSerachDocuments)
	return string(resultByte), nil
}
