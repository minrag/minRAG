package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

/**
https://github.com/Tencent/WeKnora

internal/application/service/extract.go 中 ChunkExtractService.Extract 实现使用提示词抽取实体和关系,rebuildGraph整理模型返回的实体关系数据.   TestGraph1
internal/application/service/chat_pipline/extract_entity.go 中 PluginExtractEntity 完成 根据用户的query 抽取知识图谱的实体,用于查询知识图谱 TestGraph2
internal/application/repository/retriever/neo4j/repository.go 完成对知识图谱数据的增删改查


// 文档的chunk.ID 和 知识图谱的节点绑定,用于检索出实体节点时,关联查询出对应的文档chunk内容
for _, node := range graph.Node {
		node.Chunks = []string{chunk.ID}
	}

// 使用 知识库ID-文档ID作为NameSpace,多个文档就多次查询,然后再聚合结果
s.graphEngine.AddGraph(ctx,
		types.NameSpace{KnowledgeBase: chunk.KnowledgeBaseID, Knowledge: chunk.KnowledgeID},
		[]*types.GraphData{graph},
	)

**/

type GraphEntity struct {
	Entity           string   `json:"entity,omitempty"`
	EntityAttributes []string `json:"entity_attributes,omitempty"`
	Entity1          string   `json:"entity1,omitempty"`
	Entity2          string   `json:"entity2,omitempty"`
	Relation         string   `json:"relation,omitempty"`
}
type GraphData struct {
	Text     string          `json:"text,omitempty"`
	Node     []GraphNode     `json:"node,omitempty"`
	Relation []GraphRelation `json:"relation,omitempty"`
}

type GraphNode struct {
	Name       string   `json:"name,omitempty"`
	Chunks     []string `json:"chunks,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
}
type GraphRelation struct {
	Node1 string `json:"node1,omitempty"`
	Node2 string `json:"node2,omitempty"`
	Type  string `json:"type,omitempty"`
}

const api_key = "A4FTACZVPGAIV8PZCKIBEUGV7ZBMXTIBEGUGNC11"
const api_url = "https://ai.gitee.com/v1/chat/completions"
const model_name = "DeepSeek-V3.2"

const demoStr = `公元2023年秋，北京中关村某人工智能实验室，研究员林远与项目负责人苏雯在“时空图神经网络”（TGNN）研发中形成紧密协作关系。该事件聚焦于构建基于时间序列的动态图模型，以提升城市交通流量预测的准确率。林远作为核心算法设计者，负责属性嵌入与时间感知注意力机制的实现；苏雯则主导系统集成与跨模态数据融合。实验内容涉及对10万条GPS轨迹数据的时空特征提取，采用图卷积网络（GCN）与LSTM的混合架构，实现对高峰时段拥堵传播路径的建模。该成果发表于IEEE TKDE期刊，作者署名为Lin Y.与Su W.，标志着在时空数据挖掘领域的重要进展。`

// 直接生成GraphData结构效果不好,需要先生成GraphEntity,再转换为GraphData
func TestGraph1(t *testing.T) {

	str1 := `请基于给定文本，按以下步骤完成信息提取任务，确保逻辑清晰、信息完整准确：

      ## 一、实体提取与属性补充
      1. **提取核心实体**：通读文本，按逻辑顺序（如文本叙述顺序、实体关联紧密程度）提取所有与任务相关的核心实体。
      2. **补充实体详细属性**：针对每个提取的实体，全面补充其在文本中明确提及的详细属性，确保无关键属性遗漏。

      ## 二、关系提取与验证
      1. **明确关系类型**：仅从指定关系列表中选择对应类型，限定关系类型为: %s。
      2. **提取有效关系**：基于已提取的实体及属性，识别文本中真实存在的关系，确保关系符合文本事实、无虚假关联。
      3. **明确关系主体**：对每一组提取的关系，清晰标注两个关联主体，避免主体混淆。
      4. **补充关联属性**：若文本中存在与该关系直接相关的补充信息，需将该信息作为关系的关联属性补充，进一步完善关系信息。
	  
    # Examples Question Q: %s 
	A: %s

	`
	tags := []string{"时间", "地点", "关系", "属性", "作者", "事件", "人物", "内容"}
	tagsByte, _ := json.Marshal(tags)

	// 实体和关系
	graph := []GraphEntity{
		//实体
		{Entity: "时空图神经网络", EntityAttributes: []string{"简称 TGNN", "用于构建基于时间序列的动态图模型", "目标是提升城市交通流量预测的准确率"}},
		{Entity: "林远", EntityAttributes: []string{"研究员", "核心算法设计者", "负责属性嵌入与时间感知注意力机制的实现", "作者署名为 Lin Y."}},
		{Entity: "苏雯", EntityAttributes: []string{"项目负责人", "主导系统集成与跨模态数据融合", "作者署名为 Su W."}},
		{Entity: "北京中关村某人工智能实验室", EntityAttributes: []string{"事件发生地点", "位于北京市"}},
		{Entity: "IEEE TKDE期刊", EntityAttributes: []string{"论文发表期刊", "在时空数据挖掘领域具有重要影响力"}},
		{Entity: "10万条GPS轨迹数据", EntityAttributes: []string{"实验所用数据集", "用于时空特征提取"}},
		{Entity: "图卷积网络（GCN）与LSTM的混合架构", EntityAttributes: []string{"采用的模型架构", "用于建模高峰时段拥堵传播路径"}},
		{Entity: "2023年秋"},
		{Entity: "城市交通流量预测"},
		//关系
		{Entity1: "林远", Entity2: "苏雯", Relation: "关系"},
		{Entity1: "时空图神经网络", Entity2: "林远", Relation: "作者"},
		{Entity1: "时空图神经网络", Entity2: "苏雯", Relation: "作者"},
		{Entity1: "时空图神经网络", Entity2: "北京中关村某人工智能实验室", Relation: "地点"},
		{Entity1: "时空图神经网络", Entity2: "2023年秋", Relation: "时间"},
		{Entity1: "时空图神经网络", Entity2: "10万条GPS轨迹数据", Relation: "内容"},
		{Entity1: "时空图神经网络", Entity2: "图卷积网络（GCN）与LSTM的混合架构", Relation: "内容"},
	}
	// 直接返回[]数组效果不好,需要包装一下,返回对象
	answer := map[string]any{"answer": graph}
	graphByte, _ := json.Marshal(answer)
	// 系统提示词,用于做模板例子
	systemPrompt := fmt.Sprintf(str1, string(tagsByte), demoStr, string(graphByte))
	//fmt.Println(systemPrompt)

	//构建自己的内容,让模型抽取实体关系
	userPrompt := `# Question Q: WeKnora（维娜拉） 是一款基于大语言模型（LLM）的文档理解与语义检索框架，专为结构复杂、内容异构的文档场景而打造。 框架采用模块化架构，融合多模态预处理、语义向量索引、智能召回与大模型生成推理，构建起高效、可控的文档问答流程。核心检索流程基于 RAG（Retrieval-Augmented Generation） 机制，将上下文相关片段与语言模型结合，实现更高质量的语义回答。\n A:`

	bodyMap := make(map[string]any)
	bodyMap["messages"] = []ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}}
	bodyMap["model"] = model_name

	bodyMap["response_format"] = map[string]string{"type": "json_object"}
	//输出类型
	bodyMap["stream"] = false

	//请求大模型
	client := &http.Client{
		Timeout: time.Second * time.Duration(500),
	}
	bodyByte, err := httpPostJsonBody(client, api_key, api_url, nil, bodyMap)
	if err != nil {
		t.Error(err)
	}
	rs := struct {
		Choices []Choice `json:"choices,omitempty"`
	}{}
	err = json.Unmarshal(bodyByte, &rs)
	if err != nil {
		t.Error(err)
	}

	//获取第一个结果
	resultJson := rs.Choices[0].Message.Content

	result := struct {
		Answer []GraphEntity `json:"answer,omitempty"`
	}{}
	err = json.Unmarshal([]byte(resultJson), &result)

	fmt.Println("resultJson:", resultJson)

}

// 直接生成GraphData结构效果不好,需要先生成GraphEntity,再转换为GraphData
// 根据用户问题,获取知识图谱的实体
func TestGraph2(t *testing.T) {

	str1 := `请基于用户给的问题，按以下步骤处理关键信息提取任务：
      1. 梳理逻辑关联：首先完整分析文本内容，明确其核心逻辑关系，并简要标注该核心逻辑类型；
      2. 提取关键实体：围绕梳理出的逻辑关系，精准提取文本中的关键信息并归类为明确实体，确保不遗漏核心信息、不添加冗余内容；
      3. 排序实体优先级：按实体与文本核心主题的关联紧密程度排序，优先呈现对理解文本主旨最重要的实体；

    # Examples Question Q: %s 
	A: %s
	`
	// 实体和关系
	graph := []GraphEntity{
		//实体
		{Entity: "时空图神经网络"},
		{Entity: "林远"},
		{Entity: "苏雯"},
		{Entity: "北京中关村某人工智能实验室"},
		{Entity: "IEEE TKDE期刊"},
		{Entity: "10万条GPS轨迹数据"},
		{Entity: "图卷积网络（GCN）与LSTM的混合架构"},
		{Entity: "2023年秋"},
		{Entity: "城市交通流量预测"},
	}
	// 直接返回[]数组效果不好,需要包装一下,返回对象
	answer := map[string]any{"answer": graph}
	graphByte, _ := json.Marshal(answer)

	// 系统提示词,用于做模板例子
	systemPrompt := fmt.Sprintf(str1, demoStr, string(graphByte))
	//fmt.Println(systemPrompt)

	//构建自己的内容,让模型抽取实体关系
	userPrompt := `# Question Q: 使用知识图谱回答我 ,WeKnora 是什么? \n A:`

	bodyMap := make(map[string]any)
	bodyMap["messages"] = []ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}}
	bodyMap["model"] = model_name

	bodyMap["response_format"] = map[string]string{"type": "json_object"}
	//输出类型
	bodyMap["stream"] = false

	//请求大模型
	client := &http.Client{
		Timeout: time.Second * time.Duration(500),
	}
	bodyByte, err := httpPostJsonBody(client, api_key, api_url, nil, bodyMap)
	if err != nil {
		t.Error(err)
	}
	rs := struct {
		Choices []Choice `json:"choices,omitempty"`
	}{}
	err = json.Unmarshal(bodyByte, &rs)
	if err != nil {
		t.Error(err)
	}

	//获取第一个结果
	resultJson := rs.Choices[0].Message.Content

	result := GraphData{}
	err = json.Unmarshal([]byte(resultJson), &result)

	fmt.Println("resultJson:", resultJson)

}
