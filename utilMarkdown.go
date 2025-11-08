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
	"strings"

	// 用于生成唯一的 ID
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// parseMarkdownToTree 将 Markdown 文本解析为 markdownNode 结构
// 返回 treeNode (树形结构) 和 listNode (扁平列表结构)
func parseMarkdownToTree(source []byte) ([]*DocumentChunk, []*DocumentChunk, error) {

	// 1. 初始化 Goldmark 解析器
	parser := goldmark.New().Parser()

	// 2. 解析 Markdown 文本,获取 AST
	doc := parser.Parse(text.NewReader(source))

	// treeNode 树形节点结构
	var treeNode []*DocumentChunk

	// listNode 按照顺序记录节点 (用于 PreID/NextID 赋值)
	var listNode []*DocumentChunk

	// ancestors 作为一个栈,用于跟踪当前路径上的父节点
	var ancestors []*DocumentChunk

	// 3. 遍历 AST 的所有子节点
	for node := doc.FirstChild(); node != nil; node = node.NextSibling() {

		// 4. 检查节点是否为标题
		if heading, ok := node.(*ast.Heading); ok {
			level := heading.Level

			// 提取标题的纯文本 (已修复: 替换过时的 Text(source) )
			var titleBuilder strings.Builder
			for n := heading.FirstChild(); n != nil; n = n.NextSibling() {
				if textNode, ok := n.(*ast.Text); ok {
					titleBuilder.Write(textNode.Segment.Value(source))
				}
			}
			title := titleBuilder.String()

			// 生成新的 ID
			id := FuncGenerateStringID()

			// === 5. 调整 "ancestors" 栈 (处理层级关系) ===
			for len(ancestors) > 0 && ancestors[len(ancestors)-1].Level >= level {
				ancestors = ancestors[:len(ancestors)-1]
			}

			// === 6. 确定 ParentID 并创建 newNode (在栈调整后进行) ===
			var parentID string
			if len(ancestors) > 0 {
				parentID = ancestors[len(ancestors)-1].Id
			}

			// 创建我们的新 Node,并记录 Level, ID, ParentID
			newNode := &DocumentChunk{
				Id:       id,
				ParentID: parentID,
				Title:    title,
				Level:    level,
			}

			// 7. 将新节点附加到正确的父节点或根列表
			if len(ancestors) == 0 {
				treeNode = append(treeNode, newNode)
			} else {
				parent := ancestors[len(ancestors)-1]
				parent.Nodes = append(parent.Nodes, newNode)
			}

			// 8. 将当前标题推入栈中
			ancestors = append(ancestors, newNode)

			// 9. 处理 PreID 和 NextID (链表关系)
			if len(listNode) > 0 {
				lastNode := listNode[len(listNode)-1]

				lastNode.NextID = newNode.Id
				newNode.PreID = lastNode.Id
			}

			// 10. 平行记录节点
			listNode = append(listNode, newNode)

		} else {
			// 11. 如果不是标题,则为内容节点
			if len(ancestors) > 0 {
				currentNode := ancestors[len(ancestors)-1]

				// === 修正: 专门处理 Fenced Code Block 以保留 ``` 标签 ===
				if fcb, isFencedCodeBlock := node.(*ast.FencedCodeBlock); isFencedCodeBlock {
					var codeBlock strings.Builder

					// 1. 添加开始的代码围栏和语言标签 (例如: ```shell)
					codeBlock.WriteString("\n")
					codeBlock.WriteString("```")

					// 【已修复: 替换过时的 fcb.Info.Text(source) 为 fcb.Info.Value(source) 】
					if fcb != nil && fcb.Info != nil && source != nil {
						info := string(fcb.Info.Value(source))

						if info != "" {
							codeBlock.WriteString(info)
						}
					}

					codeBlock.WriteString("\n")

					// 2. 添加代码块内容
					segments := node.Lines()
					for i := 0; i < segments.Len(); i++ {
						segment := segments.At(i)
						codeBlock.WriteString(string(source[segment.Start:segment.Stop]))
					}

					// 3. 添加结束的代码围栏 (例如: ```)
					codeBlock.WriteString("```\n")

					currentNode.Markdown += codeBlock.String()

				} else {
					// 正常的内容块处理 (Paragraph, List, etc.)
					segments := node.Lines()
					for i := 0; i < segments.Len(); i++ {
						segment := segments.At(i)
						currentNode.Markdown += string(source[segment.Start:segment.Stop])
					}
				}
			}
		}
	}

	return treeNode, listNode, nil
}
