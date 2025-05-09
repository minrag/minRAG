<img src="minragdatadir/public/minrag-logo.png" height="150px" />  

<a href="./README.zh-CN.md">简体中文</a> | <a href="./README.md">English</a> | <a href="./minragdatadir/public/doc/index.md">帮助文档</a>  

## RAG 从未如此简单 
minRAG是从零开始的RAG系统,追求极致的简单和强大,不超过1万行代码,无需安装,双击启动.支持OpenAI、Gitee AI、百度千帆、腾讯云LKE、阿里云百炼、字节火山引擎等AI平台.          
  
RAG入门教程: [<<十天手搓 minRAG, 操纵 DeepSeek 的幕后黑手>>](https://my.oschina.net/baobao/blog/17679781)  

## 支持的AI平台
因为 reranker 没有统一标准,组件参数中base_url要填写完整的路径   

### OpenAI
**minRAG实现了OpenAI的标准规范,所有兼容OpenAI的平台都可以使用.**     

### Gitee AI(默认)
AI平台默认是 [Gitee AI](https://ai.gitee.com),Gitee AI每天100次免费调用
- 注册或设置页面的AI平台```base_url``` 填写 https://ai.gitee.com/v1
- 注册或设置页面的AI平台```api_key```  填写 免费或者付费的token
- ```OpenAITextEmbedder``` 默认使用 ```bge-m3``` 模型  
- ```GiteeDocumentChunkReranker``` 组件参数 ```{"base_url":"https://ai.gitee.com/api/serverless/bge-reranker-v2-m3/rerank","model":"bge-reranker-v2-m3"}```  
- ```OpenAIChatGenerator``` 建议使用 ```DeepSeek-V3``` 模型  

### 腾讯云LKE知识引擎
- 注册或设置页面的AI平台```base_url``` 填写 ```SecretId```  ,或在组件参数配置```{"SecretId":"xxx"}```
- 注册或设置页面的AI平台```api_key```  填写 ```SecretKey``` ,或在组件参数配置```{"SecretKey":"xxx"}```
- ```LKETextEmbedder和LKEDocumentEmbedder``` 默认使用 ```lke-text-embedding-v1``` 模型  
- ```LKEDocumentChunkReranker``` 默认使用 ```lke-reranker-base``` 模型
- ```OpenAIChatGenerator``` [使用OpenAI SDK方式接入](https://console.cloud.tencent.com/lkeap),组件参数配置 ```{"base_url":"https://api.lkeap.cloud.tencent.com/v1","api_key":"xxx","model":"deepseek-v3"}```  
- 记得修改流水线中的组件

### 百度千帆
- 注册或设置页面的AI平台```base_url``` 填写 https://qianfan.baidubce.com/v2
- 注册或设置页面的AI平台```api_key```  填写 永久有效API Key
- ```OpenAITextEmbedder```和```OpenAIDocumentEmbedder``` 默认使用 ```bge-large-zh``` 模型,1024维度  
- ```DocumentChunkReranker``` 组件参数配置 ```{"base_url":"https://qianfan.baidubce.com/v2/rerankers","model":"bce-reranker-base","top_n":5,"score":0.1}```  
- ```OpenAIChatGenerator``` 建议使用 ```deepseek-v3``` 模型 
- 记得修改流水线中的组件

### 阿里云百炼  
- 注册或设置页面的AI平台```base_url``` 填写 https://dashscope.aliyuncs.com/compatible-mode/v1
- 注册或设置页面的AI平台```api_key```  填写 申请的API KEY
- ```OpenAITextEmbedder```和```OpenAIDocumentEmbedder``` 默认使用 ```text-embedding-v3``` 模型,1024维度 
- ```BaiLianDocumentChunkReranker``` 组件参数配置 ```{"base_url":"https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank","model":"gte-rerank","top_n":5,"score":0.1}```  
- ```OpenAIChatGenerator``` 建议使用 ```deepseek-v3``` 模型 
- 记得修改流水线中的组件

### 字节火山引擎
- 注册或设置页面的AI平台```base_url``` 填写 https://ark.cn-beijing.volces.com/api/v3
- 注册或设置页面的AI平台```api_key```  填写 申请的API KEY
- ```OpenAITextEmbedder```和```OpenAIDocumentEmbedder``` 建议使用```doubao-embedding```模型,兼容1024维度 
- ```DocumentChunkReranker``` 火山引擎暂时没有Reranker模型,建议使用其他平台的Reranker模型或者去掉  
- ```OpenAIChatGenerator``` 建议使用 ```deepseek-v3```模型  
- 记得修改流水线中的组件

## tika集成
默认minRAG只支持markdown和text等文本格式,可以使用```TikaConverter```组件调用```tika```服务解析文档内容,```TikaConverter```组件配置示例:
```json
{
	"tikaURL": "http://localhost:9998/tika",
	"defaultHeaders": {
		"Content-Type": "application/octet-stream"
	}
}
```
启动 ```tika``` 的命令如下:
```shell
## tika 3.x 依赖 jdk11+
java -jar tika-server-standard-3.1.0.jar --host=0.0.0.0 --port=9998

## 不输出日志
#nohup java -jar tika-server-standard-3.1.0.jar --host=0.0.0.0 --port=9998 >/dev/null 2>&1 &
```

或者下载[tika-windows](https://pan.baidu.com/s/1OR0DaAroxf8dBTwz36Ceww?pwd=1234)   ```start.bat```启动tika.  
注意修改```indexPipeline```流水线的参数,把原来的```MarkdownConverter```替换为```TikaConverter```:
```json
{
	"start": "TikaConverter",
	"process": {
		"TikaConverter": "DocumentSplitter",
		"DocumentSplitter": "OpenAIDocumentEmbedder",
		"OpenAIDocumentEmbedder": "SQLiteVecDocumentStore"
	}
}
```

## 界面预览
<img src="minragdatadir/public/demo.png" width="600px" />    

<img src="minragdatadir/public/index.png" width="600px" />    

## 开发环境  
### fts5
minRAG使用了 ```https://github.com/wangfenjin/simple``` 作为FTS5的全文检索扩展,编译好的libsimple文件放到 ```minragdatadir/extensions``` 目录下,如果minRAG启动报错连不上数据库,请检查libsimple文件是否正确,如果需要重新编译libsimple,请参考 https://github.com/wangfenjin/simple.  

默认端口738,后台管理地址 http://127.0.0.1:738/admin/login    
需要先解压```minragdatadir/dict.zip```      
运行 ```go run --tags "fts5" .```     
打包: ```go build --tags "fts5" -ldflags "-w -s"```  

开发环境需要配置CGO编译,设置```set CGO_ENABLED=1```,下载[mingw64](https://github.com/niXman/mingw-builds-binaries/releases)和[cmake](https://cmake.org/download/),并把bin配置到环境变量,注意把```mingw64/bin/mingw32-make.exe``` 改名为 ```make.exe```  
注意修改vscode的launch.json,增加 ``` ,"buildFlags": "--tags=fts5" ``` 用于调试fts5    
test需要手动测试:``` go test -v -count=1 -timeout 30s --tags "fts5"  -run ^TestVecQuery$ gitee.com/minrag/minrag ```  
打包: ``` go build --tags "fts5" -ldflags "-w -s" ```   
重新编译simple时,建议使用```https://github.com/wangfenjin/simple```编译好的.  
注意修改widnows编译脚本,去掉 mingw64 编译依赖的```libgcc_s_seh-1.dll```和```libstdc++-6.dll```,同时关闭```BUILD_TEST_EXAMPLE```,有冲突.  
注意: windows 打包之后,需要把 ```minragdatadir/libgcc_s_seh-1.dll``` 复制到minrag.exe同一个目录,兼容windows的gcc库  
```bat
rmdir /q /s build
mkdir build && cd build
cmake .. -G "Unix Makefiles" -DBUILD_TEST_EXAMPLE=OFF -DCMAKE_INSTALL_PREFIX=release -DCMAKE_CXX_FLAGS="-static-libgcc -static-libstdc++" -DCMAKE_EXE_LINKER_FLAGS="-Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic"
make && make install
```

### sqlite_vec
参考:https://alexgarcia.xyz/sqlite-vec/compiling.html  
把```mingw64/bin/gcc.exe``` 复制为 ```cc.exe```  
```shell
#git clone https://github.com/asg017/sqlite-vec.git
git clone https://gitee.com/minrag/sqlite-vec.git
###windows上用git bash 打开目录并执行命令,cmd不支持
cd sqlite-vec
./scripts/vendor.sh
make loadable
## dist/vec0.dll
```

## 后台管理支持英文
minRAG后台管理目前支持中英双语,支持扩展其他语言,语言文件在 ```minragdatadir/locales```,初始化安装默认使用的中文(```zh-CN```),如果需要英文,可以在安装前把```minragdatadir/install_config.json```中的```"locale":"zh-CN"```修改为```"locale":"en-US"```.也可以在安装成功之后,在```设置```中修改```语言```为```English```,并重启生效.  

## 表结构  
ID默认使用时间戳(23位)+随机数(9位),全局唯一.  
建表语句```minragdatadir/minrag.sql```          

### 配置(表名:config)
安装时会读取```minragdatadir/install_config.json```

| columnName  | 类型        | 说明         |  备注       | 
| ----------- | ----------- | ----------- | ----------- |
| id          | string      | 主键        |minrag_config |
| basePath    | string      | 基础路径    |  默认 /      |
| jwtSecret   | string      | jwt密钥     | 随机生成     |
| jwttokenKey | string      | jwt的key    |  默认 jwttoken  |
| serverPort  | string      | IP:端口     |  默认 :738  |
| timeout     | int         | jwt超时时间秒|  默认 7200  |
| maxRequestBodySize | int  | 最大请求     |  默认 20M  |
| locale      | string      | 语言包       |  默认 zh-CN,en-US |
| proxy       | string      | http代理地址 |             |
| createTime  | string      | 创建时间     |  2006-01-02 15:04:05  |
| updateTime  | string      | 更新时间     |  2006-01-02 15:04:05  |
| createUser  | string      | 创建人       |  初始化 system  |
| sortNo      | int         | 排序         |  倒序  |
| status      | int         | 状态     |  禁用(0),可用(1)  |

### 用户(表名:user)
后台只有一个用户.

| columnName  | 类型         | 说明        |  备注       | 
| ----------- | ----------- | ----------- | ----------- |
| id          | string      | 主键        | minrag_admin |
| account     | string      | 登录名称    |  默认admin  |
| passWord    | string      | 密码        |    -  |
| userName    | string      | 说明        |    -  |
| createTime  | string      | 创建时间     |  2006-01-02 15:04:05  |
| updateTime  | string      | 更新时间     |  2006-01-02 15:04:05  |
| createUser  | string      | 创建人       |  初始化 system  |
| sortNo      | int         | 排序         |  倒序  |
| status      | int         | 状态     |  禁用(0),可用(1)  |

### 站点信息(表名:site)
站点的信息,例如 title,logo,keywords,description等

| columnName    | 类型         | 说明    |  备注       | 
| ----------- | ----------- | ----------- | ----------- |
| id          | string      | 主键        |minrag_site  |
| title       | string      | 站点名称     |     -  |
| keyword     | string      | 关键字       |     -  |
| description | string      | 站点描述    |     -  |
| theme       | string      | 默认主题     | 默认使用default  |
| themePC     | string      | PC主题      | 先从cookie获取,如果没有从Header头取值,写入cookie,默认使用default  |
| themeWAP    | string      | 手机主题    | 先从cookie获取,如果没有从Header头取值,写入cookie,默认使用default  |
| themeWX     | string      | 微信主题    | 先从cookie获取,如果没有从Header头取值,写入cookie,默认使用default  |
| logo        | string      | logo       |     -  |
| favicon     | string      | Favicon    |     -  |
| createTime  | string      | 创建时间     |  2006-01-02 15:04:05  |
| updateTime  | string      | 更新时间     |  2006-01-02 15:04:05  |
| createUser  | string      | 创建人       |  初始化 system  |
| sortNo      | int         | 排序         |  倒序  |
| status      | int         | 状态     |  禁用(0),可用(1)  |

### 知识库(表名:knowledgeBase)
| columnName    | 类型         | 说明    |  备注       | 
| ----------- | ----------- | ----------- | ----------- |
| id          | string      | 主键         | URL路径,用/隔开,例如/web/ |
| name        | string      | 知识库名称     |    -  |
| pid         | string      | 父知识库ID     | 父知识库ID  |
| knowledgeBaseType  | int      | 知识库类型       | -  |
| createTime  | string      | 创建时间     |  2006-01-02 15:04:05  |
| updateTime  | string      | 更新时间     |  2006-01-02 15:04:05  |
| createUser  | string      | 创建人       |  初始化 system  |
| sortNo      | int         | 排序         |  倒序  |
| status      | int         | 状态     |  禁用(0),可用(1)  |

### 文档(表名:document)
| columnName  | 类型        | 说明        | 是否分词 |  备注                  | 
| ----------- | ----------- | ----------- | ------- | ---------------------- |
| id          | string      | 主键         |   否    | URL路径,用/隔开,例如/web/nginx-use-hsts |
| name        | string      | 文档名称     | 否      |    -    |
| knowledgeBaseID  | string | 知识库ID     | 否      |    -    |
| knowledgeBaseName | string | 知识库名称   | 否      |    -    |
| toc         | string      | 目录         | 否      |      -  |
| summary     | string      | 摘要         | 否      |      -  |
| markdown    | string      | Markdown内容 | 否      | - |
| filePath    | string      | 文件路径     | 否      |                         |
| fileSize    | int      | 文件大小 | 否   |                         |
| fileExt     | string      | 文件扩展名   | 否   |                         |
| createTime  | string      | 创建时间     | -       |  2006-01-02 15:04:05    |
| updateTime  | string      | 更新时间     | -       |  2006-01-02 15:04:05    |
| createUser  | string      | 创建人       | -       |  初始化 system          |
| sortNo      | int         | 排序         | -       |  倒序                   |
| status      | int         | 状态     | - | 禁用(0),可用(1),处理中(2),处理失败(3) |


### 文档拆分(表名:document_chunk)
| columnName  | 类型        | 说明        | 是否分词 |  备注                  | 
| ----------- | ----------- | ----------- | ------- | ---------------------- |
| id          | string      | 主键         |   否    | - |
| documentID  | string      | 文档ID     | 否      |    -    |
| knowledgeBaseID  | string | 知识库ID     | 否      |    -    |
| knowledgeBaseName | string | 知识库名称   | 否      |    -    |
| markdown    | string      | Markdown内容 | 是      | 使用 jieba 分词器 |
| createTime  | string      | 创建时间     | -       |  2006-01-02 15:04:05    |
| updateTime  | string      | 更新时间     | -       |  2006-01-02 15:04:05    |
| createUser  | string      | 创建人       | -       |  初始化 system          |
| sortNo      | int         | 排序         | -       |  倒序                   |
| status      | int         | 状态     | - | 禁用(0),可用(1),处理中(2),处理失败(3) |

### 组件(表名:component)
| columnName  | 类型        | 说明        | 是否分词 |  备注                  | 
| ----------- | ----------- | ----------- | ------- | ---------------------- |
| id          | string      | 主键         |   否    | - |
| componentType| string     | 组件类型     | 否      |    -    |
| parameter  | string       | 组件参数     | 否      |    -    |
| createTime  | string      | 创建时间     | -       |  2006-01-02 15:04:05    |
| updateTime  | string      | 更新时间     | -       |  2006-01-02 15:04:05    |
| createUser  | string      | 创建人       | -       |  初始化 system          |
| sortNo      | int         | 排序         | -       |  倒序                   |
| status      | int         | 状态         | -       | 禁用(0),可用(1) |

### 智能体(表名:agent)
| columnName  | 类型        | 说明        | 是否分词 |  备注                  | 
| ----------- | ----------- | ----------- | ------- | ---------------------- |
| id          | string      | 主键         |   否    | - |
| name        | string      | 智能体名称    | 否      |    -    |
| knowledgeBaseID  | string | 知识库ID     | 否      |    -    |
| pipelineID  | string      | 流水线ID     | 否      |    -    |
| defaultReply  | string    | 默认回复     | 否      |    -    |
| agentType   | int         | 智能体类型     | 否      |    -    |
| agentPrompt | string      | 智能体提示词 | 否      |    -    |
| avatar      | string      | 智能体头像   | 否      |    -    |
| welcome     | string      | 欢迎语       | 否      |    -    |
| tools       | string      | 调用的函数   | 否      |    -    |
| memoryLength| int         | 上下文记忆长度| 否      |    -    |
| createTime  | string      | 创建时间     | -       |  2006-01-02 15:04:05    |
| updateTime  | string      | 更新时间     | -       |  2006-01-02 15:04:05    |
| createUser  | string      | 创建人       | -       |  初始化 system          |
| sortNo      | int         | 排序         | -       |  倒序                   |
| status      | int         | 状态         | -       | 禁用(0),可用(1) |

### 聊天室(表名:chat_room)
| columnName  | 类型        | 说明        | 是否分词 |  备注                  | 
| ----------- | ----------- | ----------- | ------- | ---------------------- |
| id          | string      | 主键         |   否    | - |
| name        | string      | 名称         | 否      |    -    |
| agentID     | string      | 智能体ID     | 否      |    -    |
| pipelineID  | string      | 流水线ID     | 否      |    -    |
| knowledgeBaseID  | string | 知识库ID     | 否      |    -    |
| userID      | string      | 用户ID       | 否      |    -    |
| createTime  | string      | 创建时间     | -       |  2006-01-02 15:04:05 |

### 消息日志(表名:message_log)
| columnName  | 类型        | 说明        | 是否分词 |  备注                  | 
| ----------- | ----------- | ----------- | ------- | ---------------------- |
| id          | string      | 主键         |   否    | - |
| agentID     | string      | 智能体ID     | 否      |    -    |
| roomID      | string      | 聊天室ID     | 否      |    -    |
| pipelineID  | string      | 流水线ID     | 否      |    -    |
| knowledgeBaseID  | string | 知识库ID     | 否      |    -    |
| userMessage | string      | 用户的消息    | 否      |    -    |
| aiMessage   | string      | AI回复的消息 | 否      |    -    |
| userID      | string      | 用户ID       | 否      |    -    |
| createTime  | string      | 创建时间     | -       | 2006-01-02 15:04:05 |


## 版权软著说明
* 本minRAG软件著作权登记号2025SR0616004
* 本minRAG软件著作权归我们所有,禁止进行二次的软著申请,侵权必究
* 开发者使用minRAG开发的程序版权归开发者所有
* 请保留版权,而无任何其他的限制.也就是说,您必须在您的发行版里包含原许可协议的声明,无论您是以二进制发布的还是以源代码发布
* 开源版遵循AGPL-3.0开源协议发布,并提供免费使用,但不允许修改后和衍生的代码做为闭源的商业软件发布和销售!
