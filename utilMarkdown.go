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
	"fmt"
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

	var treeNode []*DocumentChunk
	var listNode []*DocumentChunk
	var ancestors []*DocumentChunk

	// 3. 遍历 AST 的所有子节点
	for node := doc.FirstChild(); node != nil; node = node.NextSibling() {

		// 4. 检查节点是否为标题
		if heading, ok := node.(*ast.Heading); ok {
			level := heading.Level

			// 提取标题的纯文本
			var titleBuilder strings.Builder
			for n := heading.FirstChild(); n != nil; n = n.NextSibling() {
				// 确保只提取 ast.Text 节点的内容
				if textNode, ok := n.(*ast.Text); ok {
					titleBuilder.Write(textNode.Segment.Value(source))
				}
				// 忽略其他类型的节点，如 Link, CodeSpan 等，只关注纯文本标题
			}
			title := titleBuilder.String()

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

				// === 特殊处理 A: Fenced Code Block (必须手动添加围栏) ===
				if fcb, isFencedCodeBlock := node.(*ast.FencedCodeBlock); isFencedCodeBlock {
					var codeBlock strings.Builder

					// 确保代码块前有换行
					if currentNode.Markdown != "" && !strings.HasSuffix(currentNode.Markdown, "\n") {
						codeBlock.WriteString("\n")
					}

					// 1. 添加开始的代码围栏和语言标签 (例如: ```shell)
					codeBlock.WriteString("```")
					if fcb.Info != nil {
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
					// === 通用处理 B: 所有其他块级元素 (列表、段落、引用、HTML块等) ===
					var minStart, maxStop = -1, -1
					foundContent := false

					// 1. 递归遍历该节点下的所有子节点，找到文本在源码中的【最小开始】和【最大结束】位置
					ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
						if !entering {
							return ast.WalkContinue, nil
						}
						// 只有拥有 lines() 的块级节点，才参与边界计算
						if n.Type() == ast.TypeBlock {
							lines := n.Lines()
							if lines.Len() > 0 {
								foundContent = true
								firstLine := lines.At(0)
								lastLine := lines.At(lines.Len() - 1)

								if minStart == -1 || firstLine.Start < minStart {
									minStart = firstLine.Start
								}
								if lastLine.Stop > maxStop {
									maxStop = lastLine.Stop
								}
							}

						}
						return ast.WalkContinue, nil
					})

					// 2. 如果找到了有效的文本范围
					if foundContent && minStart != -1 && maxStop != -1 {
						// 【核心逻辑】回溯：从文本开始处向前寻找行首
						// 这能自动包含列表符（- ）、引用符（> ）和缩进
						realStart := minStart
						for realStart > 0 && source[realStart-1] != '\n' {
							realStart--
						}

						// 提取原始内容
						chunkContent := string(source[realStart:maxStop])

						// 3. 拼接内容
						// 如果当前 Markdown 已经有内容且不以换行结尾，补一个换行
						if currentNode.Markdown != "" && !strings.HasSuffix(currentNode.Markdown, "\n") {
							currentNode.Markdown += "\n"
						}
						// 添加提取的内容，并确保末尾有一个换行符，以便与下一个块隔开
						currentNode.Markdown += chunkContent + "\n"
					} else {
						// 专门处理没有内容但占据行数的节点，例如 <hr> 或空 HTML 块
						if node.Lines().Len() > 0 {
							// 提取原始行
							segments := node.Lines()
							var content strings.Builder
							for i := 0; i < segments.Len(); i++ {
								segment := segments.At(i)
								content.Write(source[segment.Start:segment.Stop])
							}

							if content.Len() > 0 {
								if currentNode.Markdown != "" && !strings.HasSuffix(currentNode.Markdown, "\n") {
									currentNode.Markdown += "\n"
								}
								currentNode.Markdown += content.String() + "\n"
							}
						}
					}
				}
			}
		}
	}

	// 12. 最终处理: 统一添加标题头
	for i := 0; i < len(listNode); i++ {
		node := listNode[i]
		if node.Markdown != "" {
			// 移除内容末尾多余的空格和换行符，然后加上标题和单个换行符
			node.Markdown = strings.TrimSpace(node.Markdown)
			node.Markdown = fmt.Sprintf("%s %s\n%s\n", strings.Repeat("#", node.Level), node.Title, node.Markdown)
		}
	}

	return treeNode, listNode, nil
}
