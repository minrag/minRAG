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
	"encoding/json"
	"fmt"
	"testing"
)

func TestMarkdown(t *testing.T) {
	markdownInput := `
## 错误二级节点1
## 错误二级节点2
### 错误三级节点2

`

	// 解析 Markdown
	tree, list, err := parseMarkdownToTree([]byte(markdownInput))
	if err != nil {
		fmt.Println("Error parsing markdown:", err)
		return
	}

	// 将结果格式化为带缩进的 JSON
	jsonData, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling to JSON:", err)
		return
	}

	// 打印 JSON
	fmt.Println(string(jsonData))

	for _, v := range list {
		fmt.Println(*v)
	}

}
