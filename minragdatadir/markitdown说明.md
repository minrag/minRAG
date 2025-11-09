使用 [https://gitee.com/minrag/markitdown](https://gitee.com/minrag/markitdown) 解析文档,使用```python build.py```编译的```dist/markitdown```放到 ```minragdatadir```目录下,```MarkdownConverter```组件配置示例:
```json
{
    "model":"Qwen3-VL-30B-A3B-Instruct", 
    "prompt":"提取图片内容,不要有引导语,介绍语,换行等", 
    "markitdown":"minragdatadir/markitdown/markitdown",
    "markdownDir":"minragdatadir/upload/markitdown/markdown",
    "imageFileDir":"minragdatadir/upload/markitdown/image",
    "imageURLPrefix":"/upload/markitdown/image"
}
```
字段说明:
- `model`:图片解析的模型
- `prompt`:理解文档中图片的提示词
- `markitdown`:markitdown的命令
- `markdownDir`:生成的markdown文件目录
- `imageFileDir`:图片存放的目录
- `imageURLPrefix`:图片的URL前缀