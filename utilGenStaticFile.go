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
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gitee.com/chunanyong/zorm"
	"golang.org/x/crypto/sha3"
)

// onlyOnce控制并发
// var onlyOnce = make(chan struct{}, 1)
var genStaticHtmlLock = &sync.Mutex{}

// genStaticFile 生成全站静态文件和gzip文件,包括静态的html和search-data.json
func genStaticFile() error {
	genStaticHtmlLock.Lock()
	defer genStaticHtmlLock.Unlock()

	ctx := context.Background()
	documents := make([]Document, 0)

	f_post := zorm.NewSelectFinder(tableDocumentName, "id").Append(" WHERE status=1 order by status desc, sortNo desc")
	err := zorm.Query(ctx, f_post, &documents, nil)
	if err != nil {
		return err
	}
	//生成知识库的静态网页
	knowledgeBaseIDs := make([]string, 0)
	f_knowledgeBase := zorm.NewSelectFinder(tableKnowledgeBaseName, "id").Append(" WHERE status=1 order by status desc,sortNo desc")
	err = zorm.Query(ctx, f_knowledgeBase, &knowledgeBaseIDs, nil)
	if err != nil {
		return err
	}
	//删除整个目录
	os.RemoveAll(staticHtmlDir)

	// 生成 default,pc,wap,weixin 等平台的静态文件
	useThemes := map[string]bool{}
	useThemes[""] = true
	err = genStaticFileByTheme(documents, knowledgeBaseIDs, site.Theme, "")
	if err != nil {
		FuncLogError(ctx, err)
		//return err
	}
	useThemes[site.Theme] = true
	_, has := useThemes[site.ThemePC]
	//生成PC模板的静态网页
	if !has {
		err = genStaticFileByTheme(documents, knowledgeBaseIDs, site.ThemePC, "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		if err != nil {
			FuncLogError(ctx, err)
			//return err
		}
		useThemes[site.ThemePC] = true
	}

	// 生成手机WAP模板的静态网页
	_, has = useThemes[site.ThemeWAP]
	if !has {
		err = genStaticFileByTheme(documents, knowledgeBaseIDs, site.ThemeWAP, "Mozilla/5.0 (Linux; Android 13;) Mobile")
		if err != nil {
			FuncLogError(ctx, err)
			//return err
		}
		useThemes[site.ThemeWAP] = true
	}
	//生成微信WX模板的静态网页
	_, has = useThemes[site.ThemeWX]
	if !has {
		err = genStaticFileByTheme(documents, knowledgeBaseIDs, site.ThemeWX, "Mozilla/5.0 (Linux; Android 13;) Mobile MicroMessenger WeChat Weixin")
		if err != nil {
			FuncLogError(ctx, err)
			//return err
		}
		useThemes[site.ThemeWX] = true
	}

	return err
}

// genStaticFileByTheme 根据主题模板,生成静态文件
func genStaticFileByTheme(documents []Document, categories []string, theme string, userAgent string) error {
	//生成首页index网页
	fileHash, _, err := writeStaticHtml("", "", theme, userAgent)
	if fileHash == "" || err != nil {
		return err
	}

	//上一个分页
	prvePageFileHash := ""
	//生成文章的静态网页
	for i := 0; i < len(documents); i++ {
		//postURL := httpServerPath + "post/" + postId
		fileHash, _, err := writeStaticHtml(funcTrimPrefixSlash(documents[i].Id), "", theme, userAgent)
		if fileHash == "" || err != nil {
			continue
		}

		fileHash, _, err = writeStaticHtml("page/"+strconv.Itoa(i+1), prvePageFileHash, theme, userAgent)
		if fileHash == "" || err != nil {
			continue
		}

		//如果hash完全一致,认为是最后一页
		prvePageFileHash = fileHash
	}

	for i := 0; i < len(categories); i++ {
		//生成知识库首页index
		fileHash, _, err := writeStaticHtml(funcTrimSlash(categories[i]), "", theme, userAgent)
		if fileHash == "" || err != nil {
			return err
		}

		for j := 0; j < len(documents); j++ {
			fileHash, _, err := writeStaticHtml(funcTrimSlash(categories[i])+"/page/"+strconv.Itoa(j+1), prvePageFileHash, theme, userAgent)
			if fileHash == "" || err != nil {
				continue
			}

			//如果hash完全一致,认为是最后一页
			prvePageFileHash = fileHash
		}
	}
	//遍历当前使用的模板文件夹,压缩文本格式的文件
	err = filepath.Walk(templateDir+"theme/"+theme+"/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 分隔符统一为 / 斜杠
		path = filepath.ToSlash(path)

		// 只处理 js 和 css 文件夹
		if !(strings.Contains(path, "/js/") || strings.Contains(path, "/css/")) {
			return nil
		}

		//获取文件后缀
		suffix := filepath.Ext(path)

		// 压缩 js,mjs,json,css,html
		// 压缩字体文件 ttf,otf,svg  gzip_types font/ttf font/otf image/svg+xml
		if !(suffix == ".js" || suffix == ".mjs" || suffix == ".json" || suffix == ".css" || suffix == ".html" || suffix == ".ttf" || suffix == ".otf" || suffix == ".svg") {
			return nil
		}

		// 获取要打包的文件信息
		readFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer readFile.Close()
		reader := bufio.NewReader(readFile)
		//压缩文件
		err = doGzipFile(path+compressedFileSuffix, reader)

		return err
	})
	return err
}

// writeStaticHtml 写入静态html
func writeStaticHtml(urlFilePath string, fileHash string, theme string, userAgent string) (string, bool, error) {
	httpurl := httpServerPath + urlFilePath
	filePath := staticHtmlDir + theme + funcBasePath() + urlFilePath
	if urlFilePath != "" {
		filePath = filePath + "/"
	}
	client := &http.Client{}
	req, err := http.NewRequest("GET", httpurl, nil)
	if err != nil {

		return "", false, err
	}

	// 设置请求头
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	response, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer response.Body.Close()

	// 读取资源数据 body: []byte
	body, err := io.ReadAll(response.Body)
	// 关闭资源流
	response.Body.Close()
	if err != nil {
		return "", false, err
	}
	//计算hash
	bytehex := sha3.Sum256(body)
	bodyHash := hex.EncodeToString(bytehex[:])
	if bodyHash == fileHash { //如果hash一致,不再生成文件
		return bodyHash, false, nil
	}
	// 写入文件
	os.MkdirAll(filePath, os.ModePerm)
	err = os.WriteFile(filePath+"index.html", body, os.ModePerm)
	if err != nil {
		return bodyHash, false, err
	}
	// 压缩gzip文件
	err = doGzipFile(filePath+"index.html"+compressedFileSuffix, bytes.NewReader(body))
	if err != nil {
		return bodyHash, false, err
	}
	return bodyHash, true, nil
}

// doGzipFile 压缩gzip文件
func doGzipFile(gzipFilePath string, reader io.Reader) error {

	//如果文件存在就删除
	if pathExist(gzipFilePath) {
		os.Remove(gzipFilePath)
	}
	//创建文件
	gzipFile, err := os.Create(gzipFilePath)
	if err != nil {
		return err
	}
	defer gzipFile.Close()

	gzipWrite, err := gzip.NewWriterLevel(gzipFile, gzip.BestCompression)
	if err != nil {
		return err
	}
	defer gzipWrite.Close()
	_, err = io.Copy(gzipWrite, reader)
	return err
}
