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
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/cloudwego/hertz/pkg/app"
	"golang.org/x/net/html"
)

const (
	errorKey         string = "__error__"
	nextComponentKey string = "__next__"
	endKey           string = "__end__"
	ifEmptyStop      string = "__ifEmptyStop__"
)

// componentTypeMap 组件类型对照,key是类型名称,value是组件实例
var componentTypeMap = map[string]IComponent{
	"Pipeline":                     &Pipeline{},
	"ChatMessageLogStore":          &ChatMessageLogStore{},
	"OpenAIChatGenerator":          &OpenAIChatGenerator{},
	"OpenAIChatMemory":             &OpenAIChatMemory{},
	"PromptBuilder":                &PromptBuilder{},
	"DocumentChunkReranker":        &DocumentChunkReranker{},
	"QianFanDocumentChunkReranker": &QianFanDocumentChunkReranker{},
	"BaiLianDocumentChunkReranker": &BaiLianDocumentChunkReranker{},
	"LKEDocumentChunkReranker":     &LKEDocumentChunkReranker{},
	"MarkdownTOCRetriever":         &MarkdownTOCRetriever{},
	"FtsKeywordRetriever":          &FtsKeywordRetriever{},
	"VecEmbeddingRetriever":        &VecEmbeddingRetriever{},
	"OpenAITextEmbedder":           &OpenAITextEmbedder{},
	"LKETextEmbedder":              &LKETextEmbedder{},
	"WebSearch":                    &WebSearch{},
	"SQLiteVecDocumentStore":       &SQLiteVecDocumentStore{},
	"OpenAIDocumentEmbedder":       &OpenAIDocumentEmbedder{},
	"LKEDocumentEmbedder":          &LKEDocumentEmbedder{},
	"MarkdownTOCIndex":             &MarkdownTOCIndex{},
	"DocumentSplitter":             &DocumentSplitter{},
	"HtmlCleaner":                  &HtmlCleaner{},
	"WebScraper":                   &WebScraper{},
	"MarkdownConverter":            &MarkdownConverter{},
	"TikaConverter":                &TikaConverter{},
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
	if componentMap[componentName] == nil {
		return fmt.Errorf(funcT("The %s component of the pipeline does not exist"), componentName)
	}
	pipelineComponent := componentMap[componentName]
	err := pipelineComponent.Run(ctx, input)
	if err != nil {
		input[errorKey] = err
		return err
	}
	if input[errorKey] != nil {
		return input[errorKey].(error)
	}
	if input[endKey] != nil {
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

// TikaConverter 使用tika服务解析文档内容
type TikaConverter struct {
	FilePath       string            `json:"filePath,omitempty"`
	TiKaURL        string            `json:"tikaURL,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"maxRetries,omitempty"`
	client         *http.Client      `json:"-"`
}

func (component *TikaConverter) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 180
	}

	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
	}
	if component.TiKaURL == "" {
		component.TiKaURL = "http://localhost:9998/tika"
	}

	if component.DefaultHeaders == nil {
		component.DefaultHeaders = make(map[string]string, 0)
	}
	accept := component.DefaultHeaders["Accept"]
	contentType := component.DefaultHeaders["Content-Type"]
	//获取文本内容,没有html等其他标签
	if accept == "" {
		component.DefaultHeaders["Accept"] = "text/plain"
	}
	//上传文件的默认类型
	if contentType == "" {
		component.DefaultHeaders["Content-Type"] = "application/octet-stream"
	}

	//使用tesseract OCR组件处理PDF里的图片
	//component.DefaultHeaders["X-Tika-PDFextractInlineImages"] = "true"
	//OCR的字体
	//component.DefaultHeaders["X-Tika-OCRLanguage"] = "chi_sim"

	return nil
}
func (component *TikaConverter) Run(ctx context.Context, input map[string]interface{}) error {
	if input["document"] == nil {
		err := errors.New(funcT("The document of TikaConverter cannot be empty"))
		input[errorKey] = err
		return err
	}
	document := input["document"].(*Document)

	filePath := component.FilePath
	if filePath == "" {
		filePath = document.FilePath
	} else {
		document.FilePath = filePath
	}
	if filePath == "" && document.Markdown == "" {
		err := errors.New(funcT("The filePath of TikaConverter cannot be empty"))
		input[errorKey] = err
		return err
	}

	if document.Markdown == "" {
		markdownByte, err := httpUploadFile(component.client, "PUT", component.TiKaURL, datadir+filePath, component.DefaultHeaders)
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

// MarkdownConverter 调用markitdown解析文档
type MarkdownConverter struct {
	OpenAIChatGenerator

	Prompt          string `json:"prompt,omitempty"`          // 理解文档中图片的提示词
	Markitdown      string `json:"markitdown,omitempty"`      // markdown的命令路径
	MarkdownFileDir string `json:"markdownFileDir,omitempty"` // 生成的markdown文件目录
	ImageFileDir    string `json:"imageFileDir,omitempty"`    // 图片存放的目录
	ImageURLPrefix  string `json:"imageURLPrefix,omitempty"`  // 图片的URL前缀
	FilePath        string `json:"filePath,omitempty"`
}

func (component *MarkdownConverter) Initialization(ctx context.Context, input map[string]interface{}) error {

	if component.BaseURL == "" {
		component.BaseURL = config.AIBaseURL
	}
	component.OpenAIChatGenerator.Initialization(ctx, input)
	if component.Prompt == "" {
		component.Prompt = "提取图片内容,不要有引导语,介绍语,换行等"
	}
	defaultExecFile := datadir + "markitdown/markitdown"
	if component.Markitdown == "" && pathExist(defaultExecFile) {
		component.Markitdown = defaultExecFile
	}

	if component.MarkdownFileDir == "" {
		component.MarkdownFileDir = datadir + "upload/markitdown/markdown"
	}
	if !pathExist(component.MarkdownFileDir) {
		os.MkdirAll(component.MarkdownFileDir, 0755)
	}
	defaultImageFileDir := datadir + "upload/markitdown/image"
	if component.ImageFileDir == "" {
		component.ImageFileDir = defaultImageFileDir
	}
	if !pathExist(component.ImageFileDir) {
		os.MkdirAll(component.ImageFileDir, 0755)
	}
	if component.ImageURLPrefix == "" {
		component.ImageURLPrefix = "/upload/markitdown/image"
	}

	return nil
}
func (component *MarkdownConverter) Run(ctx context.Context, input map[string]interface{}) error {
	if input["document"] == nil {
		err := errors.New(funcT("The document of MarkdownConverter cannot be empty"))
		input[errorKey] = err
		return err
	}
	document := input["document"].(*Document)

	filePath := component.FilePath
	if filePath == "" {
		filePath = document.FilePath
	} else {
		document.FilePath = filePath
	}
	if filePath == "" && document.Markdown == "" {
		err := errors.New(funcT("The filePath of MarkitdownConverter cannot be empty"))
		input[errorKey] = err
		return err
	}

	// 优先使用markitdown转换
	if document.Markdown == "" && component.Markitdown != "" {
		uploadFilePath := datadir + filePath
		markdownFilePath := component.MarkdownFileDir + "/" + FuncGenerateStringID() + ".md"
		// 获取上传的文件信息
		fileInfo, err := os.Stat(uploadFilePath)
		if err != nil {
			input[errorKey] = err
			return err
		}
		document.FileSize = int(fileInfo.Size())
		cmd := component.Markitdown + " " + uploadFilePath + " -o " + markdownFilePath
		envs := make([]string, 0)
		envs = append(envs, "markitdown_api_key="+component.APIKey)
		envs = append(envs, "markitdown_base_url="+component.BaseURL)
		envs = append(envs, "markitdown_model="+component.Model)
		envs = append(envs, "markitdown_prompt="+component.Prompt)
		envs = append(envs, "markitdown_imageFileDir="+component.ImageFileDir)
		envs = append(envs, "markitdown_imageURLPrefix="+component.ImageURLPrefix)
		_, err = ExecCMD(cmd, envs, time.Second*60)
		if err != nil {
			input[errorKey] = err
			return err
		}
		markdownByte, err := os.ReadFile(markdownFilePath)
		if err != nil {
			input[errorKey] = err
			return err
		}
		document.Markdown = string(markdownByte)
		//document.FileSize = len(markdownByte)
	} else if document.Markdown == "" { //读取文本文件
		markdownByte, err := os.ReadFile(datadir + filePath)
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

// WebScraper 网络爬虫
type WebScraper struct {
	OpenAIChatGenerator
	//调用大模型转换为markdown的提示词,如果不为空,则调用大模型进行转换
	Prompt    string `json:"prompt,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	WebURL    string `json:"webURL,omitempty"`
	// Depth 抓取的深度,默认1,也就是当前页面
	Depth int `json:"depth,omitempty"`
	// QuerySelector 需要抓取的 querySelector
	QuerySelector   []string `json:"querySelector,omitempty"`
	KnowledgeBaseID string   `json:"knowledgeBaseID,omitempty"`
	Timeout         int      `json:"timeout,omitempty"`
	// RemoteChromeAddress 远程的chrome地址,例如 "ws://10.0.0.131:9222/",建议使用 chromedp/headless-shell 镜像
	RemoteChromeAddress string `json:"remoteChromeAddress,omitempty"`
	chromedpOptions     []chromedp.ExecAllocatorOption
}

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0"

func (component *WebScraper) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Timeout == 0 {
		component.Timeout = 60
	}
	if component.Depth == 0 {
		component.Depth = 1
	}
	if component.UserAgent == "" {
		component.UserAgent = userAgent
	}

	qs := make([]string, 0)
	for i := 0; i < len(component.QuerySelector); i++ {
		if component.QuerySelector[i] == "" {
			continue
		}
		qs = append(qs, component.QuerySelector[i])
	}
	component.QuerySelector = qs
	if len(component.QuerySelector) < 1 {
		component.QuerySelector = []string{"html"}
	}

	component.chromedpOptions = []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", true),             // debug使用 false
		chromedp.Flag("disable-hang-monitor", true), // 禁用页面无响应检测

		// 核心:禁用自动化指示器
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("useAutomationExtension", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		// 辅助:增强伪装
		chromedp.UserAgent(userAgent),
		chromedp.Flag("disable-web-security", false),
		chromedp.Flag("ignore-certificate-errors", false),
		// 随机化窗口大小,避免所有实例千篇一律
		chromedp.WindowSize(1920, 1080),
	}

	//初始化参数,先传一个空的数据
	component.chromedpOptions = append(chromedp.DefaultExecAllocatorOptions[:], component.chromedpOptions...)

	//component.RemoteChromeAddress = "ws://10.0.0.131:9222/"

	if component.Prompt != "" { //初始化大模型
		component.OpenAIChatGenerator.Initialization(ctx, input)
	}

	return nil
}
func (component *WebScraper) Run(ctx context.Context, input map[string]interface{}) error {

	if input["document"] == nil {
		err := errors.New(funcT("The document of WebScraper cannot be empty"))
		input[errorKey] = err
		return err
	}

	document := input["document"].(*Document)
	_, err := component.FetchPage(ctx, document, input)
	if err != nil {
		input[errorKey] = err
		return err
	}
	return nil
}

// FetchPage 抓取网页,方便后期扩展为递归
func (component *WebScraper) FetchPage(ctx context.Context, document *Document, input map[string]interface{}) ([]string, error) {
	webURL, has := input["webScraper_webURL"].(string)
	if webURL == "" || (!has) {
		webURL = component.WebURL
	}
	if webURL == "" {
		err := errors.New(funcT("The webScraper_webURL of WebScraper cannot be empty"))
		input[errorKey] = err
		return nil, err
	}

	var allocatorContext context.Context
	var cancel1 context.CancelFunc

	if component.RemoteChromeAddress != "" {
		allocatorContext, cancel1 = chromedp.NewRemoteAllocator(ctx, component.RemoteChromeAddress)
	} else {
		allocatorContext, cancel1 = chromedp.NewExecAllocator(ctx, component.chromedpOptions...)
	}
	defer cancel1()
	chromeCtx, cancel2 := chromedp.NewContext(allocatorContext)
	defer cancel2()
	// 执行一个空task, 用提前创建Chrome实例
	//chromedp.Run(chromeCtx, make([]chromedp.Action, 0, 1)...)

	//创建一个上下文,超时时间为60s
	chromeCtx, cancel3 := context.WithTimeout(chromeCtx, time.Duration(component.Timeout)*time.Second)
	defer cancel3()

	title := ""
	qsLen := len(component.QuerySelector)
	hcs := make([]string, qsLen)
	hrefs := make([][]string, qsLen)

	//执行动作
	err := chromedp.Run(chromeCtx, chromedp.Tasks{
		// 启用网络事件监听,这是关键一步
		//network.Enable(),
		// 可同时设置额外请求头
		//network.SetExtraHTTPHeaders(network.Headers{"Accept-Language": "zh-CN,zh;q=0.9", "User-Agent": userAgent}),

		// 覆盖 navigator.userAgent 等
		emulation.SetUserAgentOverride(userAgent),
		// 指定分辨率的窗口
		emulation.SetDeviceMetricsOverride(1920, 1080, 1.0, false).
			WithScreenOrientation(&emulation.ScreenOrientation{
				Type:  emulation.OrientationTypePortraitPrimary,
				Angle: 0,
			}),

		chromedp.Navigate(webURL),
		// 双重等待机制
		chromedp.WaitReady("body", chromedp.BySearch), // 等待body标签存在
		chromedp.Sleep(2 * time.Second),               // 容错性等待

		// 自定义处理逻辑,忽略页面错误
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i := 0; i < qsLen; i++ {
				//获取网页的内容,chromedp.AtLeast(0)立即执行,不要等待
				//err := chromedp.OuterHTML(component.QuerySelector[i], &hcs[i], chromedp.ByQueryAll, chromedp.AtLeast(0)).Do(ctx)
				outerHTMLs := make([]string, 0)
				err := chromedp.Evaluate(fmt.Sprintf(`Array.from(document.querySelectorAll('%s')).map(el => el.innerText)`, component.QuerySelector[i]), &outerHTMLs).Do(ctx)
				if err != nil {
					continue
				}
				hcs = append(hcs, outerHTMLs...)
				// 获取页面的超链接
				if component.Depth > 1 {
					chromedp.Evaluate(fmt.Sprintf("Array.from(document.querySelectorAll('%s a')).map(a => a.href)", component.QuerySelector[i]), &hrefs[i]).Do(ctx)
				}
			}
			return nil
		}),
		// 获取网页的title,放到最后再执行
		chromedp.Title(&title),
	})
	if err != nil {
		return nil, err
	}
	document.Markdown = strings.Join(hcs, ".")
	markdown, err := component.convertMarkdown(ctx, title, document.Markdown)
	if markdown != "" {
		document.Markdown = markdown
	}
	document.Name = title
	hrefSlice := make([]string, 0)
	for i := 0; i < len(hrefs); i++ {
		for _, v := range hrefs[i] {
			if slices.Contains(hrefSlice, v) {
				continue
			}
			hrefSlice = append(hrefSlice, v)
		}

	}

	return hrefSlice, nil
}

// @TODO 也可以尝试网页截屏,然后让多模态大模型识别网页正文,方便去除广告.
// convertMarkdown 使用大模型,把抓取的html页面转换为markdown格式
func (component *WebScraper) convertMarkdown(ctx context.Context, title string, html string) (string, error) {
	if component.Prompt == "" || html == "" {
		return html, nil
	}

	/*
		message := `
			提供内容整理成markdown格式,根据内容拆分合理的目录标题,不要做扩展,只整理格式.如果可能,末级标题的内容至少200字,必须是原文档的内容,不要修改内容.
			返回的json格式示例:{"markdown":<整理的markdown内容>}
			需要整理为markdown的内容:
			` + html
	*/
	message := component.Prompt + " \n 网页标题:" + title + "\n 网页内容:" + html
	component.Temperature = 0.1
	// 请求大模型,获取json结果
	resultJson, err := llmJSONResult(ctx, component.OpenAIChatGenerator, message)
	if err != nil {
		return "", err
	}
	markdownResult := struct {
		Markdown string `json:"markdown,omitempty"`
	}{}
	err = json.Unmarshal([]byte(resultJson), &markdownResult)
	return markdownResult.Markdown, err
}

// HtmlCleaner 清理html标签
type HtmlCleaner struct {
}

func (component *HtmlCleaner) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *HtmlCleaner) Run(ctx context.Context, input map[string]interface{}) error {
	if input["document"] == nil {
		err := errors.New(funcT("The document of DocumentSplitter cannot be empty"))
		input[errorKey] = err
		return err
	}
	document := input["document"].(*Document)
	doc, err := html.Parse(bytes.NewReader([]byte(document.Markdown)))
	if err != nil {
		return err
	}

	var buf strings.Builder
	var inScript, inStyle bool // 新增状态标记

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		// 新增标签检测逻辑
		switch n.Type {
		case html.ElementNode:
			switch n.Data {
			case "script":
				inScript = true
			case "style":
				inStyle = true
			}
		case html.TextNode:
			if !inScript && !inStyle { // 仅收集非脚本/样式内容
				buf.WriteString(strings.TrimSpace(n.Data) + " ")
			}
		}

		// 递归处理子节点(跳过脚本/样式内容)
		if !inScript && !inStyle {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extract(c)
			}
		}

		// 重置标签状态
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script":
				inScript = false
			case "style":
				inStyle = false
			}
		}
	}

	extract(doc)
	document.Markdown = html.UnescapeString(buf.String())
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

	if input["document"] == nil {
		err := errors.New(funcT("The document of DocumentSplitter cannot be empty"))
		input[errorKey] = err
		return err
	}
	document := input["document"].(*Document)
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

// MarkdownTOCIndex markdown目录索引
type MarkdownTOCIndex struct {
	OpenAIChatGenerator
}

func (component *MarkdownTOCIndex) Initialization(ctx context.Context, input map[string]interface{}) error {
	component.OpenAIChatGenerator.Initialization(ctx, input)
	return nil
}
func (component *MarkdownTOCIndex) Run(ctx context.Context, input map[string]interface{}) error {

	if input["document"] == nil {
		err := errors.New(funcT("The document of DocumentSplitter cannot be empty"))
		input[errorKey] = err
		return err
	}
	document := input["document"].(*Document)
	if document.Markdown == "" { //没有内容
		return nil
	}

	// 解析 Markdown
	tree, list, err := parseMarkdownToTree([]byte(document.Markdown))
	if err != nil {
		input[errorKey] = err
		return err
	}

	markdown := ""
	// 内容没有树形结构,调用模型生成markdown格式
	if len(tree) == 0 || len(list) == 0 {
		message := `
		提供内容整理成markdown格式,根据内容拆分合理的目录标题,不要做扩展,只整理格式.如果可能,末级标题的内容至少200字,必须是原文档的内容,不要修改内容.
		返回的json格式示例:{"markdown":<整理的markdown内容>}
		需要整理为markdown的内容:
		` + document.Markdown
		component.Temperature = 0.1
		// 请求大模型,获取json结果
		resultJson, err := llmJSONResult(ctx, component.OpenAIChatGenerator, message)
		if err != nil {
			input[errorKey] = err
			return err
		}
		markdownResult := struct {
			Markdown string `json:"markdown,omitempty"`
		}{}
		err = json.Unmarshal([]byte(resultJson), &markdownResult)
		if err != nil {
			input[errorKey] = err
			return err
		}
		if markdownResult.Markdown != "" {
			markdown = markdownResult.Markdown
		} else {
			return nil
		}
	}

	// 解析 Markdown
	if markdown != "" {
		tree, list, err = parseMarkdownToTree([]byte(markdown))
		if err != nil {
			input[errorKey] = err
			return err
		}
		// 内容没有树形结构,调用模型生成markdown格式
		if len(tree) == 0 || len(list) == 0 {
			return nil
		}
		document.Markdown = markdown
	}
	jsonByte, err := json.Marshal(tree)
	if err != nil {
		input[errorKey] = err
		return err
	}
	//使用整个对象进行json,减少不不必要的字段
	tocList := make([]DocumentTOC, 0)
	err = json.Unmarshal(jsonByte, &tocList)
	if err != nil {
		input[errorKey] = err
		return err
	}
	tocBytes, err := json.Marshal(tocList)
	if err != nil {
		input[errorKey] = err
		return err
	}
	//文档的目录
	document.Toc = string(tocBytes)

	documentChunks := make([]DocumentChunk, 0)
	for i := 0; i < len(list); i++ {
		documentChunk := list[i]
		documentChunk.DocumentID = document.Id
		documentChunk.KnowledgeBaseID = document.KnowledgeBaseID
		documentChunk.CreateTime = document.CreateTime
		documentChunk.UpdateTime = document.UpdateTime
		documentChunk.SortNo = i
		documentChunk.Status = document.Status
		documentChunks = append(documentChunks, *documentChunk)
	}

	input["documentChunks"] = documentChunks
	return nil
}

// OpenAIDocumentEmbedder 向量化文档字符串
type OpenAIDocumentEmbedder struct {
	OpenAIChatGenerator
}

func (component *OpenAIDocumentEmbedder) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Model == "" {
		return errors.New("Initialization OpenAIDocumentEmbedder error:Model is empty")
	}
	if component.BaseURL == "" {
		component.BaseURL = config.AIBaseURL + "/embeddings"
	}
	component.OpenAIChatGenerator.Initialization(ctx, input)
	return nil
}
func (component *OpenAIDocumentEmbedder) Run(ctx context.Context, input map[string]interface{}) error {

	if input["documentChunks"] == nil {
		return errors.New(funcT("input['documentChunks'] cannot be empty"))
	}
	documentChunks := input["documentChunks"].([]DocumentChunk)
	vecDocumentChunks := make([]VecDocumentChunk, 0)
	for i := 0; i < len(documentChunks); i++ {
		bodyMap := make(map[string]interface{}, 0)
		bodyMap["input"] = []string{documentChunks[i].Markdown}
		bodyMap["model"] = component.Model
		bodyMap["encoding_format"] = "float"
		bodyByte, err := httpPostJsonBody(component.client, component.APIKey, component.BaseURL, component.DefaultHeaders, bodyMap)

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
	var document *Document
	if input["document"] == nil {
		err := errors.New(funcT("The document of SQLiteVecDocumentStore cannot be empty"))
		input[errorKey] = err
		return err
	}
	document = input["document"].(*Document)

	var documentChunks []DocumentChunk
	var vecDocumentChunks []VecDocumentChunk
	if input["documentChunks"] != nil {
		documentChunks = input["documentChunks"].([]DocumentChunk)
	}
	if input["vecDocumentChunks"] != nil {
		vecDocumentChunks = input["vecDocumentChunks"].([]VecDocumentChunk)
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
	OpenAIChatGenerator
}

func (component *OpenAITextEmbedder) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Model == "" {
		return errors.New("Initialization OpenAITextEmbedder error:Model is empty")
	}
	if component.BaseURL == "" {
		component.BaseURL = config.AIBaseURL + "/embeddings"
	}
	component.OpenAIChatGenerator.Initialization(ctx, input)
	return nil
}
func (component *OpenAITextEmbedder) Run(ctx context.Context, input map[string]interface{}) error {

	if input["query"] == nil {
		return errors.New(funcT("input['query'] cannot be empty"))
	}
	query := input["query"].(string)
	bodyMap := make(map[string]interface{}, 0)
	bodyMap["input"] = []string{query}
	bodyMap["model"] = component.Model
	bodyMap["encoding_format"] = "float"
	bodyByte, err := httpPostJsonBody(component.client, component.APIKey, component.BaseURL, component.DefaultHeaders, bodyMap)
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
	// TopN 检索多少条
	TopN int `json:"top_n,omitempty"`
	// Score 向量表的score匹配分数
	Score float32 `json:"score,omitempty"`
}

func (component *VecEmbeddingRetriever) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *VecEmbeddingRetriever) Run(ctx context.Context, input map[string]interface{}) error {
	documentID := ""
	knowledgeBaseID := ""
	topN := 0
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
	tId, has := input["topN"]
	if has {
		topN = tId.(int)
	}
	if topN == 0 {
		topN = component.TopN
	}
	if topN == 0 {
		topN = 5
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
	finder.Append("ORDER BY score LIMIT " + strconv.Itoa(topN))
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
	documentChunks = sortDocumentChunksScore(documentChunks, topN, score)

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
	// TopN 检索多少条
	TopN int `json:"top_n,omitempty"`
	// Score BM25的FTS5实现在返回结果之前将结果乘以-1,得分越小(数值上更负),表示匹配越好
	Score float32 `json:"score,omitempty"`
}

func (component *FtsKeywordRetriever) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}
func (component *FtsKeywordRetriever) Run(ctx context.Context, input map[string]interface{}) error {
	documentID := ""
	knowledgeBaseID := ""
	topN := 0
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
	tId, has := input["topN"]
	if has {
		topN = tId.(int)
	}
	if topN == 0 {
		topN = component.TopN
	}
	if topN == 0 {
		topN = 5
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
	finder.Append("ORDER BY score DESC LIMIT " + strconv.Itoa(topN))
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

// MarkdownTOCRetriever 使用文档目录索引
type MarkdownTOCRetriever struct {
	// DocumentID 文档ID
	DocumentID string `json:"documentID,omitempty"`
	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `json:"knowledgeBaseID,omitempty"`
	// Query 需要查询的关键字
	Query string `json:"query,omitempty"`
	// TopN 检索多少条
	TopN int `json:"top_n,omitempty"`

	// 声明LLM大语言类型
	OpenAIChatGenerator

	// PromptTemplate 大模型检索目录的提示词
	PromptTemplate string             `json:"promptTemplate,omitempty"`
	t              *template.Template `json:"-"`
}

func (component *MarkdownTOCRetriever) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.TopN == 0 {
		component.TopN = 5
	}

	var err error
	tmpl := template.New("minrag-DocumentTOCIndexRetriever")
	component.t, err = tmpl.Parse(component.PromptTemplate)
	if err != nil {
		return err
	}

	// 调用 LLM模型的初始化方法
	component.OpenAIChatGenerator.Initialization(ctx, input)

	return nil
}
func (component *MarkdownTOCRetriever) Run(ctx context.Context, input map[string]interface{}) error {
	documentID := ""
	knowledgeBaseID := ""
	topN := 0
	query := ""
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
	tId, has := input["topN"]
	if has {
		topN = tId.(int)
	}
	if topN == 0 {
		topN = component.TopN
	}

	// 查询文档的目录
	finder := zorm.NewFinder().Append("SELECT id,name,summary,toc from " + tableDocumentName + " WHERE 1=1")
	if documentID != "" {
		finder.Append(" and documentID=?", documentID)
	}
	if knowledgeBaseID != "" {
		finder.Append(" and knowledgeBaseID like ?", knowledgeBaseID+"%")
	}

	documents := make([]Document, 0)
	err := zorm.Query(ctx, finder, &documents, nil)
	if err != nil {
		input[errorKey] = err
		return err
	}

	if len(documents) < 1 { //没有文档
		return nil
	}

	tocMap := map[string]interface{}{
		"documents": documents,
	}

	// 创建一个 bytes.Buffer 用于存储渲染后的 text 内容
	var buf bytes.Buffer
	// 执行模板并将结果写入到 bytes.Buffer
	if err := component.t.Execute(&buf, tocMap); err != nil {
		input[errorKey] = err
		return err
	}
	// 获取编译后的内容
	message := buf.String()
	var messages []ChatMessage
	if input["messages"] != nil {
		messages = input["messages"].([]ChatMessage)
	}
	//添加文档的树状结构
	messages = append(messages, ChatMessage{Role: "system", Content: message})
	input["messages"] = messages

	// input 中的tools对象
	var tools []interface{}
	if input["tools"] != nil {
		tools = input["tools"].([]interface{})
	}
	fc := functionCallingMap[fcSearchDocumentByNodeName]
	tools = append(tools, fc.Description(ctx))
	input["tools"] = tools

	return nil
}

// llmJSONResult 请求大模型获得json结果
func llmJSONResult(ctx context.Context, component OpenAIChatGenerator, message string) (string, error) {
	bodyMap := make(map[string]interface{})
	bodyMap["messages"] = []ChatMessage{{Role: "user", Content: message}}
	bodyMap["model"] = component.Model
	if component.Temperature != 0 {
		bodyMap["temperature"] = component.Temperature
	}
	bodyMap["response_format"] = map[string]string{"type": "json_object"}
	//输出类型
	bodyMap["stream"] = false
	//请求大模型
	bodyByte, err := httpPostJsonBody(component.client, component.APIKey, component.BaseURL, component.DefaultHeaders, bodyMap)
	if err != nil {
		return "", err
	}
	rs := struct {
		Choices []Choice `json:"choices,omitempty"`
	}{}
	err = json.Unmarshal(bodyByte, &rs)
	if err != nil {
		return "", err
	}
	if len(rs.Choices) < 1 {
		return "", nil
	}
	//获取第一个结果
	resultJson := rs.Choices[0].Message.Content
	return resultJson, nil
}

// DocumentChunkReranker 对DocumentChunks进行重新排序
type DocumentChunkReranker struct {
	// 声明LLM大语言类型
	OpenAIChatGenerator

	// Query 需要查询的关键字
	Query string `json:"query,omitempty"`
	// TopN 检索多少条
	TopN int `json:"top_n,omitempty"`
	// Score ranker的score匹配分数
	Score float32 `json:"score,omitempty"`
}

func (component *DocumentChunkReranker) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Model == "" {
		return errors.New("Initialization DocumentChunkReranker error:Model is empty")
	}
	if component.BaseURL == "" {
		// 兼容 Jina
		component.BaseURL = config.AIBaseURL + "/rerank"
	}

	component.OpenAIChatGenerator.Initialization(ctx, input)
	return nil
}
func (component *DocumentChunkReranker) Run(ctx context.Context, input map[string]interface{}) error {

	query, topN, score, documentChunks, documents, err := component.checkRerankParameter(ctx, input)
	if err != nil {
		input[errorKey] = err
		return err
	}
	if documentChunks == nil {
		return nil
	}
	bodyMap := map[string]interface{}{
		"model":     component.Model,
		"query":     query,
		"top_n":     topN,
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
			Index          int     `json:"index,omitempty"`
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
	rerankerDCS = sortDocumentChunksScore(rerankerDCS, topN, score)
	input["documentChunks"] = rerankerDCS
	return nil
}

// checkRerankParameter 检查input里的参数,返回 query,topN,score,documentChunks,documents,error
func (component *DocumentChunkReranker) checkRerankParameter(ctx context.Context, input map[string]interface{}) (string, int, float32, []DocumentChunk, []string, error) {
	topN := 0
	var score float32 = 0.0

	if input["documentChunks"] == nil {
		err := errors.New(funcT("input['documentChunks'] cannot be empty"))
		return "", 0, 0.0, nil, nil, err
	}
	documentChunks := input["documentChunks"].([]DocumentChunk)

	if input["query"] == nil {
		err := errors.New(funcT("input['query'] cannot be empty"))
		return "", 0, 0.0, nil, nil, err
	}
	query := input["query"].(string)
	tId, has := input["topN"]
	if has {
		topN = tId.(int)
	}
	if topN == 0 {
		topN = component.TopN
	}
	if topN == 0 {
		topN = 5
	}

	tScore, has := input["score"]
	if has {
		score = tScore.(float32)
	}
	if score <= 0 {
		score = component.Score
	}
	if topN > len(documentChunks) {
		topN = len(documentChunks)
	}
	if len(documentChunks) < 1 { //没有文档,不需要重排
		return "", 0, 0.0, nil, nil, nil
	}
	documents := make([]string, 0)
	for i := 0; i < len(documentChunks); i++ {
		documents = append(documents, documentChunks[i].Markdown)
	}
	return query, topN, score, documentChunks, documents, nil

}

func sortDocumentChunksScore(documentChunks []DocumentChunk, topN int, score float32) []DocumentChunk {
	sort.Slice(documentChunks, func(i, j int) bool {
		return documentChunks[i].Score > documentChunks[j].Score
	})

	resultDCS := make([]DocumentChunk, 0)
	for i := 0; i < len(documentChunks); i++ {
		documentChunk := documentChunks[i]
		if len(resultDCS) >= topN {
			break
		}
		if documentChunk.Score < score {
			continue
		}
		resultDCS = append(resultDCS, documentChunk)
	}
	return resultDCS
}

// WebSearch 联网搜索,基于网络爬虫扩展
type WebSearch struct {
	WebScraper
	// TopN 检索前几个链接,默认5
	TopN int `json:"top_n,omitempty"`
}

func (component *WebSearch) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.TopN == 0 {
		component.TopN = 5
	}
	err := component.WebScraper.Initialization(ctx, input)
	return err
}

func (component *WebSearch) Run(ctx context.Context, input map[string]interface{}) error {
	// input 中的tools对象
	var tools []interface{}
	if input["tools"] != nil {
		tools = input["tools"].([]interface{})
	}
	fc := functionCallingMap[fcWebSearchName]
	tools = append(tools, fc.Description(ctx))
	input["tools"] = tools
	return nil
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

		if input["documentChunks"] == nil {
			input[endKey] = true
			return nil
		}
		documentChunks := input["documentChunks"].([]DocumentChunk)
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

	if input["prompt"] == nil {
		err := errors.New(funcT("input['prompt'] cannot be empty"))
		input[errorKey] = err
		return err
	}
	prompt := input["prompt"].(string)
	messages := make([]ChatMessage, 0)
	if input["messages"] != nil {
		messages = input["messages"].([]ChatMessage)
	}
	if input["agentID"] != nil {
		agent, err := findAgentByID(ctx, input["agentID"].(string))
		if err != nil {
			input[errorKey] = err
			return err
		}
		agentPrompt := ChatMessage{Role: "system", Content: agent.AgentPrompt}
		messages = append(messages, agentPrompt)

		// input 中的tools对象
		var tools []interface{}
		if input["tools"] != nil {
			tools = input["tools"].([]interface{})
		}

		//tools
		if len(agent.Tools) > 0 {
			toolSlice := make([]string, 0)
			json.Unmarshal([]byte(agent.Tools), &toolSlice)

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

	roomID, _ := input["roomID"].(string)

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

	promptMessage := ChatMessage{Role: "user", Content: prompt}
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
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	Type             string     `json:"type,omitempty"`              //thinking 思维链内容
	ReasoningContent string     `json:"reasoning_content,omitempty"` //reasoning_content 思维链内容
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
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
	MaxDeep        int               `json:"maxDeep,omitempty"` //最大迭代深度
	//MaxCompletionTokens int64             `json:"maxCompletionTokens,omitempty"`
	client *http.Client `json:"-"`
}

func (component *OpenAIChatGenerator) Initialization(ctx context.Context, input map[string]interface{}) error {
	if component.Model == "" {
		component.Model = config.LLMModel
	}
	if component.Model == "" {
		return errors.New("Initialization OpenAIChatGenerator error:Model is empty")
	}

	if component.Timeout == 0 {
		component.Timeout = 180
	}
	if component.MaxDeep == 0 {
		component.MaxDeep = 10
	}

	component.client = &http.Client{
		Timeout: time.Second * time.Duration(component.Timeout),
	}
	if component.BaseURL == "" {
		component.BaseURL = config.AIBaseURL + "/chat/completions"
	}
	if component.APIKey == "" {
		component.APIKey = config.AIAPIkey
	}
	if component.DefaultHeaders == nil {
		component.DefaultHeaders = make(map[string]string, 0)
	}
	return nil
}
func (component *OpenAIChatGenerator) Run(ctx context.Context, input map[string]interface{}) error {

	if input["query"] == nil {
		err := errors.New(funcT("input['query'] cannot be empty"))
		input[errorKey] = err
		return err
	}
	query := input["query"].(string)
	messages := make([]ChatMessage, 0)
	if input["messages"] != nil {
		messages = input["messages"].([]ChatMessage)
	} else {
		cm := ChatMessage{Role: "user", Content: query}
		messages = append(messages, cm)
	}

	bodyMap := make(map[string]interface{})

	bodyMap["model"] = component.Model
	if component.Temperature != 0 {
		bodyMap["temperature"] = component.Temperature
	}

	tools, has := input["tools"]
	if has {
		bodyMap["tools"] = tools
	} else { //没有tools函数,默认迭代一次
		component.MaxDeep = 1
	}

	c := input["c"].(*app.RequestContext)

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

	// 多次深入迭代调用
	for i := 0; i < component.MaxDeep; i++ {
		// 设置message 参数
		bodyMap["messages"] = messages

		if !stream { //一次性输出,不是流式输出
			//请求大模型
			bodyByte, err := httpPostJsonBody(component.client, component.APIKey, component.BaseURL, component.DefaultHeaders, bodyMap)
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
			rsByte, _ := json.Marshal(rs)
			rsStr := string(rsByte)
			//没有函数调用,把模型返回的choice放入到input["choice"],并输出
			if len(choice.Message.ToolCalls) == 0 {
				input["choice"] = choice
				c.WriteString(rsStr)
				c.Flush()
				return nil
			}
			//追加返回的 assistant message
			messages = append(messages, choice.Message)

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
			//delete(input, "tools")
			//重新运行组件,调用大模型.
			//component.Run(ctx, input)

			// 进入下一次循环
			continue
		}

		// toolCalls 需要调用的函数列表,如果有值,说明需要调用函数,不能直接返回结果
		var toolCalls []ToolCall

		//设置SSE的协议头
		component.DefaultHeaders["Accept"] = "text/event-stream"
		component.DefaultHeaders["Cache-Control"] = "no-cache"
		component.DefaultHeaders["Connection"] = "keep-alive"

		//请求大模型
		resp, err := httpPostJsonResponse(component.client, component.APIKey, component.BaseURL, component.DefaultHeaders, bodyMap)
		if err != nil {
			input[errorKey] = err
			return err
		}
		defer resp.Body.Close()
		//用于拼接stream返回的最终结果
		choice := Choice{FinishReason: "stop"}
		//消息内容
		var messageContent strings.Builder
		//推理内容
		var reasoningContent strings.Builder
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
				//err := errors.New("httpPostJsonResponse choices is empty")
				//input[errorKey] = err
				//return err
				continue
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
				toolCalls[tc.Index].Type = "function"
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

			rsByte, _ := json.Marshal(rs)
			rsStr := string(rsByte)

			// 不是函数调用,把返回的内容输出到页面
			if c != nil && len(toolCalls) == 0 {
				c.WriteString("data: " + rsStr + "\n\n")
				c.Flush()
			}
			// 不是函数调用,把内容拼接起来
			if len(toolCalls) == 0 {
				messageContent.WriteString(rs.Choices[0].Delta.Content)
				reasoningContent.WriteString(rs.Choices[0].Delta.ReasoningContent)
			}
		}
		// 大模型的返回
		message := ChatMessage{Role: "assistant", Content: messageContent.String(), ReasoningContent: reasoningContent.String(), ToolCalls: toolCalls}
		//没有函数调用,把模型返回的choice放入到input["choice"]
		if len(toolCalls) == 0 {
			choice.Message = message
			input["choice"] = choice
			return nil
		}
		//追加返回的 assistant message
		messages = append(messages, message)

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
		//delete(input, "tools")
		//重新运行组件,调用大模型.
		//component.Run(ctx, input)
	}
	return nil

}

// ChatMessageLogStore 保存消息记录到数据库
type ChatMessageLogStore struct {
}

func (component *ChatMessageLogStore) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}

func (component *ChatMessageLogStore) Run(ctx context.Context, input map[string]interface{}) error {

	if input["c"] == nil {
		return errors.New(`input["c"] is nil`)
	}
	c := input["c"].(*app.RequestContext)

	if input["roomID"] == nil {
		return errors.New(`input["roomID"] is nil`)
	}
	roomID := input["roomID"].(string)

	if input["agentID"] == nil {
		return errors.New(`input["agentID"] is nil`)
	}
	agentID := input["agentID"].(string)

	if input["query"] == nil {
		return errors.New(`input["query"] is nil`)
	}
	query := input["query"].(string)

	agent, err := findAgentByID(ctx, agentID)
	if err != nil {
		return err
	}
	if input["choice"] == nil {
		return errors.New(`input["choice"] is nil`)
	}

	choice := input["choice"].(Choice)

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
