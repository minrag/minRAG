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
	"errors"
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
	h.GET("/page/:pageNo", funcIndex)
	h.GET("/page/:pageNo/", funcIndex)

	// 查看标签
	h.GET("/tag/:urlPathParam", funcListTags)
	h.GET("/tag/:urlPathParam/", funcListTags)
	h.GET("/tag/:urlPathParam/page/:pageNo", funcListTags)
	h.GET("/tag/:urlPathParam/page/:pageNo/", funcListTags)

	//初始化知识库路由
	initKnowledgeBaseRoute()

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

// funcListKnowledge知识库 知识库数据列表
func funcListKnowledgeBase(ctx context.Context, c *app.RequestContext) {
	data := warpRequestMap(c)
	urlPathParam := c.Param("urlPathParam")
	if urlPathParam == "" { //知识库路径访问,例如:/web
		urlPathParam = c.GetString("urlPathParam")
	}
	data["UrlPathParam"] = urlPathParam
	templateFile, err := findThemeTemplate(ctx, tableKnowledgeBaseName, urlPathParam)
	if err != nil || templateFile == "" {
		templateFile = "knowledgeBase.html"
	}
	cHtml(c, http.StatusOK, templateFile, data)
}

// funcListTags 标签列表
func funcListTags(ctx context.Context, c *app.RequestContext) {
	data := warpRequestMap(c)
	urlPathParam := c.Param("urlPathParam")

	data["UrlPathParam"] = urlPathParam
	cHtml(c, http.StatusOK, "tag.html", data)
}

// funcOneDocument 查询一篇文章
func funcOneDocument(ctx context.Context, c *app.RequestContext) {
	data := warpRequestMap(c)
	urlPathParam := c.Param("urlPathParam")
	if urlPathParam == "" { //知识库路径访问,例如:/web/nginx-use-hsts
		urlPathParam = c.GetString("urlPathParam")
	}
	data["UrlPathParam"] = urlPathParam

	templateFile, err := findThemeTemplate(ctx, tableDocumentName, urlPathParam)
	if err != nil || templateFile == "" {
		templateFile = "document.html"
	}
	cHtml(c, http.StatusOK, templateFile, data)
}

// initKnowledgeBaseRout知识库化知识库的映射路径
func initKnowledgeBaseRoute() {
	categories, _ := findAllKnowledgeBase(context.Background())
	for i := 0; i < len(categories); i++ {
		knowledgeBase := categories[i]
		knowledgeBaseID := knowledgeBase.Id
		addKnowledgeBaseRoute(knowledgeBaseID)
	}
}

// addKnowledgeBaseRou知识库加知识库的路由
func addKnowledgeBaseRoute(knowledgeBaseID string) {

	// 处理重复注册路由的panic,不对外抛出
	defer func() {
		if r := recover(); r != nil {
			panicMessage := fmt.Sprintf("%s", r)
			FuncLogPanic(nil, errors.New(panicMessage))
		}
	}()

	//知识库的访问映射
	h.GET(funcTrimSuffixSlash(knowledgeBaseID), addListKnowledgeBaseRoute(knowledgeBaseID))
	h.GET(knowledgeBaseID, addListKnowledgeBaseRoute(knowledgeBaseID))
	//知识库分页数据的访问映射
	h.GET(knowledgeBaseID+"page/:pageNo", addListKnowledgeBaseRoute(knowledgeBaseID))
	h.GET(knowledgeBaseID+"page/:pageNo/", addListKnowledgeBaseRoute(knowledgeBaseID))
	//知识库下文章的访问映射
	h.GET(knowledgeBaseID+":documentURI", addOneDocumentRoute(knowledgeBaseID))
	h.GET(knowledgeBaseID+":documentURI/", addOneDocumentRoute(knowledgeBaseID))
}

// addListKnowledgeBaseRou知识库加知识库的GET请求路由,用于自定义设置知识库的路由
func addListKnowledgeBaseRoute(knowledgeBaseID string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.Set("urlPathParam", knowledgeBaseID)
		funcListKnowledgeBase(ctx, c)
	}
}

// addOneDocumentRoute 增加内容的GET请求路由
func addOneDocumentRoute(knowledgeBaseID string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		documentURI := c.Param("documentURI")
		key := knowledgeBaseID + documentURI
		c.Set("urlPathParam", key)
		funcOneDocument(ctx, c)
	}
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
