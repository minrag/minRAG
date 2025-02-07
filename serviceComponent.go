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
)

// componentTypeMap 组件类型对照,key是类型名称,value是组件实例
var componentTypeMap = map[string]IComponent{
	"Pipeline":                &Pipeline{},
	"OpenAIChatCompletion":    &OpenAIChatCompletion{},
	"OpenAIChatMessageMemory": &OpenAIChatMessageMemory{},
	"PromptBuilder":           &PromptBuilder{},
	"DocumentChunksReranker":  &DocumentChunksReranker{},
	"FtsKeywordRetriever":     &FtsKeywordRetriever{},
	"VecEmbeddingRetriever":   &VecEmbeddingRetriever{},
	"OpenAITextEmbedder":      &OpenAITextEmbedder{},
	"DocumentSplitter":        &DocumentSplitter{},
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
	finder := zorm.NewSelectFinder(tableComponentName).Append("WHERE status=1")
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
	BaseURL        string            `json:"baseURL,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	client         *http.Client      `json:"-"`
}

func (component *OpenAITextEmbedder) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 60
	}

	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
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
	//bodyMap["dimensions"] = 1
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
		// 不支持 like
		//finder.Append(" and knowledgeBaseID like ?", knowledgeBaseID+"%")
		finder.Append(" and knowledgeBaseID = ?", knowledgeBaseID)
	}
	// 范围查询
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

// DocumentChunksReranker 对DocumentChunks进行重新排序
type DocumentChunksReranker struct {
	APIKey         string            `json:"apikey,omitempty"`
	Model          string            `json:"model,omitempty"`
	BaseURL        string            `json:"baseURL,omitempty"`
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

func (component *DocumentChunksReranker) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 60
	}

	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
	}

	return nil
}
func (component *DocumentChunksReranker) Run(ctx context.Context, input map[string]interface{}) error {
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

// OpenAIChatMessageMemory 上下文记忆聊天记录
type OpenAIChatMessageMemory struct {
	// 上下文记忆长度
	MemoryLength int `json:"memoryLength,omitempty"`
}

func (component *OpenAIChatMessageMemory) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *OpenAIChatMessageMemory) Run(ctx context.Context, input map[string]interface{}) error {
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
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
type ToolCall struct {
	Id         string       `json:"id,omitempty"`
	Type       string       `json:"type,omitempty"`
	ToolCallId string       `json:"tool_call_id,omitempty"`
	Function   ChatFunction `json:"function,omitempty"`
}
type ChatFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// OpenAIChatCompletion OpenAI的LLM大语言模型
type OpenAIChatCompletion struct {
	APIKey         string            `json:"apikey,omitempty"`
	Model          string            `json:"model,omitempty"`
	BaseURL        string            `json:"baseURL,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	Temperature    float32           `json:"temperature,omitempty"`
	Stream         bool              `json:"stream,omitempty"`
	//MaxCompletionTokens int64             `json:"maxCompletionTokens,omitempty"`
	client *http.Client `json:"-"`
}

func (component *OpenAIChatCompletion) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 60
	}

	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
	}

	return nil
}
func (component *OpenAIChatCompletion) Run(ctx context.Context, input map[string]interface{}) error {
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
	bodyMap["stream"] = component.Stream
	url := component.BaseURL + "/chat/completions"
	if !component.Stream {
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
		input["choice"] = rs.Choices[0]
		return nil
	}
	component.DefaultHeaders["Accept"] = "text/event-stream"
	component.DefaultHeaders["Cache-Control"] = "no-cache"
	component.DefaultHeaders["Connection"] = "keep-alive"
	resp, err := httpPostJsonResponse(component.client, component.APIKey, url, component.DefaultHeaders, bodyMap)
	if err != nil {
		input[errorKey] = err
		return err
	}
	defer resp.Body.Close()

	var c *app.RequestContext
	cObj, has := input["c"]
	if has {
		c = cObj.(*app.RequestContext)
	}

	choice := Choice{FinishReason: "stop"}
	var message strings.Builder
	// 使用 bufio.NewReader 逐行读取响应体
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			input[errorKey] = err
			return err
		}

		// 去掉行首的换行符
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			err := errors.New("stream data format is error")
			input[errorKey] = err
			return err
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			// TODO 需要输出到前端页面
			if c != nil {
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
		if rs.Choices[0].FinishReason != "" {
			choice.FinishReason = rs.Choices[0].FinishReason
		}
		// TODO 需要输出到前端页面
		if c != nil {
			c.WriteString("data: " + rs.Choices[0].Delta.Content + "\n\n")
			c.Flush()
		}
		message.WriteString(rs.Choices[0].Delta.Content)
	}
	choice.Message = ChatMessage{Role: "assistant", Content: message.String()}
	input["choice"] = choice
	return nil

}

// findAllComponentList 查询所有的组件
func findAllComponentList(ctx context.Context) ([]Component, error) {
	finder := zorm.NewSelectFinder(tableComponentName).Append("order by sortNo desc")
	list := make([]Component, 0)
	err := zorm.Query(ctx, finder, &list, nil)
	return list, err
}
