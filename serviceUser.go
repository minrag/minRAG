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

// insertUser 插入用户
func insertUser(ctx context.Context, user Userinfo) error {
	// 清空用户,只能有一个管理员
	deleteAll(ctx, tableUserinfoName)
	// 初始化数据
	user.Id = "minrag_admin"
	user.SortNo = 1
	user.Status = 1
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		return zorm.Insert(ctx, &user)
	})
	return err
}

// findUserId 查询用户ID
func findUserId(ctx context.Context, account string, password string) (string, error) {
	finder := zorm.NewSelectFinder(tableUserinfoName, "id").Append(" WHERE account=? and password=?", account, password)
	userId := ""
	_, err := zorm.QueryRow(ctx, finder, &userId)
	return userId, err
}
