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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const (
	errorKey         string = "__error__"
	nextComponentKey string = "__next__"
	endKey           string = "__end__"
)

// componentTypeMap 组件类型对照,key是类型名称,value是组件实例
var componentTypeMap = map[string]IComponent{
	"OpenAIChatCompletion":    &OpenAIChatCompletion{},
	"OpenAIChatMessageMemory": &OpenAIChatMessageMemory{},
	"PromptBuilder":           &PromptBuilder{},
	"DocumentChunksReranker":  &DocumentChunksReranker{},
	"DocumentSplitter":        &DocumentSplitter{},
	"OpenAITextEmbedder":      &OpenAITextEmbedder{},
	"VecEmbeddingRetriever":   &VecEmbeddingRetriever{},
	"FtsKeywordRetriever":     &FtsKeywordRetriever{},
}

// componentMap 组件的Map,从数据查询拼装参数
var componentMap = make(map[string]IComponent, 0)

// IComponent 组件的接口
type IComponent interface {
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
	Start   string            `json:"start,omitempty"`
	Process map[string]string `json:"process,omitempty"`
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
	APIBaseURL     string            `json:"apiBaseURL,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	client         *openai.Client    `json:"-"`
}

func (component *OpenAITextEmbedder) Run(ctx context.Context, input map[string]interface{}) error {
	if component.client == nil {
		if component.Timeout == 0 {
			component.Timeout = 60
		}
		component.client = openai.NewClient(
			option.WithAPIKey(component.APIKey),
			option.WithBaseURL(component.APIBaseURL),
			option.WithMaxRetries(component.MaxRetries),
			option.WithRequestTimeout(time.Second*time.Duration(component.Timeout)),
		)
	}

	headerOpention := make([]option.RequestOption, 0)
	if len(component.DefaultHeaders) > 0 {
		for k, v := range component.DefaultHeaders {
			headerOpention = append(headerOpention, option.WithHeader(k, v))
		}
	}
	query := input["query"].(string)
	response, err := component.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model:          openai.F(component.Model),
		EncodingFormat: openai.F(openai.EmbeddingNewParamsEncodingFormatFloat),
		Input:          openai.F[openai.EmbeddingNewParamsInputUnion](shared.UnionString(query))}, headerOpention...)
	if err != nil {
		input[errorKey] = err
		return err
	}
	input["embedding"] = response.Data[0].Embedding
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
		finder.Append(" and knowledgeBaseID like ?", knowledgeBaseID+"%")
	}
	if score > 0.0 {
		finder.Append(" and score >= ?", score)
	}
	finder.Append("ORDER BY score LIMIT " + strconv.Itoa(topK))
	documentChunks := make([]DocumentChunk, 0)
	err = zorm.Query(ctx, finder, &documentChunks, nil)
	if err != nil {
		input[errorKey] = err
		return err
	}
	//更新markdown内容
	findDocumentChunkMarkDown(ctx, documentChunks)
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
	finder := zorm.NewFinder().Append("SELECT rowid,rank as score,* from fts_document_chunk where fts_document_chunk match jieba_query(?)", query)
	if documentID != "" {
		finder.Append(" and documentID=?", documentID)
	}
	if knowledgeBaseID != "" {
		finder.Append(" and knowledgeBaseID like ?", knowledgeBaseID+"%")
	}
	if score > 0.0 { // BM25的FTS5实现在返回结果之前将结果乘以-1,得分越小(数值上更负),表示匹配越好
		finder.Append(" and score <= ?", 0-score)
	}
	finder.Append("ORDER BY score LIMIT " + strconv.Itoa(topK))
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
	APIBaseURL     string            `json:"apiBaseURL,omitempty"`
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

func (component *DocumentChunksReranker) Run(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 60
	}
	if component.client == nil {
		component.client = &http.Client{
			Timeout: time.Second * time.Duration(component.Timeout),
		}
	}
	dcs, has := input["documentChunks"]
	if !has || dcs == nil {
		err := errors.New(funcT("input['documentChunks'] cannot be empty"))
		input[errorKey] = err
		return err
	}
	documentChunks := dcs.([]DocumentChunk)
	documents := make([]string, 0)
	for i := 0; i < len(documentChunks); i++ {
		documents = append(documents, documentChunks[i].Markdown)
	}
	bodyMap := map[string]interface{}{
		"model":     component.Model,
		"query":     component.Query,
		"top_n":     component.TopK,
		"documents": documents,
	}
	// 序列化请求体
	payloadBytes, err := json.Marshal(bodyMap)
	if err != nil {
		input[errorKey] = err
		return err
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", component.APIBaseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		input[errorKey] = err
		return err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+component.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if len(component.DefaultHeaders) > 0 {
		for k, v := range component.DefaultHeaders {
			req.Header.Set(k, v)
		}
	}
	resp, err := component.client.Do(req)
	if err != nil {
		input[errorKey] = err
		return err
	}
	defer resp.Body.Close()
	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		err := errors.New("DocumentChunksReranker http post error")
		input[errorKey] = err
		return err
	}

	// 读取响应体内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		input[errorKey] = err
		return err
	}

	// 将 JSON 数据解析为 map[string]interface{}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		input[errorKey] = err
		return err
	}

	fmt.Println(result["results"])

	return nil
}

// PromptBuilder 使用模板构建Prompt提示词
type PromptBuilder struct {
	PromptTemplate string             `json:"promptTemplate,omitempty"`
	t              *template.Template `json:"-"`
}

func (component *PromptBuilder) Run(ctx context.Context, input map[string]interface{}) error {
	if component.t == nil {
		var err error
		tmpl := template.New("minrag-promptBuilder")
		component.t, err = tmpl.Parse(component.PromptTemplate)
		if err != nil {
			input[errorKey] = err
			return err
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

// OpenAIChatMessageMemory 上下文记忆聊天记录
type OpenAIChatMessageMemory struct {
	// 上下文记忆长度
	MemoryLength int `json:"memoryLength,omitempty"`
}

func (component *OpenAIChatMessageMemory) Run(ctx context.Context, input map[string]interface{}) error {
	prompt, has := input["prompt"]
	if !has {
		err := errors.New(funcT("input['prompt'] cannot be empty"))
		input[errorKey] = err
		return err
	}
	messages := make([]openai.ChatCompletionMessageParamUnion, 0)
	ms, has := input["messages"]
	if has {
		messages = ms.([]openai.ChatCompletionMessageParamUnion)
	}
	promptMessage := openai.UserMessage(prompt.(string))
	messages = append(messages, promptMessage)
	input["messages"] = messages
	return nil
}

// OpenAIChatCompletion OpenAI的LLM大语言模型
type OpenAIChatCompletion struct {
	APIKey         string            `json:"apikey,omitempty"`
	Model          string            `json:"model,omitempty"`
	APIBaseURL     string            `json:"apiBaseURL,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	Temperature    float32           `json:"temperature,omitempty"`
	Stream         bool              `json:"stream,omitempty"`
	//MaxCompletionTokens int64             `json:"maxCompletionTokens,omitempty"`
	client *openai.Client `json:"-"`
}

func (component *OpenAIChatCompletion) Run(ctx context.Context, input map[string]interface{}) error {
	if component.client == nil {
		if component.Timeout == 0 {
			component.Timeout = 60
		}
		component.client = openai.NewClient(
			option.WithAPIKey(component.APIKey),
			option.WithBaseURL(component.APIBaseURL),
			option.WithMaxRetries(component.MaxRetries),
			option.WithRequestTimeout(time.Second*time.Duration(component.Timeout)),
		)
	}
	headerOpention := make([]option.RequestOption, 0)
	if len(component.DefaultHeaders) > 0 {
		for k, v := range component.DefaultHeaders {
			headerOpention = append(headerOpention, option.WithHeader(k, v))
		}
	}

	var messages []openai.ChatCompletionMessageParamUnion
	ms, has := input["messages"]

	if !has {
		err := errors.New(funcT("input['messages'] cannot be empty"))
		input[errorKey] = err
		return err
	}
	messages = ms.([]openai.ChatCompletionMessageParamUnion)
	chatCompletionNewParams := openai.ChatCompletionNewParams{
		Model:    openai.F(component.Model),
		Messages: openai.F(messages),
	}
	if !component.Stream {
		chatCompletion, err := component.client.Chat.Completions.New(ctx, chatCompletionNewParams)
		if err != nil {
			input[errorKey] = err
			return err
		}
		chatCompletionMessage := chatCompletion.Choices[0].Message
		input["chatCompletionMessage"] = chatCompletionMessage
		fmt.Println(chatCompletionMessage)
		return nil
	}
	stream := component.client.Chat.Completions.NewStreaming(ctx, chatCompletionNewParams)
	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) > 0 {
			print(evt.Choices[0].Delta.Content)
		}
	}
	if err := stream.Err(); err != nil {
		input[errorKey] = err
		return err
	}
	return nil

}

// findAllComponentList 查询所有的组件
func findAllComponentList(ctx context.Context) ([]Component, error) {
	finder := zorm.NewSelectFinder(tableComponentName).Append("order by sortNo desc")
	list := make([]Component, 0)
	err := zorm.Query(ctx, finder, &list, nil)
	return list, err
}
