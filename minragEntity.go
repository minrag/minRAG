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

	"gitee.com/chunanyong/zorm"
)

// Config 配置表
type Config struct {
	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// ID 主键
	Id string `column:"id" json:"id,omitempty"`

	// BasePath 根路径,默认/
	BasePath string `column:"basePath" json:"basePath,omitempty"`

	// JwtSecret jwt加密密钥
	JwtSecret string `column:"jwtSecret" json:"jwtSecret,omitempty"`

	// JwttokenKey jwt加密密钥
	JwttokenKey string `column:"jwttokenKey" json:"jwttokenKey,omitempty"`

	// ServerPort 服务器端口
	ServerPort string `column:"serverPort" json:"serverPort,omitempty"`

	// Timeout 超时时间,单位秒
	Timeout int `column:"timeout" json:"timeout,omitempty"`

	// MaxRequestBodySize 最大请求
	MaxRequestBodySize int `column:"maxRequestBodySize" json:"maxRequestBodySize,omitempty"`

	// Locale 语言包
	Locale string `column:"locale" json:"locale,omitempty"`

	// Proxy 代理,用于翻墙 格式: http://127.0.0.1:8090
	Proxy string `column:"proxy" json:"proxy,omitempty"`

	// AIBaseURL AI平台base_url
	AIBaseURL string `column:"aiBaseURL" json:"aiBaseURL,omitempty"`

	// AIAPIkey AI平台api_key
	AIAPIkey string `column:"aiAPIKey" json:"aiAPIKey,omitempty"`

	// LLMModel 默认的LLM模型
	LLMModel string `column:"llmModel" json:"llmModel,omitempty"`

	// CreateTime 创建时间
	CreateTime string `column:"createTime" json:"createTime,omitempty"`

	// UpdateTime 更新时间
	UpdateTime string `column:"updateTime" json:"updateTime,omitempty"`

	// CreateUser 创建人,初始化 system
	CreateUser string `column:"createUser" json:"createUser,omitempty"`

	// SortNo 排序
	SortNo int `column:"sortNo" json:"sortNo,omitempty"`

	// Status 状态 禁用(0),可用(1)
	Status int `column:"status" json:"status,omitempty"`
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *Config) GetTableName() string {
	return tableConfigName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
// 如果没有主键,也需要实现这个方法, return "" 即可
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *Config) GetPKColumnName() string {
	return "id"
}

// KnowledgeBase 知识库
type KnowledgeBase struct {
	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// ID
	Id string `column:"id" json:"id,omitempty"`

	// Name 名称
	Name string `column:"name" json:"name,omitempty"`

	// Pid 父级ID
	Pid string `column:"pid" json:"pid,omitempty"`

	// KnowledgeBaseType 知识库类型
	KnowledgeBaseType int `column:"knowledgeBaseType" json:"knowledgeBaseType,omitempty"`

	// CreateTime 创建时间
	CreateTime string `column:"createTime" json:"createTime,omitempty"`

	// UpdateTime 更新时间
	UpdateTime string `column:"updateTime" json:"updateTime,omitempty"`

	// CreateUser 创建人,初始化 system
	CreateUser string `column:"createUser" json:"createUser,omitempty"`

	// SortNo 排序
	SortNo int `column:"sortNo" json:"sortNo,omitempty"`

	// Status 状态 禁用(0),可用(1)
	Status int `column:"status" json:"status,omitempty"`

	Leaf []*KnowledgeBase `json:"leaf,omitempty"`
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *KnowledgeBase) GetTableName() string {
	return tableKnowledgeBaseName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
// 如果没有主键,也需要实现这个方法, return "" 即可
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *KnowledgeBase) GetPKColumnName() string {
	return "id"
}

// Document 文档
type Document struct {

	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// ID
	Id string `column:"id" json:"id,omitempty"`

	// Name 文章标题
	Name string `column:"name" json:"name,omitempty"`

	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `column:"knowledgeBaseID" json:"knowledgeBaseID,omitempty"`

	// KnowledgeBaseName 知识库
	KnowledgeBaseName string `column:"knowledgeBaseName" json:"knowledgeBaseName,omitempty"`

	// Toc 目录
	Toc string `column:"toc" json:"toc,omitempty"`

	// Summary 摘要
	Summary string `column:"summary" json:"summary,omitempty"`

	// Markdown Markdown内容
	Markdown string `column:"markdown" json:"markdown,omitempty"`

	// FilePath 上传的文件路径
	FilePath string `column:"filePath" json:"filePath,omitempty"`

	// FileSize 上传的文件大小,单位k
	FileSize int `column:"fileSize" json:"fileSize,omitempty"`

	// FileExt 文档后缀
	FileExt string `column:"fileExt" json:"fileExt,omitempty"`

	// CreateTime 创建时间
	CreateTime string `column:"createTime" json:"createTime,omitempty"`

	// UpdateTime 更新时间
	UpdateTime string `column:"updateTime" json:"updateTime,omitempty"`

	// CreateUser 创建人,初始化 system
	CreateUser string `column:"createUser" json:"createUser,omitempty"`

	// SortNo 排序
	SortNo int `column:"sortNo" json:"sortNo,omitempty"`

	// Status 状态 禁用(0),可用(1),处理中(2),处理失败(3)
	Status int `column:"status" json:"status,omitempty"`
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *Document) GetTableName() string {
	return tableDocumentName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
// 如果没有主键,也需要实现这个方法, return "" 即可
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *Document) GetPKColumnName() string {
	return "id"
}

// DocumentTOC 生成文档TOC目录
type DocumentTOC struct {
	// ID
	Id    string         `column:"id" json:"id,omitempty"`
	Title string         `column:"title" json:"title,omitempty"` // 标题
	Nodes []*DocumentTOC `json:"nodes,omitempty"`                // 子节点
}

// DocumentChunk 文档分块
type DocumentChunk struct {

	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// ID
	Id string `column:"id" json:"id,omitempty"`

	// DocumentID 文档ID
	DocumentID string `column:"documentID" json:"documentID,omitempty"`

	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `column:"knowledgeBaseID" json:"knowledgeBaseID,omitempty"`

	Title    string `column:"title" json:"title,omitempty"`       // 标题
	ParentID string `column:"parentID" json:"parentID,omitempty"` // 上级ID
	PreID    string `column:"preID" json:"preID,omitempty"`       // 上一个节点ID
	NextID   string `column:"nextID" json:"nextID,omitempty"`     // 下一个节点ID
	Level    int    `column:"level" json:"level,omitempty"`       // 标题级别

	// Markdown Markdown内容
	Markdown string `column:"markdown" json:"markdown,omitempty"`

	// CreateTime 创建时间
	CreateTime string `column:"createTime" json:"createTime,omitempty"`

	// UpdateTime 更新时间
	UpdateTime string `column:"updateTime" json:"updateTime,omitempty"`

	// CreateUser 创建人,初始化 system
	CreateUser string `column:"createUser" json:"createUser,omitempty"`

	// SortNo 排序
	SortNo int `column:"sortNo" json:"sortNo,omitempty"`

	// Status 状态 禁用(0),可用(1),处理中(2),处理失败(3)
	Status int `column:"status" json:"status,omitempty"`

	//----------数据库字段结束-----------//
	// Embedding markdown向量化二进制
	Embedding []byte `json:"embedding,omitempty"`
	// RowID 默认的rowid字段
	RowID int `json:"rowID,omitempty"`
	// Score 向量表的score匹配分数
	Score float32 `json:"score,omitempty"`

	Nodes []*DocumentChunk `json:"nodes,omitempty"` // 子节点
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *DocumentChunk) GetTableName() string {
	return tableDocumentChunkName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
// 如果没有主键,也需要实现这个方法, return "" 即可
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *DocumentChunk) GetPKColumnName() string {
	return "id"
}

// VecDocumentChunk 向量化的数据表
type VecDocumentChunk struct {

	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// ID
	Id string `column:"id" json:"id,omitempty"`

	// DocumentID 文档ID
	DocumentID string `column:"documentID" json:"documentID,omitempty"`

	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `column:"knowledgeBaseID" json:"knowledgeBaseID,omitempty"`

	// Markdown Markdown内容
	Markdown string `json:"markdown,omitempty"`

	// Embedding markdown向量化二进制
	Embedding []byte `column:"embedding" json:"embedding,omitempty"`

	// SortNo 排序
	SortNo int `column:"sortNo" json:"sortNo,omitempty"`

	// Status 状态 禁用(0),可用(1),处理中(2),处理失败(3)
	Status int `column:"status" json:"status,omitempty"`

	// RowID 默认的rowid字段
	RowID int `json:"rowID,omitempty"`
	// Score 向量表的score匹配分数
	Score float32 `json:"score,omitempty"`
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *VecDocumentChunk) GetTableName() string {
	return tableVecDocumentChunkName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
// 如果没有主键,也需要实现这个方法, return "" 即可
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *VecDocumentChunk) GetPKColumnName() string {
	return "id"
}

// Component 组件表
type Component struct {

	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// Id 组件ID,可以根据ID获取声明的公共组件,也可以是自定义组件
	Id string `column:"id" json:"id,omitempty"`

	// ComponentType 组件类型,和componentTypeMap关联
	ComponentType string `column:"componentType" json:"componentType,omitempty"`

	// Parameter 参数,json格式字符串.如果有值,必须是完整的参数,为空可用只保留id,从map中获取
	Parameter string `column:"parameter" json:"parameter,omitempty"`

	// RunExpression 运行表达式,组件运行时先验证表达式是否通过,可以为空. 例如 "{{.size}}>100"
	RunExpression string `json:"runExpression,omitempty"`

	// 流水线里的所有组件都放到一个map<Id,Component>,可以根据ID获取单例,避免使用指针,因为每个流水线的组件要互相隔离
	// UpStream 上游组件,必须上游组件都执行完成后,才会执行当前组件.默认为空,只有一个上游时,可以为空
	UpStream []*Component `json:"upstream,omitempty"`

	// DownStream 下游组件,多个节点时,一般指定runExpression,同时执行多个下游节点
	DownStream []*Component `json:"downstream,omitempty"`

	// CreateTime 创建时间
	CreateTime string `column:"createTime" json:"createTime,omitempty"`

	// UpdateTime 更新时间
	UpdateTime string `column:"updateTime" json:"updateTime,omitempty"`

	// CreateUser 创建人,初始化 system
	CreateUser string `column:"createUser" json:"createUser,omitempty"`

	// SortNo 排序
	SortNo int `column:"sortNo" json:"sortNo,omitempty"`

	// Status 状态 禁用(0),可用(1)
	Status int `column:"status" json:"status,omitempty"`
}

// Initialization 初始化方法
func (entity *Component) Initialization(ctx context.Context, input map[string]interface{}) error {
	return nil
}

// Run 执行方法
func (entity *Component) Run(ctx context.Context, input map[string]interface{}) error {
	return nil
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *Component) GetTableName() string {
	return tableComponentName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
// 如果没有主键,也需要实现这个方法, return "" 即可
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *Component) GetPKColumnName() string {
	return "id"
}

// Agent 智能体
type Agent struct {

	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// ID
	Id string `column:"id" json:"id,omitempty"`

	// Name 智能体名称
	Name string `column:"name" json:"name,omitempty"`

	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `column:"knowledgeBaseID" json:"knowledgeBaseID,omitempty"`

	// PipelineID 流水线ID
	PipelineID string `column:"pipelineID" json:"pipelineID,omitempty"`

	// DefaultReply 默认回复
	DefaultReply string `column:"defaultReply" json:"defaultReply,omitempty"`

	// AgentType 智能体类型
	AgentType int `column:"agentType" json:"agentType,omitempty"`

	// AgentPrompt 智能体的提示词
	AgentPrompt string `column:"agentPrompt" json:"agentPrompt,omitempty"`

	// Avatar 头像
	Avatar string `column:"avatar" json:"avatar,omitempty"`

	// Welcome 欢迎语
	Welcome string `column:"welcome" json:"welcome,omitempty"`

	// Tools tools调用的函数名称
	Tools string `column:"tools" json:"tools,omitempty"`

	// MemoryLength 上下文记忆的长度
	MemoryLength int `column:"memoryLength" json:"memoryLength,omitempty"`

	// CreateTime 创建时间
	CreateTime string `column:"createTime" json:"createTime,omitempty"`

	// UpdateTime 更新时间
	UpdateTime string `column:"updateTime" json:"updateTime,omitempty"`

	// CreateUser 创建人,初始化 system
	CreateUser string `column:"createUser" json:"createUser,omitempty"`

	// SortNo 排序
	SortNo int `column:"sortNo" json:"sortNo,omitempty"`

	// Status 状态 禁用(0),可用(1)
	Status int `column:"status" json:"status,omitempty"`
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *Agent) GetTableName() string {
	return tableAgentName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
// 如果没有主键,也需要实现这个方法, return "" 即可
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *Agent) GetPKColumnName() string {
	return "id"
}

// ChatRoom 聊天室
type ChatRoom struct {

	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// ID
	Id string `column:"id" json:"id,omitempty"`

	// RoomID 聊天室名称
	Name string `column:"name" json:"name,omitempty"`

	// AgentID 智能体ID
	AgentID string `column:"agentID" json:"agentID,omitempty"`

	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `column:"knowledgeBaseID" json:"knowledgeBaseID,omitempty"`

	// PipelineID 流水线ID
	PipelineID string `column:"pipelineID" json:"pipelineID,omitempty"`

	// UserID 用户ID
	UserID string `column:"userID" json:"userID,omitempty"`

	// CreateTime 创建时间
	CreateTime string `column:"createTime" json:"createTime,omitempty"`
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *ChatRoom) GetTableName() string {
	return tableChatRoomName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
// 如果没有主键,也需要实现这个方法, return "" 即可
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *ChatRoom) GetPKColumnName() string {
	return "id"
}

// MessageLog 消息日志
type MessageLog struct {

	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// ID
	Id string `column:"id" json:"id,omitempty"`

	// AgentID 智能体ID
	AgentID string `column:"agentID" json:"agentID,omitempty"`

	// RoomID 聊天室ID
	RoomID string `column:"roomID" json:"roomID,omitempty"`

	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `column:"knowledgeBaseID" json:"knowledgeBaseID,omitempty"`

	// PipelineID 流水线ID
	PipelineID string `column:"pipelineID" json:"pipelineID,omitempty"`

	// UserMessage 用户发送的消息
	UserMessage string `column:"userMessage" json:"userMessage,omitempty"`

	// AIMessage AI发送的信息
	AIMessage string `column:"aiMessage" json:"aiMessage,omitempty"`

	// UserID 用户ID
	UserID string `column:"userID" json:"userID,omitempty"`

	// CreateTime 创建时间
	CreateTime string `column:"createTime" json:"createTime,omitempty"`
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *MessageLog) GetTableName() string {
	return tableMessageLogName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
// 如果没有主键,也需要实现这个方法, return "" 即可
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *MessageLog) GetPKColumnName() string {
	return "id"
}

// Site 站点信息
type Site struct {
	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// ID 主键
	Id string `column:"id" json:"id"`

	// Title 标题
	Title string `column:"title" json:"title,omitempty"`

	// Name 名称
	Name string `column:"name" json:"name,omitempty"`

	// Domain 域名
	Domain string `column:"domain" json:"domain,omitempty"`

	// Keyword 关键字
	Keyword string `column:"keyword" json:"keyword,omitempty"`

	// Description 描述
	Description string `column:"description" json:"description,omitempty"`

	// Theme 主题
	Theme string `column:"theme" json:"theme,omitempty"`

	// ThemePC PC主题
	ThemePC string `column:"themePC" json:"themePC,omitempty"`

	// ThemeWAP WAP主题WAP
	ThemeWAP string `column:"themeWAP" json:"themeWAP,omitempty"`

	// ThemeWX 微信主题
	ThemeWX string `column:"themeWX" json:"themeWX,omitempty"`

	// Logo 站点logo
	Logo string `column:"logo" json:"logo,omitempty"`

	// Favicon 站点favicon
	Favicon string `column:"favicon" json:"favicon,omitempty"`

	// Footer 页脚
	Footer string `column:"footer" json:"footer,omitempty"`

	// CreateTime 创建时间
	CreateTime string `column:"createTime" json:"createTime,omitempty"`

	// UpdateTime 更新时间
	UpdateTime string `column:"updateTime" json:"updateTime,omitempty"`

	// CreateUser 创建人,初始化 system
	CreateUser string `column:"createUser" json:"createUser,omitempty"`

	// SortNo 排序
	SortNo int `column:"sortNo" json:"sortNo,omitempty"`

	// Status 状态 禁用(0),可用(1)
	Status int `column:"status" json:"status,omitempty"`
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *Site) GetTableName() string {
	return tableSiteName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
// 如果没有主键,也需要实现这个方法, return "" 即可
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *Site) GetPKColumnName() string {
	return "id"
}

// User 用户信息
type User struct {
	// 引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

	// Id 主键
	Id string `column:"id" json:"id,omitempty"`

	// Account 账号
	Account string `column:"account" json:"account,omitempty"`

	// Password 密码
	Password string `column:"password" json:"password,omitempty"`

	// UserName 用户名
	UserName string `column:"userName" json:"userName,omitempty"`

	// CreateTime 创建时间
	CreateTime string `column:"createTime" json:"createTime,omitempty"`

	// UpdateTime 更新时间
	UpdateTime string `column:"updateTime" json:"updateTime,omitempty"`

	// CreateUser 创建人,初始化 system
	CreateUser string `column:"createUser" json:"createUser,omitempty"`

	// SortNo 排序
	SortNo int `column:"sortNo" json:"sortNo,omitempty"`

	// Status 状态 禁用(0),可用(1)
	Status int `column:"status" json:"status,omitempty"`
}

// GetTableName 获取表名称
// IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *User) GetTableName() string {
	return tableUserName
}

func (entity *User) GetPKColumnName() string {
	return "id"
}

// ThemeTemplate 主题模板
type ThemeTemplate struct {
	// Id 主键,完整路径
	Id string `json:"id,omitempty"`

	// Pid 上级目录
	Pid string `json:"pid,omitempty"`

	// Name 名称,
	Name string `json:"name,omitempty"`

	// FileType 文件类型:dir(目录),file(文件)
	FileType string `json:"fileType,omitempty"`

	// FileSuffix 文件后缀
	FileSuffix string `json:"fileSuffix,omitempty"`

	// FilePath 模板路径
	FilePath string `json:"filePath,omitempty"`

	// FileDocument 文件内容
	FileDocument string `json:"fileDocument,omitempty"`
}
