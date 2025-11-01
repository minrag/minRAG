使用 [https://gitee.com/minrag/markitdown](https://gitee.com/minrag/markitdown) 解析文档,使用```python build.py```编译的```dist/markitdown```放到 ```minragdatadir```目录下,```MarkdownConverter```组件配置示例:
```json
{
	//图片解析的模型
	"model":"Qwen3-VL-30B-A3B-Instruct", 
	//理解文档中图片的提示词
    "prompt":"准确提取图片内容,直接描述图片,不要有引导语之类的无关信息", 
	// markdown的命令路径
	"markitdown":"minragdatadir/markitdown/markitdown",
	// 生成的markdown文件目录
	"markdownDir":"minragdatadir/upload/markitdown/markdown",
	// 图片存放的目录
	"imageFileDir":"minragdatadir/upload/markitdown/image",
	// URL的前缀目录
	"imageURLDir":"/upload/markitdown/image"
}
```
