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
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	//"github.com/bytedance/go-tagexpr/v2/binding"
	"gitee.com/chunanyong/zorm"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	"golang.org/x/crypto/sha3"
)

// alphaNumericReg 传入的列名只能是字母数字或下划线,长度不超过20
var alphaNumericReg = regexp.MustCompile("^[a-zA-Z0-9_]{1,20}$")

// init 初始化函数
func init() {
	// adminGroup 初始化管理员路由组
	var adminGroup = h.Group("/admin")
	//设置权限
	adminGroup.Use(permissionHandler())

	//设置json处理函数
	//binding.ResetJSONUnmarshaler(json.Unmarshal)
	/*
		binding.Default().ResetJSONUnmarshaler(func(data []byte, v interface{}) error {
			dec := json.NewDecoder(bytes.NewBuffer(data))
			dec.UseNumber()
			return dec.Decode(v)
		})
	*/

	// 异常页面
	h.GET("/admin/error", func(ctx context.Context, c *app.RequestContext) {
		cHtmlAdmin(c, http.StatusOK, "admin/error.html", nil)
	})

	// 安装
	h.GET("/admin/install", funcAdminInstallPre)
	h.POST("/admin/install", funcAdminInstall)

	// 后台管理员登录
	h.GET("/admin/login", funcAdminLoginPre)
	h.POST("/admin/login", funcAdminLogin)

	// 后台管理员首页
	adminGroup.GET("/index", func(ctx context.Context, c *app.RequestContext) {
		cHtmlAdmin(c, http.StatusOK, "admin/index.html", nil)
	})

	// 刷新站点,重新加载资源包含模板和对应的静态文件
	adminGroup.GET("/reload", funcAdminReload)

	//上传文件
	adminGroup.POST("/upload", funcUploadFile)
	//上传文档文件
	adminGroup.POST("/document/uploadDocument", funcUploadDocument)
	//上传主题文件
	adminGroup.POST("/themeTemplate/uploadTheme", funcUploadTheme)

	// 通用list列表
	adminGroup.GET("/:urlPathParam/list", funcList)
	// 查询主题模板
	adminGroup.GET("/themeTemplate/list", funcListThemeTemplate)
	// 查询Document列表,根据KnowledgeBaseId like
	adminGroup.GET("/document/list", funcDocumentList)
	// 查询Component列表
	adminGroup.GET("/component/list", funcComponentList)
	// 查询Agent列表
	adminGroup.GET("/agent/list", funcAgentList)

	// 通用查看
	adminGroup.GET("/:urlPathParam/look", funcLook)

	//跳转到修改页面
	adminGroup.GET("/:urlPathParam/update", funcUpdatePre)
	// 修改Config
	adminGroup.POST("/config/update", funcUpdateConfig)
	// 修改Site
	adminGroup.POST("/site/update", funcUpdateSite)
	// 修改User
	adminGroup.POST("/user/update", funcUpdateUser)
	// 修改KnowledgeBase
	adminGroup.POST("/knowledgeBase/update", funcUpdateKnowledgeBase)
	// 修改Document
	adminGroup.POST("/document/update", funcUpdateDocument)
	// 修改Component
	adminGroup.POST("/component/update", funcUpdateComponent)
	// 修改Agent
	adminGroup.POST("/agent/update", funcUpdateAgent)
	// 修改ThemeTemplate
	adminGroup.POST("/themeTemplate/update", funcUpdateThemeTemplate)

	//跳转到保存页面
	adminGroup.GET("/:urlPathParam/save", funcSavePre)
	//保存KnowledgeBase
	adminGroup.POST("/knowledgeBase/save", funcSaveKnowledgeBase)
	//保存Document
	adminGroup.POST("/document/save", funcSaveDocument)
	//保存Component
	adminGroup.POST("/component/save", funcSaveComponent)
	//保存Agent
	adminGroup.POST("/agent/save", funcSaveAgent)

	//ajax POST删除数据
	adminGroup.POST("/:urlPathParam/delete", funcDelete)
	//ajax POST删除Document
	adminGroup.POST("/document/delete", funcDeleteDocument)

	//ajax POST执行更新语句
	adminGroup.POST("/updatesql", funcUpdateSQL)
}

// funcAdminInstallPre 跳转到安装界面
func funcAdminInstallPre(ctx context.Context, c *app.RequestContext) {
	if installed { // 如果已经安装过了,跳转到登录
		c.Redirect(http.StatusOK, cRedirecURI("admin/login"))
		c.Abort() // 终止后续调用
		return
	}
	cHtmlAdmin(c, http.StatusOK, "admin/install.html", nil)
}

// funcAdminInstall 后台安装
func funcAdminInstall(ctx context.Context, c *app.RequestContext) {
	if installed { // 如果已经安装过了,跳转到登录
		c.Redirect(http.StatusOK, cRedirecURI("admin/login"))
		c.Abort() // 终止后续调用
		return
	}
	// 使用后端管理界面配置,jwtSecret也有后端随机产生
	user := User{}
	user.Account = c.PostForm("account")
	user.UserName = c.PostForm("account")
	user.Password = c.PostForm("password")
	// 重新hash密码,避免拖库后撞库
	sha3Bytes := sha3.Sum512([]byte(user.Password))
	user.Password = hex.EncodeToString(sha3Bytes[:])

	loginHtml := "admin/login?message=" + funcT("Congratulations, you have successfully installed MINRAG. Please log in now")

	err := insertUser(ctx, user)
	if err != nil {
		c.Redirect(http.StatusOK, cRedirecURI("admin/error"))
		c.Abort() // 终止后续调用
		return
	}
	// 安装成功,更新安装状态
	updateInstall(ctx)
	c.Redirect(http.StatusOK, cRedirecURI(loginHtml))
}

// funcAdminLoginPre 跳转到登录界面
func funcAdminLoginPre(ctx context.Context, c *app.RequestContext) {
	if !installed { // 如果没有安装,跳转到安装
		c.Redirect(http.StatusOK, cRedirecURI("admin/install"))
		c.Abort() // 终止后续调用
		return
	}
	responseData := make(map[string]string, 0)
	message, ok := c.GetQuery("message")
	if ok {
		responseData["message"] = message
	}
	if errorLoginCount.Load() >= errCount { //连续错误3次显示验证码
		responseData["showCaptcha"] = "1"
		generateCaptcha()
		responseData["captchaBase64"] = captchaBase64
	}
	c.SetCookie(config.JwttokenKey, "", config.Timeout, "/", "", protocol.CookieSameSiteStrictMode, false, true)
	cHtmlAdmin(c, http.StatusOK, "admin/login.html", responseData)
}

// funcAdminLogin 后台登录
func funcAdminLogin(ctx context.Context, c *app.RequestContext) {
	if !installed { // 如果没有安装,跳转到安装
		c.Redirect(http.StatusOK, cRedirecURI("admin/install"))
		c.Abort() // 终止后续调用
		return
	}

	if errorLoginCount.Load() >= errCount { //连续错误3次显示验证码
		answer := c.PostForm("answer")
		if answer != captchaAnswer { //答案不对
			c.Redirect(http.StatusOK, cRedirecURI("admin/login?message="+funcT("Incorrect verification code")))
			c.Abort() // 终止后续调用
			return
		}
	}

	account := strings.TrimSpace(c.PostForm("account"))
	password := strings.TrimSpace(c.PostForm("password"))
	if account == "" || password == "" { // 用户不存在或者异常
		c.Redirect(http.StatusOK, cRedirecURI("admin/login?message="+funcT("Account or password cannot be empty")))
		c.Abort() // 终止后续调用
		return
	}
	// 重新hash密码,避免拖库后撞库
	sha3Bytes := sha3.Sum512([]byte(password))
	password = hex.EncodeToString(sha3Bytes[:])

	userId, err := findUserId(ctx, account, password)
	if userId == "" || err != nil { // 用户不存在或者异常
		errorLoginCount.Add(1)
		c.Redirect(http.StatusOK, cRedirecURI("admin/login?message="+funcT("Account or password is incorrect")))
		c.Abort() // 终止后续调用
		return
	}
	jwttoken, _ := newJWTToken(userId)
	// c.HTML(http.StatusOK, "admin/index.html", nil)
	c.SetCookie(config.JwttokenKey, jwttoken, config.Timeout, "/", "", protocol.CookieSameSiteStrictMode, false, true)
	errorLoginCount.Store(0)
	c.Redirect(http.StatusOK, cRedirecURI("admin/index"))
}

// funcAdminReload 刷新站点,会重新加载模板文件,生成静态文件
func funcAdminReload(ctx context.Context, c *app.RequestContext) {
	err := loadTemplate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, ERR: err})
		c.Abort() // 终止后续调用
		return
	}
	// 刷新组件Map
	initComponentMap()

	//重新生成静态文件
	go genStaticFile()
	c.JSON(http.StatusOK, ResponseData{StatusCode: 1})
}

// funcUploadFile 上传文件
func funcUploadFile(ctx context.Context, c *app.RequestContext) {
	filePath, _, err := funcUploadFilePath(c, "upload/")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, ERR: err})
		c.Abort() // 终止后续调用
		return
	}
	c.JSON(http.StatusOK, ResponseData{StatusCode: 1, Data: filePath})
}

// funcUploadDocument 上传文档
func funcUploadDocument(ctx context.Context, c *app.RequestContext) {
	filePath, knowledgeBaseId, err := funcUploadFilePath(c, "upload/")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, ERR: err})
		c.Abort() // 终止后续调用
		return
	}
	knowledgeBaseName, err := findKnowledgeBaseNameById(ctx, knowledgeBaseId)
	if err != nil || knowledgeBaseName == "" {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, ERR: err})
		c.Abort() // 终止后续调用
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	document := Document{}
	document.Status = 2
	document.FilePath = filePath
	document.KnowledgeBaseID = knowledgeBaseId
	document.KnowledgeBaseName = knowledgeBaseName
	document.SortNo = funcMaxSortNo(tableDocumentName)
	document.Name = funcLastURI(filePath)
	document.FileExt = filepath.Ext(document.Name)
	document.CreateTime = now
	document.UpdateTime = now

	documentID, _ := findDocumentIdByFilePath(ctx, filePath)

	zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		if documentID == "" {
			document.Id = FuncGenerateStringID()
			return zorm.Insert(ctx, &document)
		}
		// 有updateDocumentChunk更新
		document.Id = documentID
		return nil, nil
	})
	// 文档分块,分析处理
	go updateDocumentChunk(ctx, &document)

	c.JSON(http.StatusOK, ResponseData{StatusCode: 1, Data: filePath})
}

// funcUploadFilePath 上传文件,返回文件的path路径
func funcUploadFilePath(c *app.RequestContext, baseDir string) (string, string, error) {
	fileHeader, err := c.FormFile("file")
	// 相对于上传的目录路径,只能是目录路径
	dirPath := string(c.FormValue("dirPath"))
	if err != nil {
		return "", "", err
	}
	dirPath = filepath.ToSlash(dirPath)
	dirPath = funcTrimSlash(dirPath)
	fileName := FuncGenerateStringID() + filepath.Ext(fileHeader.Filename)
	if dirPath == "/" {
		dirPath = ""
	}
	if dirPath != "" {
		dirPath = dirPath + "/"
		fileName = fileHeader.Filename
	}
	//服务器的目录,并创建目录
	serverDirPath := datadir + baseDir + dirPath
	err = os.MkdirAll(serverDirPath, 0600)
	if err != nil && !os.IsExist(err) {
		return "", dirPath, err
	}
	path := baseDir + dirPath + fileName
	newFileName := datadir + path
	err = c.SaveUploadedFile(fileHeader, newFileName)

	return path, "/" + dirPath, err
}

// funcUploadTheme 上传主题
func funcUploadTheme(ctx context.Context, c *app.RequestContext) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, ERR: err})
		c.Abort() // 终止后续调用
		return
	}
	ext := filepath.Ext(fileHeader.Filename)
	if ext != ".zip" { //不是zip
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, ERR: err})
		c.Abort() // 终止后续调用
		return
	}
	path := themeDir + fileHeader.Filename
	err = c.SaveUploadedFile(fileHeader, path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, ERR: err})
		c.Abort() // 终止后续调用
		return
	}
	defer func() {
		_ = os.Remove(path)
	}()
	//解压压缩包
	err = unzip(path, themeDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, ERR: err})
		c.Abort() // 终止后续调用
		return
	}
	c.JSON(http.StatusOK, ResponseData{StatusCode: 1, Data: path})
}

// funcList 通用list列表
func funcList(ctx context.Context, c *app.RequestContext) {
	urlPathParam := c.Param("urlPathParam")
	//获取页码
	pageNoStr := c.DefaultQuery("pageNo", "1")
	pageNo, _ := strconv.Atoi(pageNoStr)
	q := strings.TrimSpace(c.Query("q"))
	mapParams := make(map[string]interface{}, 0)
	//获取所有的参数
	c.Bind(&mapParams)
	//删除掉固定的两个
	delete(mapParams, "pageNo")
	delete(mapParams, "q")
	where := " WHERE 1=1 "
	var values []interface{} = make([]interface{}, 0)
	for k := range mapParams {
		if !alphaNumericReg.MatchString(k) {
			continue
		}
		value := c.Query(k)
		if strings.TrimSpace(value) == "" {
			continue
		}
		where = where + " and " + k + "=? "
		values = append(values, value)
	}
	sql := "* from " + urlPathParam + where + " order by sortNo desc "
	var responseData ResponseData
	var err error
	if len(values) == 0 {
		responseData, err = funcSelectList(urlPathParam, q, pageNo, defaultPageSize, sql)
	} else {
		responseData, err = funcSelectList(urlPathParam, q, pageNo, defaultPageSize, sql, values)
	}
	responseData.UrlPathParam = urlPathParam
	if err != nil {
		c.Redirect(http.StatusOK, cRedirecURI("admin/error"))
		c.Abort() // 终止后续调用
		return
	}
	listFile := "admin/" + urlPathParam + "/list.html"
	cHtmlAdmin(c, http.StatusOK, listFile, responseData)
}

// funcLook 通用查看,根据id查看
func funcLook(ctx context.Context, c *app.RequestContext) {
	funcLookById(ctx, c, "look.html")
}

// funcDocumentList 查询Document列表,根据KnowledgeBaseId like
func funcDocumentList(ctx context.Context, c *app.RequestContext) {
	urlPathParam := "document"
	//获取页码
	pageNoStr := c.DefaultQuery("pageNo", "1")
	q := strings.TrimSpace(c.Query("q"))
	pageNo, _ := strconv.Atoi(pageNoStr)
	id := strings.TrimSpace(c.Query("id"))
	values := make([]interface{}, 0)
	sql := ""
	if id != "" {
		sql = " * from document where knowledgeBaseID like ?  order by sortNo desc "
		values = append(values, id+"%")
	} else {
		sql = " * from document order by sortNo desc "
	}
	var responseData ResponseData
	var err error
	if len(values) == 0 {
		responseData, err = funcSelectList(urlPathParam, q, pageNo, defaultPageSize, sql)
	} else {
		responseData, err = funcSelectList(urlPathParam, q, pageNo, defaultPageSize, sql, values)
	}
	responseData.UrlPathParam = urlPathParam
	if err != nil {
		c.Redirect(http.StatusOK, cRedirecURI("admin/error"))
		c.Abort() // 终止后续调用
		return
	}
	listFile := "admin/" + urlPathParam + "/list.html"
	cHtmlAdmin(c, http.StatusOK, listFile, responseData)
}

// funcListThemeTemplate 所有的主题文件列表
func funcListThemeTemplate(ctx context.Context, c *app.RequestContext) {
	urlPathParam := "themeTemplate"
	var responseData ResponseData
	extMap := make(map[string]interface{})
	extMap["file"] = ""
	responseData.ExtMap = extMap
	list := make([]ThemeTemplate, 0)

	//遍历当前使用的模板文件夹
	err := filepath.Walk(themeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 分隔符统一为 / 斜杠
		path = filepath.ToSlash(path)
		path = path[strings.Index(path, themeDir)+len(themeDir):]
		if path == "" {
			return err
		}
		//获取文件后缀
		ext := filepath.Ext(path)
		ext = strings.ToLower(ext)
		// 跳过压缩的 gz文件
		if ext == ".gz" {
			return nil
		}

		pid := filepath.ToSlash(filepath.Dir(path))
		if pid == "." {
			pid = ""
		}

		themeTemplate := ThemeTemplate{}
		themeTemplate.FilePath = path
		themeTemplate.Pid = pid
		themeTemplate.Id = path
		themeTemplate.FileSuffix = ext
		themeTemplate.Name = info.Name()
		if info.IsDir() {
			themeTemplate.FileType = "dir"
		} else {
			themeTemplate.FileType = "file"
		}
		list = append(list, themeTemplate)
		return nil
	})

	responseData.UrlPathParam = urlPathParam
	responseData.Data = list
	responseData.ERR = err
	listFile := "admin/" + urlPathParam + "/list.html"

	filePath := c.Query("file")
	if filePath == "" || strings.Contains(filePath, "..") {
		//c.HTML(http.StatusOK, listFile, responseData)
		cHtmlAdmin(c, http.StatusOK, listFile, responseData)
		return
	}
	filePath = filepath.ToSlash(filePath)
	fileDocument, err := os.ReadFile(themeDir + filePath)
	if err != nil {
		responseData.ERR = err
		cHtmlAdmin(c, http.StatusOK, listFile, responseData)
		return
	}
	responseData.ExtMap["file"] = string(fileDocument)
	cHtmlAdmin(c, http.StatusOK, listFile, responseData)
}

// funcComponentList 查询组件列表
func funcComponentList(ctx context.Context, c *app.RequestContext) {
	urlPathParam := "component"
	listFile := "admin/" + urlPathParam + "/list.html"
	list, err := findAllComponentList(ctx)
	if err != nil {
		c.Redirect(http.StatusOK, cRedirecURI("admin/error"))
		c.Abort() // 终止后续调用
		return
	}
	var responseData ResponseData
	responseData.UrlPathParam = urlPathParam
	responseData.Data = list
	responseData.ERR = err
	cHtmlAdmin(c, http.StatusOK, listFile, responseData)
}

// funcAgentList 查询智能体列表
func funcAgentList(ctx context.Context, c *app.RequestContext) {
	urlPathParam := "agent"
	listFile := "admin/" + urlPathParam + "/list.html"
	list, err := findAllAgentList(ctx)
	if err != nil {
		c.Redirect(http.StatusOK, cRedirecURI("admin/error"))
		c.Abort() // 终止后续调用
		return
	}
	var responseData ResponseData
	responseData.UrlPathParam = urlPathParam
	responseData.Data = list
	responseData.ERR = err
	cHtmlAdmin(c, http.StatusOK, listFile, responseData)
}

// funcUpdateThemeTemplate 更新主题模板
func funcUpdateThemeTemplate(ctx context.Context, c *app.RequestContext) {
	themeTemplate := ThemeTemplate{}
	c.Bind(&themeTemplate)
	filePath := filepath.ToSlash(themeTemplate.FilePath)
	if filePath == "" || strings.Contains(filePath, "..") {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0})
		c.Abort() // 终止后续调用
		return
	}
	if !pathExist(themeDir + filePath) {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0})
		c.Abort() // 终止后续调用
		return
	}

	//打开文件
	file, err := os.OpenFile(themeDir+filePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0})
		c.Abort() // 终止后续调用
		return
	}
	defer file.Close() // 确保在函数结束时关闭文件

	// 写入内容
	_, err = file.WriteString(themeTemplate.FileDocument)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0})
		c.Abort() // 终止后续调用
		return
	}
	c.JSON(http.StatusOK, ResponseData{StatusCode: 1})
}

// funcUpdatePre 跳转到修改页面
func funcUpdatePre(ctx context.Context, c *app.RequestContext) {
	funcLookById(ctx, c, "update.html")
}

// funcUpdateConfig 更新配置
func funcUpdateConfig(ctx context.Context, c *app.RequestContext) {
	now := time.Now().Format("2006-01-02 15:04:05")
	entity := &Config{}
	ok := funcUpdateInit(ctx, c, entity)
	if !ok {
		return
	}
	if !strings.HasPrefix(entity.BasePath, "/") {
		entity.BasePath = "/" + entity.BasePath
	}
	if !strings.HasSuffix(entity.BasePath, "/") {
		entity.BasePath = entity.BasePath + "/"
	}
	entity.UpdateTime = now
	funcUpdate(ctx, c, entity, entity.Id)
	// 刷新组件Map
	initComponentMap()
}

// funcUpdateSite 更新站点
func funcUpdateSite(ctx context.Context, c *app.RequestContext) {
	now := time.Now().Format("2006-01-02 15:04:05")
	entity := &Site{}
	ok := funcUpdateInit(ctx, c, entity)
	if !ok {
		return
	}
	entity.UpdateTime = now
	funcUpdate(ctx, c, entity, entity.Id)
}

// funcUpdateUser 更新用户信息
func funcUpdateUser(ctx context.Context, c *app.RequestContext) {
	now := time.Now().Format("2006-01-02 15:04:05")
	entity := &User{}
	ok := funcUpdateInit(ctx, c, entity)
	if !ok {
		return
	}
	if entity.Password != "" {
		// 重新hash密码,避免拖库后撞库
		sha3Bytes := sha3.Sum512([]byte(entity.Password))
		entity.Password = hex.EncodeToString(sha3Bytes[:])
	} else {
		f1 := zorm.NewSelectFinder(tableUserName, "password").Append("WHERE id=?", entity.Id)
		password := ""
		zorm.QueryRow(ctx, f1, &password)
		entity.Password = password
	}
	entity.UpdateTime = now
	funcUpdate(ctx, c, entity, entity.Id)
}

// funcUpdateKnowledgeBa知识库新知识库
func funcUpdateKnowledgeBase(ctx context.Context, c *app.RequestContext) {
	entity := &KnowledgeBase{}
	ok := funcUpdateInit(ctx, c, entity)
	if !ok {
		return
	}
	funcUpdate(ctx, c, entity, entity.Id)
}

// funcUpdateDocument 更新内容
func funcUpdateDocument(ctx context.Context, c *app.RequestContext) {
	entity := &Document{}
	ok := funcUpdateInit(ctx, c, entity)
	if !ok {
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	entity.UpdateTime = now
	go updateDocumentChunk(ctx, entity)

	c.JSON(http.StatusOK, ResponseData{StatusCode: 1})
}

// funcUpdateComponent 更新组件
func funcUpdateComponent(ctx context.Context, c *app.RequestContext) {
	entity := &Component{}
	ok := funcUpdateInit(ctx, c, entity)
	if !ok {
		return
	}
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		return zorm.Update(ctx, entity)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, UrlPathParam: "component", Message: funcT("Failed to update data")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	// 刷新组件Map
	initComponentMap()
	c.JSON(http.StatusOK, ResponseData{StatusCode: 1, UrlPathParam: "component"})
}

// funcUpdateAgent 更新智能体
func funcUpdateAgent(ctx context.Context, c *app.RequestContext) {
	entity := &Agent{}
	ok := funcUpdateInit(ctx, c, entity)
	if !ok {
		return
	}
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		return zorm.Update(ctx, entity)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, UrlPathParam: "component", Message: funcT("Failed to update data")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	c.JSON(http.StatusOK, ResponseData{StatusCode: 1, UrlPathParam: "agent"})
}

// funcUpdateInit 初始化更新的对象参数,先从数据库查询,再更新数据
func funcUpdateInit(ctx context.Context, c *app.RequestContext, entity zorm.IEntityStruct) bool {
	jsontmp := make(map[string]interface{}, 0)
	c.Bind(&jsontmp)
	id := jsontmp["id"]
	finder := zorm.NewSelectFinder(entity.GetTableName()).Append("WHERE id=?", id)
	has, err := zorm.QueryRow(ctx, finder, entity)
	if !has || err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("ID does not exist")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return false
	}
	err = c.Bind(entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("JSON data conversion error")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return false
	}
	return true
}

// funcUpdate 更新表数据
func funcUpdate(ctx context.Context, c *app.RequestContext, entity zorm.IEntityStruct, id string) {
	if id == "" { //没有id,终止调用
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("ID cannot be empty")})
		c.Abort() // 终止后续调用
		return
	}
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		_, err := zorm.Update(ctx, entity)
		return nil, err
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("Failed to update data")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	c.JSON(http.StatusOK, ResponseData{StatusCode: 1})
}

// funcSavePre 跳转到保存页面
func funcSavePre(ctx context.Context, c *app.RequestContext) {
	urlPathParam := c.Param("urlPathParam")
	templateFile := "admin/" + urlPathParam + "/save.html"
	responseData := ResponseData{UrlPathParam: urlPathParam}
	responseData.QueryStringMap = wrapQueryStringMap(c)
	cHtmlAdmin(c, http.StatusOK, templateFile, responseData)
}

// funcSaveKnowledgeBa知识库存知识库
func funcSaveKnowledgeBase(ctx context.Context, c *app.RequestContext) {
	entity := &KnowledgeBase{}
	err := c.Bind(entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("JSON data conversion error")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	if entity.CreateTime == "" {
		entity.CreateTime = now
	}
	if entity.UpdateTime == "" {
		entity.UpdateTime = now
	}
	if entity.Pid != "" {
		entity.Id = entity.Pid + entity.Id + "/"
	} else {
		entity.Id = "/" + entity.Id + "/"
	}
	has := validateIDExists(ctx, entity.Id)
	if has {
		c.JSON(http.StatusConflict, ResponseData{StatusCode: 0, Message: funcT("URL path is duplicated, please modify the path identifier")})
		c.Abort() // 终止后续调用
		return
	}
	count, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		return zorm.Insert(ctx, entity)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("Failed to save data")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}

	c.JSON(http.StatusOK, ResponseData{StatusCode: count.(int), Message: funcT("Saved successfully!")})
}

// funcSaveDocument 保存内容
func funcSaveDocument(ctx context.Context, c *app.RequestContext) {
	entity := &Document{}
	err := c.Bind(entity)
	if err != nil || entity.Id == "" || entity.KnowledgeBaseID == "" {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("JSON data conversion error")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	// 构建ID
	entity.Id = entity.KnowledgeBaseID + entity.Id
	has := validateIDExists(ctx, entity.Id)
	if has {
		c.JSON(http.StatusConflict, ResponseData{StatusCode: 0, Message: funcT("URL path is duplicated, please modify the path identifier")})
		c.Abort() // 终止后续调用
		return
	}
	if entity.CreateTime == "" {
		entity.CreateTime = now
	}
	if entity.UpdateTime == "" {
		entity.UpdateTime = now
	}

	f := zorm.NewSelectFinder(tableKnowledgeBaseName, "name as knowledgeBaseName").Append(" where id =?", entity.KnowledgeBaseID)
	zorm.QueryRow(ctx, f, entity)

	count, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		return zorm.Insert(ctx, entity)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("Failed to save data")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	c.JSON(http.StatusOK, ResponseData{StatusCode: count.(int), Message: funcT("Saved successfully!")})
}

func funcSaveComponent(ctx context.Context, c *app.RequestContext) {
	entity := &Component{}
	err := c.Bind(entity)
	if err != nil || entity.Id == "" {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("JSON data conversion error")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	entity.CreateTime = now
	entity.UpdateTime = now
	count, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		return zorm.Insert(ctx, entity)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("Failed to save data")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	// 刷新组件Map
	initComponentMap()
	c.JSON(http.StatusOK, ResponseData{StatusCode: count.(int), Message: funcT("Saved successfully!")})
}

func funcSaveAgent(ctx context.Context, c *app.RequestContext) {
	entity := &Agent{}
	err := c.Bind(entity)
	if err != nil || entity.Id == "" {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("JSON data conversion error")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	entity.CreateTime = now
	entity.UpdateTime = now
	count, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		return zorm.Insert(ctx, entity)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("Failed to save data")})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	c.JSON(http.StatusOK, ResponseData{StatusCode: count.(int), Message: funcT("Saved successfully!")})
}

// funcDelete 删除数据
func funcDelete(ctx context.Context, c *app.RequestContext) {
	id := c.PostForm("id")
	//id := c.Query("id")
	if id == "" { //没有id,终止调用
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("ID cannot be empty")})
		c.Abort() // 终止后续调用
		return
	}
	urlPathParam := c.Param("urlPathParam")
	if urlPathParam == "knowledgeBase" {
		finder := zorm.NewSelectFinder(tableKnowledgeBaseName, "*").Append(" where pid =?", id)
		page := zorm.NewPage()
		pageNo, _ := strconv.Atoi("1")
		page.PageNo = pageNo
		data := make([]KnowledgeBase, 0)
		zorm.Query(context.Background(), finder, &data, page)
		if len(data) != 0 {
			c.JSON(http.StatusOK, ResponseData{StatusCode: 0, Message: funcT("Cannot delete a knowledge item with child elements!")})
		} else {
			err := deleteById(ctx, urlPathParam, id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("Failed to delete data")})
				c.Abort() // 终止后续调用
			}
			c.JSON(http.StatusOK, ResponseData{StatusCode: 1, Message: funcT("Data deleted successfully")})
		}
	} else {
		err := deleteById(ctx, urlPathParam, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("Failed to delete data")})
			c.Abort() // 终止后续调用
		}
		c.JSON(http.StatusOK, ResponseData{StatusCode: 1, Message: funcT("Data deleted successfully")})
	}
}

// funcDeleteDocument 删除Document,DocumentChunk,VecDocumentChunk
func funcDeleteDocument(ctx context.Context, c *app.RequestContext) {
	id := c.PostForm("id")
	//id := c.Query("id")
	if id == "" { //没有id,终止调用
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("ID cannot be empty")})
		c.Abort() // 终止后续调用
		return
	}
	err := funcDeleteDocumentById(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: funcT("Failed to delete data")})
		c.Abort() // 终止后续调用
	}
	c.JSON(http.StatusOK, ResponseData{StatusCode: 1, Message: funcT("Data deleted successfully")})
}

func funcUpdateSQL(ctx context.Context, c *app.RequestContext) {
	ajaxMap := make(map[string]string, 0)
	c.Bind(&ajaxMap)
	updateSQL := ajaxMap["updateSQL"]
	finder := zorm.NewFinder().Append(updateSQL)
	finder.InjectionCheck = false
	count, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		return zorm.UpdateFinder(ctx, finder)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{StatusCode: 0, Message: err.Error()})
		c.Abort() // 终止后续调用
		FuncLogError(ctx, err)
		return
	}
	c.JSON(http.StatusOK, ResponseData{StatusCode: 1, Message: fmt.Sprintf(funcT("Updated %d records"), count)})
}

// permissionHandler 权限拦截器
func permissionHandler() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		jwttoken := string(c.Cookie(config.JwttokenKey))
		//fmt.Println(config.JwtSecret)
		userId, err := userIdByToken(jwttoken)
		if err != nil || userId == "" {
			c.Redirect(http.StatusOK, cRedirecURI("admin/login"))
			c.Abort() // 终止后续调用
			return
		}
		c.SetCookie(config.JwttokenKey, jwttoken, config.Timeout, "/", "", protocol.CookieSameSiteStrictMode, false, true)
		// 传递从jwttoken获取的userId
		c.Set(tokenUserId, userId)
		// 设置用户类型是 管理员
		c.Set(userTypeKey, 1)
	}
}

// funcLookById 根据Id,跳转到查看页面
func funcLookById(ctx context.Context, c *app.RequestContext, templateFile string) {
	id := c.Query("id")
	if id == "" {
		c.Redirect(http.StatusOK, cRedirecURI("admin/error"))
		c.Abort() // 终止后续调用
		return
	}
	urlPathParam := c.Param("urlPathParam")
	if !alphaNumericReg.MatchString(urlPathParam) {
		c.Redirect(http.StatusOK, cRedirecURI("admin/error"))
		c.Abort() // 终止后续调用
		return
	}
	responseData := ResponseData{StatusCode: 0}
	data, err := funcSelectOne(urlPathParam, "* FROM "+urlPathParam+" WHERE id=? ", id)
	responseData.Data = data
	responseData.UrlPathParam = urlPathParam

	if err != nil {
		c.Redirect(http.StatusOK, cRedirecURI("admin/error"))
		c.Abort() // 终止后续调用
		return
	}
	lookFile := "admin/" + urlPathParam + "/" + templateFile
	responseData.StatusCode = 1
	responseData.QueryStringMap = wrapQueryStringMap(c)
	cHtmlAdmin(c, http.StatusOK, lookFile, responseData)
}

// wrapQueryStringMap 包装查询参数Map
func wrapQueryStringMap(c *app.RequestContext) map[string]string {
	queryStringMap := make(map[string]string, 0)
	c.BindQuery(&queryStringMap)
	for k := range queryStringMap {
		queryStringMap[k] = c.Query(k)
	}
	return queryStringMap
}
