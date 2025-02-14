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
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/http1/resp"
)

// init 初始化函数
func init() {

	//初始化静态文件
	initStaticFS()

	// 异常页面
	h.GET("/error", funcError)

	// 默认首页
	h.GET("/", funcIndex)

	// agent 页面
	h.GET("/agent/:agentID", funcAgentPre)

	// 兼容OpenAI模型接口,api_key是agentID,user是roomID
	h.POST("/v1/chat/completions", funcChatCompletions)
}

// funcIndex 模板首页
func funcIndex(ctx context.Context, c *app.RequestContext) {
	data := warpRequestMap(c)
	cHtml(c, http.StatusOK, "index.html", data)
}

// funcError 错误页面
func funcError(ctx context.Context, c *app.RequestContext) {
	cHtml(c, http.StatusOK, "error.html", nil)
}

// funcAgentPre 智能体
func funcAgentPre(ctx context.Context, c *app.RequestContext) {
	data := warpRequestMap(c)
	agentID := c.Param("agentID")
	data["agentID"] = agentID
	cHtml(c, http.StatusOK, "agent.html", data)
}

// funcChatCompletions 兼容OpenAI模型接口,api_key是agentID,user是roomID
func funcChatCompletions(ctx context.Context, c *app.RequestContext) {

	// 设置响应头
	c.SetStatusCode(http.StatusOK)

	accept := string(c.GetHeader("Accept"))
	stream := strings.Contains(strings.ToLower(accept), "text/event-stream")

	aByte := c.GetHeader("Authorization")
	if len(aByte) < 1 {
		if stream {
			c.WriteString("data: Authorization is empty\n\n")
			c.Flush()
			c.WriteString("data: [DONE]\n\n")
			c.Flush()
		} else {
			c.WriteString("Authorization is empty")
			c.Flush()
		}
		c.Abort()
		return
	}
	authorization := string(aByte)
	agentID := strings.TrimPrefix(authorization, "Bearer ")
	if agentID == "" {
		if stream {
			c.WriteString("data: Authorization is empty\n\n")
			c.Flush()
			c.WriteString("data: [DONE]\n\n")
			c.Flush()
		} else {
			c.WriteString("Authorization is empty")
			c.Flush()
		}
		c.Abort()
		return
	}

	agentRequestBody := &AgentRequestBody{}
	err := c.BindJSON(agentRequestBody)
	if err != nil {
		if stream {
			c.WriteString(fmt.Sprintf("data: body is error:%v\n\n", err))
			c.Flush()
			c.WriteString("data: [DONE]\n\n")
			c.Flush()
		} else {
			c.WriteString(fmt.Sprintf("body is error:%v", err))
			c.Flush()
		}
	}
	if len(agentRequestBody.Messages) < 1 {
		if stream {
			c.WriteString("data: messages is empty\n\n")
			c.Flush()
			c.WriteString("data: [DONE]\n\n")
			c.Flush()
		} else {
			c.WriteString("messages is empty")
			c.Flush()
		}
		c.Abort()
		return
	}
	input := make(map[string]interface{}, 0)
	// 用户发送的第一个消息
	input["query"] = agentRequestBody.Messages[0].Content
	// agentID
	input["agentID"] = agentID
	// 获取roomID,可能会空
	roomID := agentRequestBody.User
	if roomID != "" && len(roomID) == 32 {
		input["roomID"] = roomID
	}

	if stream {
		c.Header("Accept", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		writer := resp.NewChunkedBodyWriter(&c.Response, c.GetWriter())
		c.Response.HijackWriter(writer)
	}
	input["c"] = c

	agent, err := findAgentByID(ctx, agentID)
	if err != nil || agent.Id == "" {
		if stream {
			c.WriteString("data: agent is empty\n\n")
			c.Flush()
			c.WriteString("data: [DONE]\n\n")
			c.Flush()
		} else {
			c.WriteString("agent is empty")
			c.Flush()
		}
		c.Abort()
		return
	}

	input["knowledgeBaseID"] = agent.KnowledgeBaseID
	pipeline := componentMap[agent.PipelineID]
	pipeline.Run(ctx, input)
	//choice := input["choice"]
	errObj := input[errorKey]
	if errObj != nil {
		if stream {
			c.WriteString(fmt.Sprintf("data: component run is error:%v\n\n", errObj))
			c.Flush()
			c.WriteString("data: [DONE]\n\n")
			c.Flush()
		} else {
			c.WriteString(fmt.Sprintf("component run is error:%v", errObj))
			c.Flush()
		}
		c.Abort()
		return
	}

	//fmt.Println(choice)
	//c.JSON(http.StatusOK, ResponseData{StatusCode: 1, Data: choice})
}

// warpRequestMap 包装请求参数为map
func warpRequestMap(c *app.RequestContext) map[string]interface{} {
	data := make(map[string]interface{}, 0)
	jwttoken := string(c.Cookie(config.JwttokenKey))
	userId, _ := userIdByToken(jwttoken)
	if userId != "" {
		data[userTypeKey] = 1
	} else {
		data[userTypeKey] = 0
	}
	return data
}
