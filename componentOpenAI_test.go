package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestHtmlCleaner(t *testing.T) {
	mk := `
<html><body><p>这是一个示例文本。</p><a href="https://minrag.com">链接</a></body></html>
`
	hc := HtmlCleaner{}
	input := make(map[string]any, 0)
	document := &Document{Markdown: mk}
	input["document"] = document
	ctx := context.Background()
	hc.Initialization(ctx, input)
	err := hc.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(document.Markdown)
}

func TestWebScraper(t *testing.T) {
	ws := WebScraper{QuerySelector: []string{"#s-top-left"}}
	ws.Depth = 2
	input := make(map[string]any, 0)
	document := &Document{Markdown: ""}
	input["document"] = document
	ws.WebURL = "https://www.baidu.com"
	ctx := context.Background()
	ws.Initialization(ctx, input)
	herf, err := ws.FetchPage(ctx, document, input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(document)
	fmt.Println("-------------------")
	fmt.Println(herf)

}

func TestWebSearch(t *testing.T) {
	ws := WebSearch{}
	ws.Depth = 2
	input := make(map[string]any, 0)
	document := &Document{Markdown: ""}
	input["document"] = document
	ctx := context.Background()
	ws.WebURL = "https://www.bing.com/search?q="
	input["query"] = "minrag"
	ws.QuerySelector = []string{"li.b_algo div.b_tpcn"}
	//ws.QuerySelector = []string{"body"}
	ws.Initialization(ctx, input)
	err := ws.Run(ctx, input)
	//herf, err := ws.FetchPage(ctx, document, input)
	//fmt.Println(herf)
	if err != nil {
		t.Fatal(err)
	}
	webSerachDocuments := input["webSerachDocuments"].([]Document)
	fmt.Println(webSerachDocuments)
	fmt.Println("-------------------")

}

func TestDocumentSplitter(t *testing.T) {
	ctx := context.Background()
	documentSplitter := &DocumentSplitter{
		SplitBy:      []string{"\f", "\n\n", "\n", "。", "！", "!", "？", "?", ".", ";", "；", "，", ",", " "},
		SplitLength:  500,
		SplitOverlap: 0,
	}
	input := make(map[string]any, 0)
	input["document"] = &Document{Markdown: "我是中国人,我爱中国。圣诞节,了大家安康金发傲娇考虑实际得分拉萨放假啊十六分是。1。2。3。"}
	err := documentSplitter.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	ds := input["documentChunks"]
	documentChunks := ds.([]DocumentChunk)
	for i := 0; i < len(documentChunks); i++ {
		documentChunk := documentChunks[i]
		fmt.Println(documentChunk.Markdown)
	}

}

func TestFtsKeywordRetriever(t *testing.T) {
	ctx := context.Background()
	ftsKeywordRetriever := baseComponentMap["FtsKeywordRetriever"]
	input := make(map[string]any, 0)
	input["query"] = "马斯克"
	err := ftsKeywordRetriever.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	ds := input["documentChunks"]
	documentChunks := ds.([]DocumentChunk)
	for i := 0; i < len(documentChunks); i++ {
		documentChunk := documentChunks[i]
		fmt.Println(documentChunk)
	}
}

func TestDocumentChunkReranker(t *testing.T) {
	ctx := context.Background()
	documentChunkReranker := baseComponentMap["DocumentChunkReranker"]
	input := make(map[string]any, 0)
	input["query"] = "你在哪里?"
	documentChunks := make([]DocumentChunk, 3)
	documentChunks[0] = DocumentChunk{Markdown: "我在郑州"}
	documentChunks[1] = DocumentChunk{Markdown: "今天晴天"}
	documentChunks[2] = DocumentChunk{Markdown: "我明天去旅游"}
	input["documentChunks"] = documentChunks
	err := documentChunkReranker.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	ds := input["documentChunks"]
	documentChunks = ds.([]DocumentChunk)
	for i := 0; i < len(documentChunks); i++ {
		documentChunk := documentChunks[i]
		fmt.Println(documentChunk)
	}
}

func TestPromptBuilder(t *testing.T) {
	ctx := context.Background()
	promptBuilder := baseComponentMap["PromptBuilder"]
	input := make(map[string]any, 0)
	input["query"] = "你在哪里?"
	documentChunks := make([]DocumentChunk, 3)
	documentChunks[0] = DocumentChunk{Markdown: "我在郑州"}
	documentChunks[1] = DocumentChunk{Markdown: "今天晴天"}
	documentChunks[2] = DocumentChunk{Markdown: "我明天去旅游"}
	input["documentChunks"] = documentChunks
	err := promptBuilder.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(input["prompt"])

	openAIChatMemory := baseComponentMap["OpenAIChatMemory"]
	openAIChatMemory.Run(ctx, input)

	openAIChatGenerator := baseComponentMap["OpenAIChatGenerator"]
	openAIChatGenerator.Run(ctx, input)
	choice := input["choice"]
	fmt.Println(choice)
}

func TestPipline(t *testing.T) {
	ctx := context.Background()
	defaultPipline := baseComponentMap["default"]
	input := make(map[string]any, 0)
	input["query"] = "你在哪里?"
	documentChunks := make([]DocumentChunk, 3)
	documentChunks[0] = DocumentChunk{Markdown: "我在郑州"}
	documentChunks[1] = DocumentChunk{Markdown: "今天晴天"}
	documentChunks[2] = DocumentChunk{Markdown: "我明天去旅游"}
	input["documentChunks"] = documentChunks
	err := defaultPipline.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	choice := input["choice"]
	fmt.Println(choice)
}

func TestDocumentSplitterNew(t *testing.T) {
	ctx := context.Background()
	documentSplitter := &DocumentSplitter{
		SplitBy:      []string{"\f", "\n\n", "\n", "。", "！", "!", "？", "?", ".", ";", "；", "，", ",", " "},
		SplitLength:  100, // 较小的长度以便测试分割
		SplitOverlap: 0,
	}

	// 创建一个较长的中文文本
	longText := `自然语言处理（Natural Language Processing，NLP）是人工智能和语言学领域的分支学科。
此领域探讨如何处理及运用自然语言；自然语言处理包括多方面和步骤，基本有认知、理解、生成等部分。
自然语言认知和理解是让电脑把输入的语言变成有意思的符号和关系，然后根据目的再处理。
自然语言生成系统则是把计算机数据转化为自然语言。自然语言处理要研制表示语言能力和语言应用的模型，
建立计算框架来实现这样的语言模型，提出相应的方法来不断完善这样的语言模型，根据这样的语言模型设计各种实用系统，
并探讨这些实用系统的评测技术。自然语言处理并不是一般地研究自然语言，而在于研制能有效地实现自然语言通信的计算机系统，
特别是其中的软件系统。因而它是计算机科学的一部分。自然语言处理的主要范畴包括：文本朗读、语音合成、语音识别、
自动分词、词性标注、句法分析、自然语言生成、文本分类、信息检索、信息抽取、文字校对、问答系统、机器翻译、
自动摘要、文字蕴涵等。`

	input := make(map[string]any, 0)
	input["document"] = &Document{Markdown: longText}
	err := documentSplitter.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	ds := input["documentChunks"]
	documentChunks := ds.([]DocumentChunk)

	t.Logf("原始文本长度（符文）: %d", len([]rune(longText)))
	t.Logf("分割后的块数: %d", len(documentChunks))

	for i, chunk := range documentChunks {
		chunkLen := len([]rune(chunk.Markdown))
		t.Logf("块 %d 长度: %d, 内容预览: %.50s...", i+1, chunkLen, chunk.Markdown)
		// 检查每个块的长度是否接近目标长度（允许一定的灵活性）
		if chunkLen > documentSplitter.SplitLength*2 {
			t.Errorf("块 %d 长度 %d 超过目标长度的两倍 (%d)", i+1, chunkLen, documentSplitter.SplitLength)
		}
	}

	// 验证所有块拼接起来应该接近原始文本
	allChunks := ""
	for _, chunk := range documentChunks {
		allChunks += chunk.Markdown
	}
	// 由于分割可能会丢失一些分隔符，我们只检查大致内容
	originalRunes := []rune(longText)
	chunkRunes := []rune(allChunks)
	if len(originalRunes)-len(chunkRunes) > 10 { // 允许少量差异
		t.Errorf("拼接后长度差异过大: 原始 %d, 拼接后 %d", len(originalRunes), len(chunkRunes))
	}
}

func TestDocumentSplitterFunctions(t *testing.T) {
	// 测试符文长度计算
	t.Run("TestRuneLength", func(t *testing.T) {
		// 测试中文字符
		if runeLength("你好") != 2 {
			t.Errorf("中文符文长度错误: 期望 2, 得到 %d", runeLength("你好"))
		}
		// 测试英文字符
		if runeLength("hello") != 5 {
			t.Errorf("英文符文长度错误: 期望 5, 得到 %d", runeLength("hello"))
		}
		// 测试混合字符
		if runeLength("hello世界") != 7 {
			t.Errorf("混合符文长度错误: 期望 7, 得到 %d", runeLength("hello世界"))
		}
		// 测试空字符串
		if runeLength("") != 0 {
			t.Errorf("空字符串符文长度错误: 期望 0, 得到 %d", runeLength(""))
		}
		// 测试标点符号
		if runeLength("。，！？") != 4 {
			t.Errorf("标点符号符文长度错误: 期望 4, 得到 %d", runeLength("。，！？"))
		}
	})

	// 测试 splitByLength 函数
	t.Run("TestSplitByLength", func(t *testing.T) {
		// 创建 DocumentSplitter 实例
		splitter := &DocumentSplitter{
			SplitBy: []string{"\f", "\n\n", "\n", "。", "！", "!", "？", "?", ".", ";", "；", "，", ",", " "},
		}

		// 测试短文本（不需要分割）
		text1 := "这是一个短文本。"
		result1 := splitter.splitByLength(text1, 100)
		if len(result1) != 1 || result1[0] != text1 {
			t.Errorf("短文本分割错误: 期望 [%s], 得到 %v", text1, result1)
		}

		// 测试中文文本自然分割
		text2 := "第一句。第二句！第三句？第四句，第五句。"
		result2 := splitter.splitByLength(text2, 10)
		// 检查每块是否在标点处分割
		for _, chunk := range result2 {
			if runeLength(chunk) > 12 {
				t.Errorf("块长度超过限制: %s (长度 %d)", chunk, runeLength(chunk))
			}
		}
		// 检查分割块数
		if len(result2) < 3 || len(result2) > 5 { // 允许一定灵活性
			t.Errorf("分割块数异常: 期望 3-5 块, 得到 %d 块", len(result2))
		}

		// 测试英文文本
		text3 := "This is sentence one. This is sentence two! This is sentence three?"
		result3 := splitter.splitByLength(text3, 20)
		for _, chunk := range result3 {
			if runeLength(chunk) > 30 { // 允许稍长，因为可能找不到标点
				t.Errorf("英文块长度超过限制: %s (长度 %d)", chunk, runeLength(chunk))
			}
		}

		// 测试没有标点的长文本
		text4 := "这是一段没有标点的长文本需要被分割成合适的大小以便处理"
		result4 := splitter.splitByLength(text4, 10)
		if len(result4) == 0 {
			t.Errorf("长文本分割结果为空")
		}
		// 每块长度应该在 10 左右
		for i, chunk := range result4 {
			chunkLen := runeLength(chunk)
			if chunkLen > 20 { // 允许稍长，因为没有标点
				t.Errorf("块 %d 长度 %d 超过最大限制: %s", i, chunkLen, chunk)
			}
		}
	})

	// 测试 mergeSegments 函数
	t.Run("TestMergeSegments", func(t *testing.T) {
		// 测试空片段
		result1 := mergeSegments([]string{}, 100)
		if len(result1) != 0 {
			t.Errorf("空片段合并错误: 期望 [], 得到 %v", result1)
		}

		// 测试小片段合并
		segments2 := []string{"片段1", "片段2", "片段3", "片段4"}
		result2 := mergeSegments(segments2, 20)
		// 应该合并成更少的块
		if len(result2) >= len(segments2) {
			t.Errorf("片段合并无效: 输入 %d 片段, 输出 %d 块", len(segments2), len(result2))
		}
		// 检查每块长度
		for _, chunk := range result2 {
			if runeLength(chunk) > 24 { // 120% of 20
				t.Errorf("合并后块长度超过限制: %s (长度 %d)", chunk, runeLength(chunk))
			}
		}

		// 测试大片段不合并
		segments3 := []string{"这是一个很长的片段，长度超过目标值", "另一个长片段"}
		result3 := mergeSegments(segments3, 10)
		if len(result3) != 2 {
			t.Errorf("大片段错误合并: 期望 2 块, 得到 %d 块", len(result3))
		}
	})

	// 测试 mergeChunks 方法
	t.Run("TestMergeChunks", func(t *testing.T) {
		splitter := &DocumentSplitter{SplitLength: 100}

		// 测试短块合并
		chunks1 := []string{"短块1", "短块2", "长块" + strings.Repeat("内容", 50)}
		result1 := splitter.mergeChunks(chunks1)
		// 应该合并前两个短块
		if len(result1) != 2 {
			t.Errorf("短块合并错误: 期望 2 块, 得到 %d 块", len(result1))
		}

		// 测试已达到长度的块不合并
		chunks2 := []string{strings.Repeat("内容", 60), "短块"}
		result2 := splitter.mergeChunks(chunks2)
		if len(result2) != 2 {
			t.Errorf("已达长度块错误合并: 期望 2 块, 得到 %d 块", len(result2))
		}

		// 测试合并后长度限制
		chunks3 := []string{strings.Repeat("a", 80), strings.Repeat("b", 80)}
		result3 := splitter.mergeChunks(chunks3)
		// 合并后长度 160 > 180 (SplitLength*18/10=180)，应该合并
		if len(result3) != 1 {
			t.Errorf("边界长度合并错误: 期望 1 块, 得到 %d 块", len(result3))
		}

		// 测试合并后长度超过限制不合并
		chunks4 := []string{strings.Repeat("a", 90), strings.Repeat("b", 91)}
		result4 := splitter.mergeChunks(chunks4)
		// 合并后长度 181 > 180，不应该合并
		if len(result4) != 2 {
			t.Errorf("超长合并错误: 期望 2 块, 得到 %d 块", len(result4))
		}
	})

	// 测试 recursiveSplit 方法
	t.Run("TestRecursiveSplit", func(t *testing.T) {
		splitter := &DocumentSplitter{
			SplitBy:     []string{"\f", "\n\n", "\n", "。", "！", "!", "？", "?", ".", ";", "；", "，", ",", " "},
			SplitLength: 5,
		}

		// 测试短文本
		text1 := "这是一个短文本。"
		result1 := splitter.recursiveSplit(text1, 0)
		if len(result1) != 1 || result1[0] != text1 {
			t.Errorf("短文本递归分割错误: 期望 [%s], 得到 %v", text1, result1)
		}

		// 测试带换行的文本
		text2 := "第一段。\n\n第二段。\n\n第三段。"
		result2 := splitter.recursiveSplit(text2, 0)
		// 应该按 \n\n 分割
		if len(result2) != 3 {
			t.Errorf("换行文本分割错误: 期望 3 段, 得到 %d 段", len(result2))
		}

		// 测试超长段落
		text3 := strings.Repeat("这是一个很长的段落，需要被分割。", 10)
		result3 := splitter.recursiveSplit(text3, 0)
		if len(result3) <= 1 {
			t.Errorf("超长段落分割失败: 期望多个块, 得到 %d 块", len(result3))
		}
		// 检查每块长度
		for i, chunk := range result3 {
			chunkLen := runeLength(chunk)
			if chunkLen > splitter.SplitLength*2 {
				t.Errorf("块 %d 长度 %d 超过限制: %s", i, chunkLen, chunk)
			}
		}

		// 测试混合分隔符
		text4 := "第一句。第二句，第三句。第四句！第五句？第六句；第七句."
		result4 := splitter.recursiveSplit(text4, 0)
		// 应该按各种标点分割
		if len(result4) < 5 {
			t.Errorf("混合分隔符分割不足: 期望至少5块, 得到 %d 块", len(result4))
		}
	})

	// 测试完整流程
	t.Run("TestCompleteFlow", func(t *testing.T) {
		splitter := &DocumentSplitter{
			SplitBy:      []string{"\f", "\n\n", "\n", "。", "！", "!", "？", "?", ".", ";", "；", "，", ",", " "},
			SplitLength:  100,
			SplitOverlap: 0,
		}

		// 创建复杂文本
		text := `自然语言处理（Natural Language Processing，NLP）是人工智能和语言学领域的分支学科。

此领域探讨如何处理及运用自然语言；自然语言处理包括多方面和步骤，基本有认知、理解、生成等部分。

自然语言认知和理解是让电脑把输入的语言变成有意思的符号和关系，然后根据目的再处理。

自然语言生成系统则是把计算机数据转化为自然语言。自然语言处理要研制表示语言能力和语言应用的模型，建立计算框架来实现这样的语言模型，提出相应的方法来不断完善这样的语言模型，根据这样的语言模型设计各种实用系统，并探讨这些实用系统的评测技术。`

		// 直接测试 recursiveSplit
		chunks := splitter.recursiveSplit(text, 0)

		if len(chunks) == 0 {
			t.Fatal("分块结果为空")
		}

		// 检查每块长度
		for i, chunk := range chunks {
			chunkLen := runeLength(chunk)
			t.Logf("块 %d 长度: %d", i+1, chunkLen)
			if chunkLen > splitter.SplitLength*2 {
				t.Errorf("块 %d 长度 %d 超过目标长度的两倍 (%d)", i+1, chunkLen, splitter.SplitLength)
			}
		}

		// 验证内容完整性
		allText := strings.Join(chunks, "")
		if runeLength(allText) < runeLength(text)*9/10 { // 允许10%的内容丢失（主要是分隔符）
			t.Errorf("内容丢失过多: 原始长度 %d, 拼接长度 %d", runeLength(text), runeLength(allText))
		}
	})
}
