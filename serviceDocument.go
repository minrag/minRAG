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
	"errors"
	"strings"

	"gitee.com/chunanyong/zorm"
)

// updateDocumentChunk 运行indexPipeline流水线,更新Document,DocumentChunk,VecDocumentChunk
func updateDocumentChunk(ctx context.Context, document *Document) (bool, error) {
	input := make(map[string]interface{})
	input["document"] = document
	if componentMap["indexPipeline"] == nil {
		return false, errors.New("indexPipeline is empty")
	}
	indexPipeline := componentMap["indexPipeline"]
	err := indexPipeline.Run(ctx, input)
	if err != nil {
		return false, err
	}
	errObj, has := input[errorKey]
	if has || errObj != nil {
		return false, errObj.(error)
	}

	return true, nil

}

// findDocumentIdByFilePath 根据文档路径查询文档ID
func findDocumentIdByFilePath(ctx context.Context, filePath string) (string, error) {
	finder := zorm.NewSelectFinder(tableDocumentName, "id").Append("WHERE filePath=?", filePath)
	id := ""
	_, err := zorm.QueryRow(ctx, finder, &id)
	return id, err
}

// findDocumentChunkMarkDown 查询DocumentChunk的markdown
func findDocumentChunkMarkDown(ctx context.Context, documentChunks []DocumentChunk) ([]DocumentChunk, error) {
	for i := 0; i < len(documentChunks); i++ {
		documentChunk := documentChunks[i]
		finder := zorm.NewSelectFinder(tableDocumentChunkName, "markdown").Append("WHERE id=?", documentChunk.Id)
		markdown := ""
		_, err := zorm.QueryRow(ctx, finder, &markdown)
		if err != nil {
			return documentChunks, err
		}
		documentChunks[i].Markdown = markdown
	}

	return documentChunks, nil
}

// funcDeleteDocumentById 根据文档ID删除 Document,DocumentChunk,VecDocumentChunk
func funcDeleteDocumentById(ctx context.Context, id string) error {
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		f1 := zorm.NewDeleteFinder(tableDocumentName).Append("WHERE id=?", id)
		count, err := zorm.UpdateFinder(ctx, f1)
		if err != nil {
			return count, err
		}
		f2 := zorm.NewDeleteFinder(tableDocumentChunkName).Append("WHERE documentID=?", id)
		count, err = zorm.UpdateFinder(ctx, f2)
		if err != nil {
			return count, err
		}
		f3 := zorm.NewDeleteFinder(tableVecDocumentChunkName).Append("WHERE documentID=?", id)
		return zorm.UpdateFinder(ctx, f3)
	})
	return err
}

// recursiveSplit 递归分割实现
func (component *DocumentSplitter) recursiveSplit(text string, depth int) []string {
	chunks := make([]string, 0)
	// 终止条件：处理完所有分隔符
	if depth >= len(component.SplitBy) {
		if text != "" {
			return append(chunks, text)
		}
		return chunks
	}

	currentSep := component.SplitBy[depth]
	parts := strings.Split(text, currentSep)
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		partContent := part
		if i < len(parts)-1 { //不是最后一个
			partContent = partContent + currentSep
		}

		// 处理超长内容
		if len(part) >= component.SplitLength {
			partLeaf := component.recursiveSplit(partContent, depth+1)
			if len(partLeaf) > 0 {
				chunks = append(chunks, partLeaf...)
			}
			continue
		} else {
			chunks = append(chunks, partContent)
		}
	}
	return chunks
}

// mergeChunks 合并短内容
func (component *DocumentSplitter) mergeChunks(chunks []string) []string {
	// 合并短内容
	for i := 0; i < len(chunks); i++ {
		chunk := chunks[i]
		if len(chunk) >= component.SplitLength || i+1 >= len(chunks) {
			continue
		}
		nextChunk := chunks[i+1]

		// 汉字字符占位3个长度
		if (len(chunk) + len(nextChunk)) > (component.SplitLength*18)/10 {
			continue
		}
		chunks[i] = chunk + nextChunk
		if i+2 >= len(chunks) { //倒数第二个元素,去掉最后一个
			chunks = chunks[:len(chunks)-1]
		} else { // 去掉 i+1 索引元素,合并到了 i 索引
			chunks = append(chunks[:i+1], chunks[i+2:]...)
		}
	}
	return chunks
}
