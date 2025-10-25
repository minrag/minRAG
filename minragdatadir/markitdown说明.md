使用 https://gitee.com/minrag/markitdown 解析文档,将编译的```markitdown```放到 ```minragdatadir```目录下,注意配置```markitdown```的```config.json```中的图片解析模型.```MarkitdownConverter```组件配置示例:
```json
{
	"markitdown":"minragdatadir/markitdown/markitdown",
	"markdownDir":"minragdatadir/upload/markitdown/markdown"
}
```
注意修改```indexPipeline```流水线的参数,把原来的```MarkdownConverter```替换为```MarkitdownConverter```:
```json
{
	"start": "MarkitdownConverter",
	"process": {
		"MarkitdownConverter": "DocumentSplitter",
		"DocumentSplitter": "OpenAIDocumentEmbedder",
		"OpenAIDocumentEmbedder": "SQLiteVecDocumentStore"
	}
}
```