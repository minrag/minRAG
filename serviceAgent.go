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

// AgentRequestBody 请求Agent的参数
type AgentRequestBody struct {
	Model    string        `json:"model,omitempty"`
	Messages []ChatMessage `json:"messages,omitempty"`
	Stream   bool          `json:"stream,omitempty"`
	User     string        `json:"user,omitempty"`
}

// findAllAgentList 查询所有的智能体
func findAllAgentList(ctx context.Context) ([]Agent, error) {
	finder := zorm.NewSelectFinder(tableAgentName).Append("order by sortno desc")
	list := make([]Agent, 0)
	err := zorm.Query(ctx, finder, &list, nil)
	return list, err
}

// findAgentByID 查询Agent
func findAgentByID(ctx context.Context, agentID string) (*Agent, error) {
	finder := zorm.NewSelectFinder(tableAgentName).Append("WHERE id=? and status=1", agentID)
	agent := &Agent{}
	_, err := zorm.QueryRow(ctx, finder, agent)
	return agent, err
}
