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
	"os"
	"time"

	"gitee.com/chunanyong/zorm"
)

// updateDocumentChunk 根据Document更新DocumentChunk
func updateDocumentChunk(ctx context.Context, document *Document) (bool, error) {
	zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		zorm.Update(ctx, document)
		finderDeleteChunk := zorm.NewDeleteFinder(tableDocumentChunkName).Append("WHERE knowledgeBaseID=?", document.KnowledgeBaseID)
		count, err := zorm.UpdateFinder(ctx, finderDeleteChunk)
		if err != nil {
			return count, err
		}
		documentChunks, err := splitDocument4Chunk(ctx, document)
		if err != nil {
			return documentChunks, err
		}

		dcs := make([]zorm.IEntityStruct, 0)
		for i := 0; i < len(documentChunks); i++ {
			dc := documentChunks[i]
			dcs = append(dcs, &dc)

		}
		count, err = zorm.InsertSlice(ctx, dcs)
		if err != nil {
			return count, err
		}
		finderUpdateDocument := zorm.NewUpdateFinder(tableDocumentName).Append("status=1 WHERE id=?", document.Id)
		return zorm.UpdateFinder(ctx, finderUpdateDocument)
	})

	return false, nil

}

// splitDocument4Chunk 分割文档为DocumentChunk
func splitDocument4Chunk(ctx context.Context, document *Document) ([]DocumentChunk, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	documentChunks := make([]DocumentChunk, 0)
	documentChunk := DocumentChunk{}
	documentChunk.Id = FuncGenerateStringID()
	documentChunk.DocumentID = document.Id
	documentChunk.KnowledgeBaseID = document.KnowledgeBaseID
	documentChunk.Markdown = document.Markdown
	documentChunk.CreateTime = now
	documentChunk.UpdateTime = now

	documentChunks = append(documentChunks, documentChunk)

	return documentChunks, nil
}

// readDocumentFile 读取文件内容
func readDocumentFile(ctx context.Context, document *Document) error {
	// TODO 先处理markdown文件,以后扩展获取
	markdownByte, err := os.ReadFile(datadir + document.FilePath)
	document.Markdown = string(markdownByte)
	return err
}
