v0.1.1
- 配置增加默认大语言模型```llmModel```
- 增加```MarkdownTOCIndex``` 和 ```MarkdownTOCRetriever```组件,支持markdown目录索引方式 
- 增加Dockerfile
- 增加联网搜索组件
- 默认使用[markitdown](https://gitee.com/minrag/markitdown)处理文档
- json无法序列化error类型,使用Message返回错误信息
- 完善注释,文档

v0.1.0
- 增加MCP服务
- 升级依赖
- 完善注释,文档

v0.0.9
- 升级FTS5分词组件 
- 增加WebScraper组件,实现网络爬虫
- 增加HtmlCleaner组件,清理html标签
- 完善注释,文档

v0.0.8 
- 修复日志记录的bug,只记录文本内容
- OpenAIChatMemory默认上下文长度为3
- 完善注释,文档

v0.0.7 
- 目录需要是755权限,才能正常读取,上传的文件默认是644
- 完善注释,文档

v0.0.6
- 全平台兼容DeepSeek R1思维链输出方式
- 完善注释,文档

v0.0.5
- 增加TikaConverter组件,支持tika文档解析
- 增加文档说明
- 修复删除按钮功能
- 完善注释,文档

v0.0.4
- 支持DeepSeek R1思维链
- 优化聊天界面
- 项目Logo
- 完善注释,文档

v0.0.3
- 依赖Go 1.24
- 支持字节火山引擎
- 支持阿里云百炼平台,新增BaiLianDocumentChunkReranker组件
- 完善注释,文档

v0.0.2
- 增强windows系统的兼容性
- 组件默认初始化DefaultHeaders
- 支持百度千帆平台和腾讯云LKE知识引擎,新增LKEDocumentEmbedder,LKETextEmbedder,LKEDocumentChunkReranker和GiteeDocumentChunkReranker组件
- 完善注释,文档

v0.0.1
- 实现14个核心组件
- 支持function calling
- 支持完整的Pipeline功能
- 基于gpress代码初始化版本
- 完善注释文档