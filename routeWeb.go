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
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitee.com/chunanyong/zorm"
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

	// 查看agent
	h.GET("/agent/:agentID", funcAgentPre)
	h.POST("/agent/sse", funcAgentSSE)
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

// funcAgent 智能体
func funcAgentSSE(ctx context.Context, c *app.RequestContext) {
	input := make(map[string]interface{}, 0)
	c.BindJSON(&input)

	// 设置响应头
	c.SetStatusCode(http.StatusOK)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	writer := resp.NewChunkedBodyWriter(&c.Response, c.GetWriter())
	c.Response.HijackWriter(writer)
	input["c"] = c

	agentIDObj, has := input["agentID"]
	if !has || agentIDObj == nil || agentIDObj.(string) == "" {
		c.WriteString("data: agentID is empty\n\n")
		c.Flush()
		c.WriteString("data: [DONE]\n\n")
		c.Flush()
		c.Abort()
		return
	}
	agentID := agentIDObj.(string)
	agent, err := findAgentByID(ctx, agentID)
	if err != nil || agent.Id == "" {
		c.WriteString("data: agent is empty\n\n")
		c.Flush()
		c.WriteString("data: [DONE]\n\n")
		c.Flush()
		c.Abort()
		return
	}

	roomIDObj, has := input["roomID"]
	if !has || roomIDObj.(string) == "" {
		c.WriteString("data: roomID is empty\n\n")
		c.Flush()
		c.WriteString("data: [DONE]\n\n")
		c.Flush()
		c.Abort()
		return
	}
	roomID := roomIDObj.(string)
	roomIDs := strings.Split(roomID, "_")
	if len(roomIDs) != 2 || len(roomIDs[0]) != 20 {
		c.WriteString("data: roomID is error\n\n")
		c.Flush()
		c.WriteString("data: [DONE]\n\n")
		c.Flush()
		c.Abort()
		return
	}

	input["knowledgeBaseID"] = agent.KnowledgeBaseID
	pipeline := componentMap[agent.PipelineID]
	pipeline.Run(ctx, input)
	//choice := input["choice"]
	errObj := input[errorKey]
	if errObj != nil {
		c.WriteString(fmt.Sprintf("data: component run is error:%v\n\n", errObj))
		c.Flush()
		c.Abort()
		return
	}
	choice := Choice{}
	choiceObj, has := input["choice"]
	if has && choiceObj != nil {
		choice = choiceObj.(Choice)
	}
	jwttoken := string(c.Cookie(config.JwttokenKey))
	userId, _ := userIdByToken(jwttoken)

	now := time.Now().Format("2006-01-02 15:04:05")
	query := input["query"].(string)
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

	zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		if chatRoom.Id == "" {
			chatRoom.Id = messageLog.RoomID
			zorm.Insert(ctx, chatRoom)
		}
		zorm.Insert(ctx, messageLog)

		return nil, nil
	})

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
