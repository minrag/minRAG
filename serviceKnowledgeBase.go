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

	"gitee.com/chunanyong/zorm"
)

// validateIDExists 校验ID是否已经存在
func validateIDExists(ctx context.Context, id string) bool {
	id = funcTrimSuffixSlash(id)
	if id == "" {
		return true
	}

	f1 := zorm.NewSelectFinder(tableDocumentName, "id").Append("Where id=?", id)
	cid := ""
	zorm.QueryRow(ctx, f1, &cid)
	if cid != "" {
		return true
	}
	id = id + "/"
	f2 := zorm.NewSelectFinder(tableKnowledgeBaseName, "id").Append("Where id=?", id)
	zorm.QueryRow(ctx, f2, &cid)
	return cid != ""

}

// findKnowledgeBaseNameById 根据知识库ID查找知识库名称
func findKnowledgeBaseNameById(ctx context.Context, knowledgeBaseId string) (string, error) {
	if knowledgeBaseId == "" {
		return "", errors.New(funcT("Knowledge base cannot be empty"))
	}
	finder := zorm.NewSelectFinder(tableKnowledgeBaseName, "name").Append("WHERE id=?", knowledgeBaseId)
	knowledgeBaseName := ""
	_, err := zorm.QueryRow(ctx, finder, &knowledgeBaseName)
	return knowledgeBaseName, err
}
