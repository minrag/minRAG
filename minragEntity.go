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

import "gitee.com/chunanyong/zorm"

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
