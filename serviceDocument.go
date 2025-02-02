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
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		// 更新文档
		zorm.Update(ctx, document)

		// 删除关联的数据,重新插入
		finderDeleteChunk := zorm.NewDeleteFinder(tableDocumentChunkName).Append("WHERE documentID=?", document.Id)
		count, err := zorm.UpdateFinder(ctx, finderDeleteChunk)
		if err != nil {
			return count, err
		}
		finderDeleteVec := zorm.NewDeleteFinder(tableVecDocumentChunkName).Append("WHERE documentID=?", document.Id)
		count, err = zorm.UpdateFinder(ctx, finderDeleteVec)
		if err != nil {
			return count, err
		}
		documentChunks, err := splitDocument4Chunk(ctx, document)
		if err != nil {
			return documentChunks, err
		}

		dcs := make([]zorm.IEntityStruct, 0)
		vecdcs := make([]zorm.IEntityStruct, 0)
		for i := 0; i < len(documentChunks); i++ {
			dc := documentChunks[i]
			dcs = append(dcs, &dc)
			vecdc := &VecDocumentChunk{}
			vecdc.Id = dc.Id
			vecdc.DocumentID = dc.DocumentID
			vecdc.KnowledgeBaseID = dc.KnowledgeBaseID
			vecdc.SortNo = dc.SortNo
			vecdc.Status = dc.Status

			embedder := OpenAITextEmbedder{
				APIKey:         "A4FTACZVPGAIV8PZCKIBEUGV7ZBMXTIBEGUGNC11",
				Model:          "bge-m3",
				APIBaseURL:     "https://ai.gitee.com/v1",
				DefaultHeaders: map[string]string{"X-Failover-Enabled": "true", "X-Package": "1910"},
			}
			output, err := embedder.Run(ctx, map[string]interface{}{"query": dc.Markdown})

			if err != nil {
				return output, err
			}
			//需要使用bge-m3模型进行embedding
			embedding := output["embedding"].([]float64)
			vecdc.Embedding, _ = vecSerializeFloat64(embedding)
			vecdcs = append(vecdcs, vecdc)
		}
		count, err = zorm.InsertSlice(ctx, dcs)
		if err != nil {
			return count, err
		}
		count, err = zorm.InsertSlice(ctx, vecdcs)
		if err != nil {
			return count, err
		}
		finderUpdateDocument := zorm.NewUpdateFinder(tableDocumentName).Append("status=1 WHERE id=?", document.Id)
		return zorm.UpdateFinder(ctx, finderUpdateDocument)
	})

	return false, err

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

// findDocumentIdByFilePath 根据文档路径查询文档ID
func findDocumentIdByFilePath(ctx context.Context, filePath string) (string, error) {
	finder := zorm.NewSelectFinder(tableDocumentName, "id").Append("WHERE filePath=?", filePath)
	id := ""
	_, err := zorm.QueryRow(ctx, finder, &id)
	return id, err
}
