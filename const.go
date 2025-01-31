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

const (
	// 默认名称
	appName = "minrag"

	// 基本目录
	datadir = "minragdatadir/"
	// 数据目录,如果不存在认为是第一次安装启动,会创建默认的数据
	sqliteDBfile = datadir + "minrag.db"

	// config 配置的表名称
	tableConfigName = "config"

	// user 用户的表名称
	tableUserName = "user"
	// site  站点信息
	tableSiteName = "site"

	// 知识库 knowledgeBase
	tableKnowledgeBaseName = "knowledgeBase"

	// 文档
	tableDocumentName = "document"

	// 文档分块
	tableDocumentChunkName = "document_chunk"

	// 向量化的数据表
	tableVecDocumentChunkName = "vec_document_chunk"
	//---------------------------//

	// 模板的路径
	templateDir = datadir + "template/"

	// 主题的路径
	themeDir = templateDir + "theme/"

	// 静态化文件目录,网站生成的静态html
	staticHtmlDir = datadir + "statichtml/"

	// 数据默认的创建用户
	createUser = "system"

	tokenUserId = "userId"

	letters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	userTypeKey = "userType" //0访客,1管理员

	defaultPageSize = 10

	// 静态文件压缩后缀,兼容Nginx gzip_static
	compressedFileSuffix = ".gz"

	//版本号
	version = "v0.0.1"
)
