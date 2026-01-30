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
	input := make(map[string]any)
	input["document"] = document

	indexPipeline, err := findPipelineById(ctx, "indexPipeline", input)
	if err != nil {
		return false, err
	}
	if indexPipeline == nil {
		return false, errors.New("indexPipeline is empty")
	}
	err = indexPipeline.Run(ctx, input)
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

// runeLength 计算字符串的符文长度（一个中文汉字长度算1）
func runeLength(s string) int {
	return len([]rune(s))
}

// recursiveSplit 递归分割实现
func (component *DocumentSplitter) recursiveSplit(text string, depth int) []string {
	// 计算文本的符文长度
	textLen := runeLength(text)
	// 如果文本长度小于等于目标长度，直接返回
	if textLen <= component.SplitLength {
		return []string{text}
	}
	// 终止条件：处理完所有分隔符
	if depth >= len(component.SplitBy) {
		// 如果没有更多分隔符，但文本仍然很长，我们需要强制分割
		// 这里简单按字符分割，但为了保持语义，我们可以按标点或空格分割
		// 暂时直接按长度分割
		return component.splitByLength(text, component.SplitLength)
	}

	currentSep := component.SplitBy[depth]
	parts := strings.Split(text, currentSep)
	// 收集处理后的部分
	segments := make([]string, 0)
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		partContent := part
		if i < len(parts)-1 { //不是最后一个
			partContent = partContent + currentSep
		}
		// 计算部分的符文长度
		partLen := runeLength(part)
		// 如果部分长度超过目标长度，递归分割
		if partLen >= component.SplitLength {
			subChunks := component.recursiveSplit(partContent, depth+1)
			segments = append(segments, subChunks...)
		} else {
			segments = append(segments, partContent)
		}
	}
	// 将部分合并成接近目标长度的块
	return mergeSegments(segments, component.SplitLength)
}

// mergeChunks 合并短内容
func (component *DocumentSplitter) mergeChunks(chunks []string) []string {
	// 合并短内容
	for i := 0; i < len(chunks); i++ {
		chunk := chunks[i]
		chunkLen := runeLength(chunk)
		if chunkLen >= component.SplitLength || i+1 >= len(chunks) {
			continue
		}
		nextChunk := chunks[i+1]
		nextChunkLen := runeLength(nextChunk)

		// 使用符文长度计算合并后的长度
		if (chunkLen + nextChunkLen) > (component.SplitLength*18)/10 {
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

// splitByLength 按长度分割文本，尽量在自然边界处分割
func (component *DocumentSplitter) splitByLength(text string, maxLen int) []string {
	if runeLength(text) <= maxLen {
		return []string{text}
	}

	// 将文本转换为符文切片以便正确分割中文
	runes := []rune(text)
	result := make([]string, 0)

	// 在中文文本中寻找自然分割点：标点符号、空格等
	//naturalSplitters := []rune{'。', '！', '？', '.', '!', '?', ';', '；', ',', '，', ' ', '\n', '\t', '、'}

	start := 0
	for start < len(runes) {
		// 计算本次分割的结束位置
		end := start + maxLen
		if end > len(runes) {
			end = len(runes)
		}

		// 如果已经到了文本末尾
		if start == end {
			break
		}

		// 尝试在自然边界处分割
		if end < len(runes) {
			// 向前寻找最近的标点或空格
			searchEnd := end
			if searchEnd > len(runes) {
				searchEnd = len(runes)
			}
			found := false
			for j := searchEnd - 1; j > start; j-- {
				for _, splitter := range component.SplitBy {
					splitterRunes := []rune(splitter)
					if len(splitterRunes) == 1 && runes[j] == splitterRunes[0] {
						end = j + 1 // 包含分割符
						found = true
						break
					}
				}
				if found {
					break
				}
			}

			// 如果没找到自然分割点，尝试向后寻找
			if !found && end < len(runes) {
				for j := end; j < len(runes) && j < start+maxLen*2; j++ {
					for _, splitter := range component.SplitBy {
						splitterRunes := []rune(splitter)
						if len(splitterRunes) == 1 && runes[j] == splitterRunes[0] {
							end = j + 1 // 包含分割符
							found = true
							break
						}
					}
					if found {
						break
					}
				}
			}
		}

		// 提取块
		chunk := string(runes[start:end])
		result = append(result, chunk)

		// 移动到下一个位置
		start = end
	}

	return result
}

// mergeSegments 合并小片段成接近目标长度的块
func mergeSegments(segments []string, targetLen int) []string {
	if len(segments) == 0 {
		return segments
	}

	result := make([]string, 0)
	current := ""

	for _, segment := range segments {
		segmentLen := runeLength(segment)
		currentLen := runeLength(current)

		// 如果当前块为空，直接开始新块
		if current == "" {
			current = segment
			continue
		}

		// 如果将当前片段加入不会超过目标长度的120%，则合并
		if currentLen+segmentLen <= targetLen*120/100 {
			current += segment
		} else {
			// 否则，保存当前块并开始新块
			result = append(result, current)
			current = segment
		}
	}

	// 添加最后一个块
	if current != "" {
		result = append(result, current)
	}

	return result
}
