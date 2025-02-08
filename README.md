<a href="./README.md">English</a> | <a href="./README.zh-CN.md">简体中文</a> 
## Introduction  
Minrag is a RAG system that starts from scratch, aiming for ultimate simplicity and power, with no more than 10,000 lines of code, no installation required, and double-click to launch.  
    
It uses FTS5 to implement BM25 full-text search and Vec for vector search. It has implemented components such as MarkdownConverter,DocumentSplitter,OpenAIDocumentEmbedder,SQLiteVecDocumentStore,OpenAITextEmbedder,VecEmbeddingRetriever,FtsKeywordRetriever,DocumentChunksReranker,PromptBuilder,OpenAIChatMessageMemory,OpenAIChatCompletion and Pipeline, supporting pipeline settings and extensions.  

The default AI platform is [Gitee AI](https://ai.gitee.com), with 100 free daily call credits.
- OpenAITextEmbedder uses the bge-m3 model by default.
- DocumentChunksReranker uses the bge-reranker-v2-m3 model by default.
- OpenAIChatCompletion uses the Qwen2.5-72B-Instruct model by default.
All of the above models can be modified within the components.


## Development Environment  
minrag uses ```https://github.com/wangfenjin/simple``` as the FTS5 full-text search extension. The compiled libsimple file is placed in the ```minragdatadir/extensions``` directory. If minrag fails to start and reports an error connecting to the database, please check if the libsimple file is correct. If you need to recompile libsimple, please refer to https://github.com/wangfenjin/simple.  

The default port is 738, and the backend management address is http://127.0.0.1:738/admin/login.  
First, unzip ```minragdatadir/dict.zip```.  
Run ```go run --tags "fts5" .```.  
Package: ```go build --tags "fts5" -ldflags "-w -s"```.  

The development environment requires CGO compilation configuration. Set ```set CGO_ENABLED=1```, download [mingw64](https://github.com/niXman/mingw-builds-binaries/releases) and [cmake](https://cmake.org/download/), and configure the bin to the environment variables. Note to rename ```mingw64/bin/mingw32-make.exe``` to ```make.exe```.  
Modify vscode's launch.json to add ``` ,"buildFlags": "--tags=fts5" ``` for debugging fts5.  
Test needs to be done manually: ```go test -v -count=1 -timeout 30s --tags "fts5" -run ^TestVecQuery$ gitee.com/minrag/minrag```.  
Package: ```go build --tags "fts5" -ldflags "-w -s"```.  
When recompiling simple, it is recommended to use the precompiled version from ```https://github.com/wangfenjin/simple```.  
Note to modify the Windows compilation script, remove the ```libgcc_s_seh-1.dll``` and ```libstdc++-6.dll``` dependencies for mingw64 compilation, and turn off ```BUILD_TEST_EXAMPLE``` as there are conflicts.  
```bat
rmdir /q /s build
mkdir build && cd build
cmake .. -G "Unix Makefiles" -DBUILD_TEST_EXAMPLE=OFF -DCMAKE_INSTALL_PREFIX=release -DCMAKE_CXX_FLAGS="-static-libgcc -static-libstdc++" -DCMAKE_EXE_LINKER_FLAGS="-Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic"
make && make install
```

## Backend Management Supports English
The minrag backend management currently supports both Chinese and English, with the capability to extend to other languages. Language files are located in ```minragdatadir/locales```. By default, the system uses Chinese (```zh-CN```) upon initial installation. If English is preferred, you can modify the ```"locale":"zh-CN"``` to ```"locale":"en-US"``` in the ```minragdatadir/install_config.json``` file before installation. Alternatively, after successful installation, you can change the ```Language``` setting to ```English``` in the ```Settings``` and restart the system to apply the changes.

## Table Structure  
ID defaults to timestamp (23 digits) + random number (9 digits), globally unique.  
Table creation statement ```minragdatadir/minrag.sql```  

### Configuration (Table Name: config)
Reads ```minragdatadir/install_config.json``` during installation.

| columnName  | Type| Description | Remarks       | 
|-|---|-|-----|
| id  | string      | Primary Key | minrag_config |
| basePath    | string      | Base Path   | Default /     |
| jwtSecret   | string      | JWT Secret Key      | Randomly generated |
| jwttokenKey | string      | JWT Key     | Default jwttoken |
| serverPort  | string      | IP:Port     | Default :738  |
| timeout     | int | JWT Timeout in seconds | Default 7200 |
| maxRequestBodySize  | int | Maximum Request Body Size | Default 20M |
| locale      | string      | Language Pack       | Default zh-CN,en-US |
| proxy       | string      | HTTP Proxy Address  |       |
| createTime  | string      | Creation Time       | 2006-01-02 15:04:05 |
| updateTime  | string      | Update Time | 2006-01-02 15:04:05 |
| createUser  | string      | Creator     | Initialized as system |
| sortNo      | int | Sort Order  | Descending    |
| status      | int | Status      | Disabled(0), Enabled(1) |

### User (Table Name: user)
| columnName  | Type| Description | Remarks       | 
|---|---|-|-----|
| id  | string      | Primary Key | minrag_admin  |
| account     | string      | Login Name  | Default admin |
| passWord    | string      | Password    | -     |
| userName    | string      | Description | -     |
| createTime  | string      | Creation Time       | 2006-01-02 15:04:05 |
| updateTime  | string      | Update Time | 2006-01-02 15:04:05 |
| createUser  | string      | Creator     | Initialized as system |
| sortNo      | int | Sort Order  | Descending    |
| status      | int | Status      | Disabled(0), Enabled(1) |

### Site Information (Table Name: site)
| columnName    | Type| Description | Remarks       | 
|-----|---|-|-----|
| id    | string      | Primary Key | minrag_site   |
| title | string      | Site Name   | -     |
| keyword       | string      | Keywords    | -     |
| description   | string      | Site Description    | -     |
| theme | string      | Default Theme       | Default is default |
| themePC       | string      | PC Theme    | Fetched from cookie first, if not, from Header, then written to cookie, default is default |
| themeWAP      | string      | Mobile Theme| Fetched from cookie first, if not, from Header, then written to cookie, default is default |
| themeWX       | string      | WeChat Theme| Fetched from cookie first, if not, from Header, then written to cookie, default is default |
| logo  | string      | Logo| -     |
| favicon       | string      | Favicon     | -     |
| createTime    | string      | Creation Time       | 2006-01-02 15:04:05 |
| updateTime    | string      | Update Time | 2006-01-02 15:04:05 |
| createUser    | string      | Creator     | Initialized as system |
| sortNo| int | Sort Order  | Descending    |
| status| int | Status      | Disabled(0), Enabled(1) |

### Knowledge Base (Table Name: knowledgeBase)
| columnName  | Type| Description | Remarks       | 
|-|---|-|-----|
| id  | string      | Primary Key | URL path, separated by /, e.g., /web/ |
| name| string      | Knowledge Base Name | -     |
| pid | string      | Parent Knowledge Base ID | Parent Knowledge Base ID |
| knowledgeBaseType   | int | Knowledge Base Type | -     |
| createTime  | string      | Creation Time       | 2006-01-02 15:04:05 |
| updateTime  | string      | Update Time | 2006-01-02 15:04:05 |
| createUser  | string      | Creator     | Initialized as system |
| sortNo      | int | Sort Order  | Descending    |
| status      | int | Status      | Disabled(0), Enabled(1) |

### Document (Table Name: document)
| columnName  | Type| Description | Tokenized | Remarks  | 
|-|---|-|-|------|
| id  | string      | Primary Key | No| URL path, separated by /, e.g., /web/nginx-use-hsts |
| name| string      | Document Name       | No| -|
| knowledgeBaseID     | string      | Knowledge Base ID   | No| -|
| knowledgeBaseName   | string      | Knowledge Base Name | No| -|
| toc | string      | Table of Contents   | No| -|
| summary     | string      | Summary     | No| -|
| markdown    | string      | Markdown Content    | No| -|
| filePath    | string      | File Path   | No| -|
| fileSize    | int | File Size   | No| -|
| fileExt     | string      | File Extension      | No| -|
| createTime  | string      | Creation Time       | - | 2006-01-02 15:04:05      |
| updateTime  | string      | Update Time | - | 2006-01-02 15:04:05      |
| createUser  | string      | Creator     | - | Initialized as system    |
| sortNo      | int | Sort Order  | - | Descending       |
| status      | int | Status      | - | Disabled(0), Enabled(1), Processing(2), Failed(3) |

### Document Chunk (Table Name: document_chunk)
| columnName  | Type| Description | Tokenized | Remarks  | 
|-|---|-|-|------|
| id  | string      | Primary Key | No| -|
| documentID  | string      | Document ID | No| -|
| knowledgeBaseID     | string      | Knowledge Base ID   | No| -|
| knowledgeBaseName   | string      | Knowledge Base Name | No| -|
| markdown    | string      | Markdown Content    | Yes       | Using jieba tokenizer    |
| createTime  | string      | Creation Time       | - | 2006-01-02 15:04:05      |
| updateTime  | string      | Update Time | - | 2006-01-02 15:04:05      |
| createUser  | string      | Creator     | - | Initialized as system    |
| sortNo      | int | Sort Order  | - | Descending       |
| status      | int | Status      | - | Disabled(0), Enabled(1), Processing(2), Failed(3) |

### Component (Table Name: component)
| columnName  | Type| Description | Tokenized | Remarks  | 
|-|---|-|-|------|
| id  | string      | Primary Key | No| -|
| componentType       | string      | Component Type      | No| -|
| parameter   | string      | Component Parameters| No| -|
| createTime  | string      | Creation Time       | - | 2006-01-02 15:04:05      |
| updateTime  | string      | Update Time | - | 2006-01-02 15:04:05      |
| createUser  | string      | Creator     | - | Initialized as system    |
| sortNo      | int | Sort Order  | - | Descending       |
| status      | int | Status      | - | Disabled(0), Enabled(1)  |

### Agent (Table Name: agent)
| columnName  | Type| Description | Tokenized | Remarks  | 
|-|---|-|-|------|
| id  | string      | Primary Key | No| -|
| name| string      | Agent Name  | No| -|
| knowledgeBaseID     | string      | Knowledge Base ID   | No| -|
| pipelineID  | string      | Pipeline ID | No| -|
| defaultReply| string      | Default Reply       | No| -|
| agentType   | int | Agent Type  | No| -|
| agentPrompt | string      | Agent Prompt| No| -|
| avatar      | string      | Agent Avatar| No| -|
| welcome     | string      | Welcome Message     | No| -|
| tools       | string      | Functions to Call   | No| -|
| memoryLength| int | Context Memory Length| No| -|
| createTime  | string      | Creation Time       | - | 2006-01-02 15:04:05      |
| updateTime  | string      | Update Time | - | 2006-01-02 15:04:05      |
| createUser  | string      | Creator     | - | Initialized as system    |
| sortNo      | int | Sort Order  | - | Descending       |
| status      | int | Status      | - | Disabled(0), Enabled(1)  |