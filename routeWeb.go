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
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
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

	input := make(map[string]interface{}, 0)
	input["c"] = c
	input["query"] = "你在哪里?"
	documentChunks := make([]DocumentChunk, 3)
	documentChunks[0] = DocumentChunk{Markdown: "我在郑州"}
	documentChunks[1] = DocumentChunk{Markdown: "今天晴天"}
	documentChunks[2] = DocumentChunk{Markdown: "我明天去旅游"}
	input["documentChunks"] = documentChunks

	agentID := c.Param("agentID")
	agent, _ := findAgentByID(ctx, agentID)
	pipeline := componentMap[agent.PipelineID]
	pipeline.Run(ctx, input)
	choice := input["choice"]
	err := input[errorKey]
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusOK, ResponseData{StatusCode: 1, Data: choice, ERR: err.(error)})
		return
	}

	c.JSON(http.StatusOK, ResponseData{StatusCode: 1, Data: choice})
}

// warpRequestMap 包装请求参数为map
func warpRequestMap(c *app.RequestContext) map[string]interface{} {
	pageNoStr := c.Param("pageNo")
	if pageNoStr == "" {
		pageNoStr = c.GetString("pageNo")
	}
	if pageNoStr == "" {
		//pageNoStr = c.DefaultQuery("pageNo", "1")
		pageNoStr = "1"
	}

	pageNo, _ := strconv.Atoi(pageNoStr)
	q := strings.TrimSpace(c.Query("q"))

	data := make(map[string]interface{}, 0)
	data["pageNo"] = pageNo
	data["q"] = q
	//设置用户角色,0是访客,1是管理员
	userType, ok := c.Get(userTypeKey)
	if ok {
		data[userTypeKey] = userType
	} else {
		data[userTypeKey] = 0
	}
	return data
}
