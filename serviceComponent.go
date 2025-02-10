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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/cloudwego/hertz/pkg/app"
)

const (
	errorKey         string = "__error__"
	nextComponentKey string = "__next__"
	endKey           string = "__end__"
	ifEmptyStop      string = "__ifEmptyStop__"
)

// TODO 缺少 function call 的实现和测试

// componentTypeMap 组件类型对照,key是类型名称,value是组件实例
var componentTypeMap = map[string]IComponent{
	"Pipeline":               &Pipeline{},
	"ChatMessageLogStore":    &ChatMessageLogStore{},
	"OpenAIChatGenerator":    &OpenAIChatGenerator{},
	"OpenAIChatMemory":       &OpenAIChatMemory{},
	"PromptBuilder":          &PromptBuilder{},
	"DocumentChunkReranker":  &DocumentChunkReranker{},
	"FtsKeywordRetriever":    &FtsKeywordRetriever{},
	"VecEmbeddingRetriever":  &VecEmbeddingRetriever{},
	"OpenAITextEmbedder":     &OpenAITextEmbedder{},
	"SQLiteVecDocumentStore": &SQLiteVecDocumentStore{},
	"OpenAIDocumentEmbedder": &OpenAIDocumentEmbedder{},
	"DocumentSplitter":       &DocumentSplitter{},
	"MarkdownConverter":      &MarkdownConverter{},
}

// componentMap 组件的Map,从数据查询拼装参数
var componentMap = make(map[string]IComponent, 0)

// IComponent 组件的接口
type IComponent interface {
	// Initialization 初始化方法
	Initialization(ctx context.Context, input map[string]interface{}) error
	// Run 执行方法
	Run(ctx context.Context, input map[string]interface{}) error
}

func init() {
	initComponentMap()
}

// initComponentMap 初始化componentMap
func initComponentMap() {
	componentMap = make(map[string]IComponent, 0)
	// indexPipeline 比较特殊,默认禁用,为了不让Agent绑定上
	finder := zorm.NewSelectFinder(tableComponentName).Append("WHERE status=1 or id=?", "indexPipeline")
	cs := make([]Component, 0)
	ctx := context.Background()
	zorm.Query(ctx, finder, &cs, nil)
	for i := 0; i < len(cs); i++ {
		c := cs[i]
		componentType, has := componentTypeMap[c.ComponentType]
		if componentType == nil || (!has) {
			continue
		}
		// 使用反射动态创建一个结构体的指针实例
		cType := reflect.TypeOf(componentType).Elem()
		cPtr := reflect.New(cType)
		// 将反射对象转换为接口类型
		component := cPtr.Interface().(IComponent)
		if c.Parameter == "" {
			err := component.Initialization(ctx, nil)
			if err != nil {
				FuncLogError(ctx, err)
				continue
			}
			componentMap[c.Id] = component
			continue
		}
		err := json.Unmarshal([]byte(c.Parameter), component)
		if err != nil {
			FuncLogError(ctx, err)
			continue
		}
		err = component.Initialization(ctx, nil)
		if err != nil {
			FuncLogError(ctx, err)
			continue
		}
		componentMap[c.Id] = component
	}
}

// Pipeline 流水线也是组件
type Pipeline struct {
	Start   string            `json:"start,omitempty"`
	Process map[string]string `json:"process,omitempty"`
}

func (component *Pipeline) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *Pipeline) Run(ctx context.Context, input map[string]interface{}) error {
	return component.runProcess(ctx, input, component.Start)
}
func (component *Pipeline) runProcess(ctx context.Context, input map[string]interface{}, componentName string) error {
	pipelineComponent, has := componentMap[componentName]
	if !has {
		return errors.New(fmt.Sprintf(funcT("The %s component of the pipeline does not exist"), componentName))
	}
	err := pipelineComponent.Run(ctx, input)
	if err != nil {
		input[errorKey] = err
		return err
	}
	errObj, has := input[errorKey]
	if has {
		return errObj.(error)
	}
	_, has = input[endKey]
	if has {
		return nil
	}
	nextComponentName := component.Process[componentName]
	nextComponentObj, has := input[nextComponentKey]
	if has && nextComponentObj.(string) != "" {
		nextComponentName = nextComponentObj.(string)
	}

	if nextComponentName != "" {
		return component.runProcess(ctx, input, nextComponentName)
	}

	return nil
}

// MarkdownConverter markdown文件读取
type MarkdownConverter struct {
	FilePath string `json:"filePath,omitempty"`
}

func (component *MarkdownConverter) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *MarkdownConverter) Run(ctx context.Context, input map[string]interface{}) error {
	document := &Document{}
	documentObj, has := input["document"]
	if has {
		document = documentObj.(*Document)
	}
	filePath := component.FilePath
	if filePath == "" {
		filePath = document.FilePath
	} else {
		document.FilePath = filePath
	}
	if filePath == "" {
		err := errors.New(funcT("The filePath of MarkdownConverter cannot be empty"))
		input[errorKey] = err
		return err
	}

	if document.Markdown == "" {
		markdownByte, err := os.ReadFile(datadir + document.FilePath)
		if err != nil {
			input[errorKey] = err
			return err
		}
		document.Markdown = string(markdownByte)
		document.FileSize = len(markdownByte)
	}
	document.Status = 2
	input["document"] = document
	return nil
}

// DocumentSplitter 文档拆分
type DocumentSplitter struct {
	SplitBy      []string `json:"splitBy,omitempty"`
	SplitLength  int      `json:"splitLength,omitempty"`
	SplitOverlap int      `json:"splitOverlap,omitempty"`
}

func (component *DocumentSplitter) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *DocumentSplitter) Run(ctx context.Context, input map[string]interface{}) error {
	document, has := input["document"].(*Document)
	if document == nil || (!has) {
		err := errors.New(funcT("The document of DocumentSplitter cannot be empty"))
		input[errorKey] = err
		return err
	}
	if len(component.SplitBy) < 1 {
		component.SplitBy = []string{"\f", "\n\n", "\n", "。", "!", ".", ";", "，", ",", " "}
	}
	if component.SplitLength == 0 {
		component.SplitLength = 500
	}
	// 递归分割
	chunks := component.recursiveSplit(document.Markdown, 0)

	if len(chunks) < 1 {
		return nil
	}

	// 合并3次短内容
	for j := 0; j < 3; j++ {
		chunks = component.mergeChunks(chunks)
	}

	// @TODO 处理文本重叠,感觉没有必要了,还会破坏文本的连续性

	documentChunks := make([]DocumentChunk, 0)
	for i := 0; i < len(chunks); i++ {
		chunk := chunks[i]
		documentChunk := DocumentChunk{}
		documentChunk.Id = FuncGenerateStringID()
		documentChunk.DocumentID = document.Id
		documentChunk.KnowledgeBaseID = document.KnowledgeBaseID
		documentChunk.Markdown = chunk
		documentChunk.CreateTime = document.CreateTime
		documentChunk.UpdateTime = document.UpdateTime
		documentChunk.SortNo = i
		documentChunk.Status = document.Status

		documentChunks = append(documentChunks, documentChunk)
	}

	input["documentChunks"] = documentChunks
	return nil
}

// OpenAIDocumentEmbedder 向量化文档字符串
type OpenAIDocumentEmbedder struct {
	APIKey         string            `json:"api_key,omitempty"`
	Model          string            `json:"model,omitempty"`
	BaseURL        string            `json:"base_url,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	client         *http.Client      `json:"-"`
}

func (component *OpenAIDocumentEmbedder) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 180
	}
	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
	}
	if component.BaseURL == "" {
		component.BaseURL = config.AIBaseURL
	}
	if component.APIKey == "" {
		component.APIKey = config.AIAPIkey
	}
	return nil
}
func (component *OpenAIDocumentEmbedder) Run(ctx context.Context, input map[string]interface{}) error {
	documentChunksObj, has := input["documentChunks"]
	if !has {
		return errors.New(funcT("input['documentChunks'] cannot be empty"))
	}
	documentChunks := documentChunksObj.([]DocumentChunk)
	rs := struct {
		Data []struct {
			Embedding []float64 `json:"embedding,omitempty"`
		} `json:"data,omitempty"`
	}{}
	vecDocumentChunks := make([]VecDocumentChunk, 0)
	for i := 0; i < len(documentChunks); i++ {
		bodyMap := make(map[string]interface{}, 0)
		bodyMap["input"] = documentChunks[i].Markdown
		bodyMap["model"] = component.Model
		bodyMap["encoding_format"] = "float"
		bodyByte, err := httpPostJsonBody(component.client, component.APIKey, component.BaseURL+"/embeddings", component.DefaultHeaders, bodyMap)

		if err != nil {
			input[errorKey] = err
			return err
		}

		err = json.Unmarshal(bodyByte, &rs)
		if err != nil {
			input[errorKey] = err
			return err
		}
		if len(rs.Data) < 1 {
			err := errors.New("httpPostJsonBody data is empty")
			input[errorKey] = err
			return err
		}
		embedding, err := vecSerializeFloat64(rs.Data[0].Embedding)
		if err != nil {
			input[errorKey] = err
			return err
		}
		documentChunks[i].Embedding = embedding

		vecdc := VecDocumentChunk{}
		vecdc.Id = documentChunks[i].Id
		vecdc.DocumentID = documentChunks[i].DocumentID
		vecdc.KnowledgeBaseID = documentChunks[i].KnowledgeBaseID
		vecdc.SortNo = documentChunks[i].SortNo
		vecdc.Status = 2
		vecdc.Embedding = embedding

		vecDocumentChunks = append(vecDocumentChunks, vecdc)
	}
	input["documentChunks"] = documentChunks
	input["vecDocumentChunks"] = vecDocumentChunks

	return nil
}

// SQLiteVecDocumentStore 更新文档和向量
type SQLiteVecDocumentStore struct {
}

func (component *SQLiteVecDocumentStore) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *SQLiteVecDocumentStore) Run(ctx context.Context, input map[string]interface{}) error {
	documentObj, has := input["document"]
	if !has {
		err := errors.New(funcT("The document of SQLiteVecDocumentStore cannot be empty"))
		input[errorKey] = err
		return err
	}
	document := documentObj.(*Document)

	documentChunksObj, has := input["documentChunks"]
	var documentChunks []DocumentChunk
	if has {
		documentChunks = documentChunksObj.([]DocumentChunk)
	}

	var vecDocumentChunks []VecDocumentChunk
	vecDocumentChunksObj, has := input["vecDocumentChunks"]
	if has {
		vecDocumentChunks = vecDocumentChunksObj.([]VecDocumentChunk)
	}

	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//先删除,重新插入
		zorm.Delete(ctx, document)
		document.Status = 1
		zorm.Insert(ctx, document)
		// 删除关联的数据,重新插入
		finderDeleteChunk := zorm.NewDeleteFinder(tableDocumentChunkName).Append("WHERE documentID=?", document.Id)
		count, err := zorm.UpdateFinder(ctx, finderDeleteChunk)
		if err != nil {
			return count, err
		}
		finderDeleteVec := zorm.NewDeleteFinder(tableVecDocumentChunkName).Append("WHERE documentID=?", document.Id)
		count, err = zorm.UpdateFinder(ctx, finderDeleteVec)
		if err != nil {
			return count, err
		}

		dcs := make([]zorm.IEntityStruct, 0)
		vecdcs := make([]zorm.IEntityStruct, 0)
		for i := 0; i < len(documentChunks); i++ {
			documentChunks[i].Status = 1
			dcs = append(dcs, &documentChunks[i])
			if len(vecDocumentChunks) < 1 {
				continue
			}
			vecDocumentChunks[i].Status = 1
			vecdcs = append(vecdcs, &vecDocumentChunks[i])
		}
		if len(dcs) > 0 {
			count, err = zorm.InsertSlice(ctx, dcs)
			if err != nil {
				return count, err
			}
		}
		if len(vecdcs) > 0 {
			count, err = zorm.InsertSlice(ctx, vecdcs)
			if err != nil {
				return count, err
			}
		}

		return nil, nil
	})

	if err != nil {
		input[errorKey] = err
	}

	return err
}

// OpenAITextEmbedder 向量化字符串文本
type OpenAITextEmbedder struct {
	APIKey         string            `json:"api_key,omitempty"`
	Model          string            `json:"model,omitempty"`
	BaseURL        string            `json:"base_url,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	client         *http.Client      `json:"-"`
}

func (component *OpenAITextEmbedder) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 180
	}
	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
	}
	if component.BaseURL == "" {
		component.BaseURL = config.AIBaseURL
	}
	if component.APIKey == "" {
		component.APIKey = config.AIAPIkey
	}
	return nil
}
func (component *OpenAITextEmbedder) Run(ctx context.Context, input map[string]interface{}) error {
	queryObj, has := input["query"]
	if !has {
		return errors.New(funcT("input['query'] cannot be empty"))
	}
	bodyMap := make(map[string]interface{}, 0)
	bodyMap["input"] = queryObj.(string)
	bodyMap["model"] = component.Model
	bodyMap["encoding_format"] = "float"
	bodyByte, err := httpPostJsonBody(component.client, component.APIKey, component.BaseURL+"/embeddings", component.DefaultHeaders, bodyMap)
	if err != nil {
		input[errorKey] = err
		return err
	}
	rs := struct {
		Data []struct {
			Embedding []float64 `json:"embedding,omitempty"`
		} `json:"data,omitempty"`
	}{}
	err = json.Unmarshal(bodyByte, &rs)
	if err != nil {
		input[errorKey] = err
		return err
	}
	if len(rs.Data) < 1 {
		err := errors.New("httpPostJsonBody data is empty")
		input[errorKey] = err
		return err
	}
	input["embedding"] = rs.Data[0].Embedding
	return nil
}

// VecEmbeddingRetriever 使用SQLite-Vec向量检索相似数据
type VecEmbeddingRetriever struct {
	// DocumentID 文档ID
	DocumentID string `json:"documentID,omitempty"`
	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `json:"knowledgeBaseID,omitempty"`
	// Embedding 需要查询的向量化数组
	Embedding []float64 `json:"embedding,omitempty"`
	// TopK 检索多少条
	TopK int `json:"topK,omitempty"`
	// Score 向量表的score匹配分数
	Score float32 `json:"score,omitempty"`
}

func (component *VecEmbeddingRetriever) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *VecEmbeddingRetriever) Run(ctx context.Context, input map[string]interface{}) error {
	documentID := ""
	knowledgeBaseID := ""
	topK := 0
	var score float32 = 0.0
	var embedding []float64 = nil
	eId, has := input["embedding"]
	if has {
		embedding = eId.([]float64)
	}
	if embedding == nil {
		embedding = component.Embedding
	}
	if embedding == nil {
		err := errors.New(funcT("The embedding of VecEmbeddingRetriever cannot be empty"))
		input[errorKey] = err
		return err
	}
	dId, has := input["documentID"]
	if has {
		documentID = dId.(string)
	}
	if documentID == "" {
		documentID = component.DocumentID
	}
	kId, has := input["knowledgeBaseID"]
	if has {
		knowledgeBaseID = kId.(string)
	}
	if knowledgeBaseID == "" {
		knowledgeBaseID = component.KnowledgeBaseID
	}
	tId, has := input["topK"]
	if has {
		topK = tId.(int)
	}
	if topK == 0 {
		topK = component.TopK
	}
	if topK == 0 {
		topK = 5
	}
	disId, has := input["score"]
	if has {
		score = disId.(float32)
	}
	if score <= 0 {
		score = component.Score
	}

	query, err := vecSerializeFloat64(embedding)
	if err != nil {
		input[errorKey] = err
		return err
	}
	finder := zorm.NewSelectFinder(tableVecDocumentChunkName, "rowid,distance as score,*").Append("WHERE embedding MATCH ?", query)
	if documentID != "" {
		finder.Append(" and documentID=?", documentID)
	}

	if knowledgeBaseID != "" {
		// Only one of EQUALS, GREATER_THAN, LESS_THAN_OR_EQUAL, LESS_THAN, GREATER_THAN_OR_EQUAL, NOT_EQUALS is allowed
		// vec不支持 like
		//finder.Append(" and knowledgeBaseID like ?", knowledgeBaseID+"%")
		finder.Append(" and knowledgeBaseID = ?", knowledgeBaseID)
	}
	// Only one of EQUALS, GREATER_THAN, LESS_THAN_OR_EQUAL, LESS_THAN, GREATER_THAN_OR_EQUAL, NOT_EQUALS is allowed
	// vec不支持 范围查询
	//if score > 0.0 {
	//	finder.Append(" and score >= ?", score)
	//}
	finder.Append("ORDER BY score LIMIT " + strconv.Itoa(topK))
	documentChunks := make([]DocumentChunk, 0)
	err = zorm.Query(ctx, finder, &documentChunks, nil)
	if err != nil {
		input[errorKey] = err
		return err
	}
	//更新markdown内容
	documentChunks, err = findDocumentChunkMarkDown(ctx, documentChunks)
	if err != nil {
		input[errorKey] = err
		return err
	}

	//重新排序
	documentChunks = sortDocumentChunksScore(documentChunks, topK, score)

	oldDcs, has := input["documentChunks"]
	if has && oldDcs != nil {
		oldDocumentChunks := oldDcs.([]DocumentChunk)
		documentChunks = append(oldDocumentChunks, documentChunks...)
	}

	input["documentChunks"] = documentChunks
	return nil
}

// FtsKeywordRetriever 使用Fts5全文检索相似数据
type FtsKeywordRetriever struct {
	// DocumentID 文档ID
	DocumentID string `json:"documentID,omitempty"`
	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `json:"knowledgeBaseID,omitempty"`
	// Query 需要查询的关键字
	Query string `json:"query,omitempty"`
	// TopK 检索多少条
	TopK int `json:"topK,omitempty"`
	// Score BM25的FTS5实现在返回结果之前将结果乘以-1,得分越小(数值上更负),表示匹配越好
	Score float32 `json:"score,omitempty"`
}

func (component *FtsKeywordRetriever) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *FtsKeywordRetriever) Run(ctx context.Context, input map[string]interface{}) error {
	documentID := ""
	knowledgeBaseID := ""
	topK := 0
	query := ""
	var score float32 = 0.0
	qId, has := input["query"]
	if has {
		query = qId.(string)
	}
	if query == "" {
		query = component.Query
	}
	if query == "" {
		err := errors.New(funcT("The query of FtsKeywordRetriever cannot be empty"))
		input[errorKey] = err
		return err
	}
	dId, has := input["documentID"]
	if has {
		documentID = dId.(string)
	}
	if documentID == "" {
		documentID = component.DocumentID
	}
	kId, has := input["knowledgeBaseID"]
	if has {
		knowledgeBaseID = kId.(string)
	}
	if knowledgeBaseID == "" {
		knowledgeBaseID = component.KnowledgeBaseID
	}
	tId, has := input["topK"]
	if has {
		topK = tId.(int)
	}
	if topK == 0 {
		topK = component.TopK
	}
	if topK == 0 {
		topK = 5
	}
	disId, has := input["score"]
	if has {
		score = disId.(float32)
	}
	if score <= 0 {
		score = component.Score
	}
	// BM25的FTS5实现在返回结果之前将结果乘以-1,得分越小(数值上更负),表示匹配越好
	finder := zorm.NewFinder().Append("SELECT rowid,-1*rank as score,* from fts_document_chunk where fts_document_chunk match jieba_query(?)", query)
	if documentID != "" {
		finder.Append(" and documentID=?", documentID)
	}
	if knowledgeBaseID != "" {
		finder.Append(" and knowledgeBaseID like ?", knowledgeBaseID+"%")
	}
	if score > 0.0 { // BM25的FTS5实现在返回结果之前将结果乘以-1,查询时再乘以-1
		finder.Append(" and score >= ?", score)
	}
	finder.Append("ORDER BY score DESC LIMIT " + strconv.Itoa(topK))
	documentChunks := make([]DocumentChunk, 0)
	err := zorm.Query(ctx, finder, &documentChunks, nil)
	if err != nil {
		input[errorKey] = err
		return err
	}

	oldDcs, has := input["documentChunks"]
	if has && oldDcs != nil {
		oldDocumentChunks := oldDcs.([]DocumentChunk)
		documentChunks = append(oldDocumentChunks, documentChunks...)
	}
	input["documentChunks"] = documentChunks
	return nil
}

// DocumentChunkReranker 对DocumentChunks进行重新排序
type DocumentChunkReranker struct {
	APIKey         string            `json:"api_key,omitempty"`
	Model          string            `json:"model,omitempty"`
	BaseURL        string            `json:"base_url,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	// Query 需要查询的关键字
	Query string `json:"query,omitempty"`
	// TopK 检索多少条
	TopK int `json:"topK,omitempty"`
	// Score ranker的score匹配分数
	Score  float32      `json:"score,omitempty"`
	client *http.Client `json:"-"`
}

func (component *DocumentChunkReranker) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 180
	}

	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
	}

	if component.APIKey == "" {
		component.APIKey = config.AIAPIkey
	}
	if component.BaseURL == "" {
		if config.AIBaseURL == "" {
			return nil
		}
		index := strings.Index(config.AIBaseURL, "/v1")
		if index <= 0 {
			return nil
		}
		component.BaseURL = config.AIBaseURL[:index] + "/api/serverless/bge-reranker-v2-m3/rerank"
	}
	return nil
}
func (component *DocumentChunkReranker) Run(ctx context.Context, input map[string]interface{}) error {
	topK := 0
	var score float32 = 0.0
	dcs, has := input["documentChunks"]
	if !has || dcs == nil {
		err := errors.New(funcT("input['documentChunks'] cannot be empty"))
		input[errorKey] = err
		return err
	}
	queryObj, has := input["query"]
	if !has {
		return errors.New(funcT("input['query'] cannot be empty"))
	}
	query := queryObj.(string)
	if query == "" {
		return errors.New(funcT("input['query'] cannot be empty"))
	}

	tId, has := input["topK"]
	if has {
		topK = tId.(int)
	}
	if topK == 0 {
		topK = component.TopK
	}
	if topK == 0 {
		topK = 5
	}
	disId, has := input["score"]
	if has {
		score = disId.(float32)
	}
	if score <= 0 {
		score = component.Score
	}

	documentChunks := dcs.([]DocumentChunk)
	if len(documentChunks) < 1 { //没有文档,不需要重排
		return nil
	}
	documents := make([]string, 0)
	for i := 0; i < len(documentChunks); i++ {
		documents = append(documents, documentChunks[i].Markdown)
	}

	bodyMap := map[string]interface{}{
		"model":     component.Model,
		"query":     query,
		"top_n":     topK,
		"documents": documents,
	}

	rsStringByte, err := httpPostJsonBody(component.client, component.APIKey, component.BaseURL, component.DefaultHeaders, bodyMap)
	if err != nil {
		input[errorKey] = err
		return err
	}

	rs := struct {
		Results []struct {
			Document struct {
				Text string `json:"text,omitempty"`
			} `json:"document,omitempty"`
			RelevanceScore float32 `json:"relevance_score,omitempty"`
		} `json:"results,omitempty"`
	}{}

	err = json.Unmarshal(rsStringByte, &rs)
	if err != nil {
		input[errorKey] = err
		return err
	}
	rerankerDCS := make([]DocumentChunk, 0)
	for i := 0; i < len(rs.Results); i++ {
		markdown := rs.Results[i].Document.Text
		for j := 0; j < len(documentChunks); j++ {
			dc := documentChunks[j]
			if markdown == dc.Markdown { //相等
				dc.Score = rs.Results[i].RelevanceScore
				rerankerDCS = append(rerankerDCS, dc)
				break
			}
		}
	}
	rerankerDCS = sortDocumentChunksScore(rerankerDCS, topK, score)
	input["documentChunks"] = rerankerDCS
	return nil
}

func sortDocumentChunksScore(documentChunks []DocumentChunk, topK int, score float32) []DocumentChunk {
	sort.Slice(documentChunks, func(i, j int) bool {
		return documentChunks[i].Score > documentChunks[j].Score
	})

	resultDCS := make([]DocumentChunk, 0)
	for i := 0; i < len(documentChunks); i++ {
		documentChunk := documentChunks[i]
		if len(resultDCS) >= topK {
			break
		}
		if documentChunk.Score < score {
			continue
		}
		resultDCS = append(resultDCS, documentChunk)
	}
	return resultDCS
}

// PromptBuilder 使用模板构建Prompt提示词
type PromptBuilder struct {
	PromptTemplate string             `json:"promptTemplate,omitempty"`
	t              *template.Template `json:"-"`
}

func (component *PromptBuilder) Initialization(ctx context.Context, input map[string]interface{}) error {
	var err error
	tmpl := template.New("minrag-promptBuilder")
	component.t, err = tmpl.Parse(component.PromptTemplate)
	if err != nil {
		return err
	}
	return nil
}
func (component *PromptBuilder) Run(ctx context.Context, input map[string]interface{}) error {
	_, has := input[ifEmptyStop]
	if has {
		dcs, hasdcs := input["documentChunks"]
		if !hasdcs || dcs == nil {
			input[endKey] = true
			return nil
		}
		documentChunks := dcs.([]DocumentChunk)
		if len(documentChunks) < 1 {
			input[endKey] = true
			return nil
		}
	}

	// 创建一个 bytes.Buffer 用于存储渲染后的 text 内容
	var buf bytes.Buffer
	// 执行模板并将结果写入到 bytes.Buffer
	if err := component.t.Execute(&buf, input); err != nil {
		input[errorKey] = err
		return err
	}

	// 获取编译后的内容
	input["prompt"] = buf.String()

	return nil
}

// OpenAIChatMemory 上下文记忆聊天记录
type OpenAIChatMemory struct {
	// 上下文记忆长度
	MemoryLength int `json:"memoryLength,omitempty"`
}

func (component *OpenAIChatMemory) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *OpenAIChatMemory) Run(ctx context.Context, input map[string]interface{}) error {
	prompt, has := input["prompt"]
	if !has {
		err := errors.New(funcT("input['prompt'] cannot be empty"))
		input[errorKey] = err
		return err
	}
	messages := make([]ChatMessage, 0)
	ms, has := input["messages"]
	if has {
		messages = ms.([]ChatMessage)
	}
	agentID, has := input["agentID"]
	if has {
		agent, err := findAgentByID(ctx, agentID.(string))
		if err != nil {
			input[errorKey] = err
			return err
		}
		agentPrompt := ChatMessage{Role: "system", Content: agent.AgentPrompt}
		messages = append(messages, agentPrompt)
		//tools
		if len(agent.Tools) > 0 {
			toolSlice := make([]string, 0)
			json.Unmarshal([]byte(agent.Tools), &toolSlice)
			tools := make([]interface{}, 0)
			for i := 0; i < len(toolSlice); i++ {
				toolName := toolSlice[i]
				fc, has := functionCallingMap[toolName]
				if has {
					tools = append(tools, fc.Description(ctx))
				}
			}
			if len(tools) > 0 {
				input["tools"] = tools
			}

		}
	}

	roomIDObj, has := input["roomID"]
	roomID := ""
	if has && roomIDObj != nil {
		roomID = roomIDObj.(string)
	}
	messageLogs := make([]MessageLog, 0)
	if roomID != "" && component.MemoryLength > 0 {
		finder := zorm.NewSelectFinder(tableMessageLogName).Append("WHERE roomID=? order by createTime desc", roomID)
		finder.SelectTotalCount = false
		page := zorm.NewPage()
		page.PageSize = component.MemoryLength
		zorm.Query(ctx, finder, &messageLogs, page)
	}
	for i := len(messageLogs) - 1; i >= 0; i-- {
		messageLog := messageLogs[i]
		if messageLog.UserMessage != "" {
			messages = append(messages, ChatMessage{Role: "user", Content: messageLog.UserMessage})
		}
		if messageLog.AIMessage != "" {
			messages = append(messages, ChatMessage{Role: "assistant", Content: messageLog.AIMessage})
		}

	}

	promptMessage := ChatMessage{Role: "user", Content: prompt.(string)}
	messages = append(messages, promptMessage)
	input["messages"] = messages

	return nil
}

type Choice struct {
	FinishReason string      `json:"finish_reason,omitempty"`
	Index        int         `json:"index,omitempty"`
	Message      ChatMessage `json:"message,omitempty"`
	Delta        ChatMessage `json:"delta,omitempty"`
}

type ChatMessage struct {
	Role       string     `json:"role,omitempty"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}
type ToolCall struct {
	Id         string       `json:"id,omitempty"`
	Index      int          `json:"index,omitempty"`
	Type       string       `json:"type,omitempty"`
	ToolCallId string       `json:"tool_call_id,omitempty"`
	Function   ChatFunction `json:"function,omitempty"`
}
type ChatFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// OpenAIChatGenerator OpenAI的LLM大语言模型
type OpenAIChatGenerator struct {
	APIKey         string            `json:"api_key,omitempty"`
	Model          string            `json:"model,omitempty"`
	BaseURL        string            `json:"base_url,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	Temperature    float32           `json:"temperature,omitempty"`
	Stream         *bool             `json:"stream,omitempty"`
	//MaxCompletionTokens int64             `json:"maxCompletionTokens,omitempty"`
	client *http.Client `json:"-"`
}

func (component *OpenAIChatGenerator) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 180
	}

	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
	}
	if component.BaseURL == "" {
		component.BaseURL = config.AIBaseURL
	}
	if component.APIKey == "" {
		component.APIKey = config.AIAPIkey
	}
	return nil
}
func (component *OpenAIChatGenerator) Run(ctx context.Context, input map[string]interface{}) error {
	var messages []ChatMessage
	ms, has := input["messages"]

	if !has {
		err := errors.New(funcT("input['messages'] cannot be empty"))
		input[errorKey] = err
		return err
	}
	messages = ms.([]ChatMessage)
	bodyMap := make(map[string]interface{})
	bodyMap["messages"] = messages
	bodyMap["model"] = component.Model
	if component.Temperature != 0 {
		bodyMap["temperature"] = component.Temperature
	}

	tools, has := input["tools"]
	if has {
		bodyMap["tools"] = tools
	}

	var c *app.RequestContext
	cObj, has := input["c"]
	if has {
		c = cObj.(*app.RequestContext)
	}

	url := component.BaseURL + "/chat/completions"
	stream := true
	// 如果没有设置,根据请求类型,自动获取是否流式输出
	if component.Stream == nil && c != nil {
		accept := string(c.GetHeader("Accept"))
		stream = strings.Contains(strings.ToLower(accept), "text/event-stream")
	} else {
		stream = *component.Stream
	}
	//输出类型
	bodyMap["stream"] = stream

	if !stream { //一次性输出,不是流式输出
		//请求大模型
		bodyByte, err := httpPostJsonBody(component.client, component.APIKey, url, component.DefaultHeaders, bodyMap)
		if err != nil {
			input[errorKey] = err
			return err
		}
		rs := struct {
			Choices []Choice `json:"choices,omitempty"`
		}{}
		err = json.Unmarshal(bodyByte, &rs)
		if err != nil {
			input[errorKey] = err
			return err
		}
		if len(rs.Choices) < 1 {
			err := errors.New("httpPostJsonBody choices is empty")
			input[errorKey] = err
			return err
		}
		//获取第一个结果
		choice := rs.Choices[0]
		//没有函数调用,把模型返回的choice放入到input["choice"],并输出
		if len(choice.Message.ToolCalls) == 0 {
			input["choice"] = choice
			c.WriteString(choice.Message.Content)
			c.Flush()
			return nil
		}

		//遍历所有的函数,追加到messages列表
		for i := 0; i < len(choice.Message.ToolCalls); i++ {
			tc := choice.Message.ToolCalls[i]
			funcName := tc.Function.Name
			//获取函数的实现对象
			fc, has := functionCallingMap[funcName]
			if !has {
				continue
			}
			//执行函数
			content, err := fc.Run(ctx, tc.Function.Arguments)
			if err != nil {
				continue
			}
			//将函数执行的结果和tool_call_id追加到messages列表
			messages = append(messages, ChatMessage{Role: "tool", ToolCallID: tc.Id, Content: content})
		}

		//重新放入 input["messages"]
		input["messages"] = messages
		//删除掉input中的tools,避免再次调用函数,造成递归死循环.就是带着函数结果请求大模型,结果大模型又返回调用函数,造成死循环.
		delete(input, "tools")
		//重新运行组件,调用大模型.
		component.Run(ctx, input)

		return nil
	}

	// toolCalls 需要调用的函数列表,如果有值,说明需要调用函数,不能直接返回结果
	var toolCalls []ToolCall

	//设置SSE的协议头
	component.DefaultHeaders["Accept"] = "text/event-stream"
	component.DefaultHeaders["Cache-Control"] = "no-cache"
	component.DefaultHeaders["Connection"] = "keep-alive"

	//请求大模型
	resp, err := httpPostJsonResponse(component.client, component.APIKey, url, component.DefaultHeaders, bodyMap)
	if err != nil {
		input[errorKey] = err
		return err
	}
	defer resp.Body.Close()
	//用于拼接stream返回的最终结果
	choice := Choice{FinishReason: "stop"}
	var message strings.Builder
	// 使用 bufio.NewReader 逐行读取响应体
	reader := bufio.NewReader(resp.Body)
	//循环处理stream流输出
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			input[errorKey] = err
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// SSE的标准格式是 data:空格 开头,后面跟着数据
		if !strings.HasPrefix(line, "data: ") {
			err := errors.New("stream data format is error")
			input[errorKey] = err
			return err
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" { //结束符
			// 没有需要调用的函数,就输出结束 DONE
			if c != nil && len(toolCalls) == 0 {
				c.WriteString("data: [DONE]\n\n")
				c.Flush()
			}
			break
		}
		if data == "" {
			continue
		}
		rs := struct {
			Choices []Choice `json:"choices,omitempty"`
		}{}
		//大模型返回的json对象,进行接受
		err = json.Unmarshal([]byte(data), &rs)
		if err != nil {
			input[errorKey] = err
			return err
		}
		if len(rs.Choices) < 1 {
			err := errors.New("httpPostJsonResponse choices is empty")
			input[errorKey] = err
			return err
		}
		//FinishReason结束符
		if rs.Choices[0].FinishReason != "" {
			choice.FinishReason = rs.Choices[0].FinishReason
		}

		// 模型返回的调用函数,有可能是多个函数
		tcLen := len(rs.Choices[0].Delta.ToolCalls)
		// 模型返回的有函数调用,初始化toolCalls,长度一致
		if tcLen > 0 && len(toolCalls) == 0 {
			toolCalls = make([]ToolCall, tcLen)
		}
		//stream会把函数参数片段输出,需要重新拼接为完整的函数信息
		for i := 0; i < tcLen; i++ {
			tc := rs.Choices[0].Delta.ToolCalls[i]
			if tc.Id != "" { //tool_call_id 不会分段
				toolCalls[tc.Index].Id = tc.Id
			}
			if tc.Function.Name != "" { //tool_call_name 函数名称,不会分段
				toolCalls[tc.Index].Function.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" { //函数的参数,会分段,所以循环拼接起来
				toolCalls[tc.Index].Function.Arguments += tc.Function.Arguments
			}
		}

		// 不是函数调用,把返回的内容输出到页面
		if c != nil && len(toolCalls) == 0 {
			c.WriteString("data: " + rs.Choices[0].Delta.Content + "\n\n")
			c.Flush()
		}
		// 不是函数调用,把内容拼接起来
		if len(toolCalls) == 0 {
			message.WriteString(rs.Choices[0].Delta.Content)
		}
	}
	//没有函数调用,把模型返回的choice放入到input["choice"]
	if len(toolCalls) == 0 {
		choice.Message = ChatMessage{Role: "assistant", Content: message.String()}
		input["choice"] = choice
		return nil
	}
	//遍历所有的函数,追加到messages列表
	for i := 0; i < len(toolCalls); i++ {
		tc := toolCalls[i]
		funcName := tc.Function.Name
		//获取函数的实现对象
		fc, has := functionCallingMap[funcName]
		if !has {
			continue
		}
		//执行函数
		content, err := fc.Run(ctx, tc.Function.Arguments)
		if err != nil {
			continue
		}
		//将函数执行的结果和tool_call_id追加到messages列表
		messages = append(messages, ChatMessage{Role: "tool", ToolCallID: tc.Id, Content: content})
	}
	//重新放入 input["messages"]
	input["messages"] = messages
	//删除掉input中的tools,避免再次调用函数,造成递归死循环.就是带着函数结果请求大模型,结果大模型又返回调用函数,造成死循环.
	delete(input, "tools")
	//重新运行组件,调用大模型.
	component.Run(ctx, input)

	return nil

}

// ChatMessageLogStore 保存消息记录到数据库
type ChatMessageLogStore struct {
}

func (component *ChatMessageLogStore) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}

func (component *ChatMessageLogStore) Run(ctx context.Context, input map[string]interface{}) error {
	cObj, has := input["c"]
	if !has || cObj == nil {
		return errors.New(`input["c"] is nil`)
	}
	c := cObj.(*app.RequestContext)

	roomIDObj, has := input["roomID"]
	if !has || roomIDObj == nil {
		return errors.New(`input["roomID"] is nil`)
	}
	roomID := roomIDObj.(string)

	agentIDObj, has := input["agentID"]
	if !has || agentIDObj == nil {
		return errors.New(`input["agentID"] is nil`)
	}
	agentID := agentIDObj.(string)

	queryObj, has := input["query"]
	if !has || queryObj == nil {
		return errors.New(`input["query"] is nil`)
	}
	query := queryObj.(string)

	agent, err := findAgentByID(ctx, agentID)
	if err != nil {
		return err
	}

	choice := Choice{}
	choiceObj, has := input["choice"]
	if has && choiceObj != nil {
		choice = choiceObj.(Choice)
	}

	jwttoken := string(c.Cookie(config.JwttokenKey))
	userId, _ := userIdByToken(jwttoken)

	now := time.Now().Format("2006-01-02 15:04:05")

	messageLog := &MessageLog{}
	messageLog.Id = FuncGenerateStringID()
	messageLog.CreateTime = now
	messageLog.RoomID = roomID
	messageLog.KnowledgeBaseID = agent.KnowledgeBaseID
	messageLog.AgentID = agentID
	messageLog.PipelineID = agent.PipelineID
	messageLog.UserID = userId
	messageLog.UserMessage = query
	messageLog.AIMessage = choice.Message.Content

	finder := zorm.NewSelectFinder(tableChatRoomName).Append("WHERE id=?", roomID)
	chatRoom := &ChatRoom{}
	zorm.QueryRow(ctx, finder, chatRoom)
	chatRoom.CreateTime = now
	chatRoom.KnowledgeBaseID = agent.KnowledgeBaseID
	chatRoom.AgentID = agentID
	chatRoom.PipelineID = agent.PipelineID
	chatRoom.UserID = userId
	if chatRoom.Name == "" {
		qLen := len(query)
		if qLen > 20 {
			qLen = 20
		}
		chatRoom.Name = query[:qLen]
	}

	_, err = zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		if chatRoom.Id == "" {
			chatRoom.Id = messageLog.RoomID
			count, err := zorm.Insert(ctx, chatRoom)
			if err != nil {
				return count, err
			}
		}
		count, err := zorm.Insert(ctx, messageLog)
		if err != nil {
			return count, err
		}
		return nil, nil
	})
	return err
}
