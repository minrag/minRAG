<img src="minragdatadir/public/minrag-logo.png" height="150px" />    

<a href="./README.md">English</a> | <a href="./README.zh-CN.md">简体中文</a>         
## Introduction  
minrag pursues the ultimate simplicity and power, with no more than 10,000 lines of code, to implement the core functions of dify.     
Fork the code of [gpress v1.0.8](https://gitee.com/gpress/gpress), no installation required, just double-click to start and use it.    

## Development Environment  
minrag uses ```https://github.com/wangfenjin/simple``` as the FTS5 full-text search extension. The compiled libsimple file is placed in the ```minragdatadir/extensions``` directory. If minrag fails to start and reports an error connecting to the database, please check if the libsimple file is correct. If you need to recompile libsimple, please refer to https://github.com/wangfenjin/simple.  

The default port is 738, and the backend management address is http://127.0.0.1:738/admin/login.  
First, unzip ```minragdatadir/dict.zip```.  
Run ```go run --tags "fts5" .```.  
Package: ```go build --tags "fts5" -ldflags "-w -s"```.  

The development environment requires CGO compilation configuration. Set ```set CGO_ENABLED=1```, download [mingw64](https://github.com/niXman/mingw-builds-binaries/releases) and [cmake](https://cmake.org/download/), and configure the bin to the environment variables. Note to rename ```mingw64/bin/mingw32-make.exe``` to ```make.exe```.  
Modify vscode's launch.json to add ``` ,"buildFlags": "--tags=fts5" ``` for debugging fts5.  
Test needs to be done manually: ```go test -v -timeout 30s --tags "fts5" -run ^TestVecQuery$ gitee.com/minrag/minrag```.  
Package: ```go build --tags "fts5" -ldflags "-w -s"```.  
When recompiling simple, it is recommended to use the precompiled version from ```https://github.com/wangfenjin/simple```.  
Note to modify the Windows compilation script, remove the ```libgcc_s_seh-1.dll``` and ```libstdc++-6.dll``` dependencies for mingw64 compilation, and turn off ```BUILD_TEST_EXAMPLE``` as there are conflicts.  
```bat
rmdir /q /s build
mkdir build && cd build
cmake .. -G "Unix Makefiles" -DBUILD_TEST_EXAMPLE=OFF -DCMAKE_INSTALL_PREFIX=release -DCMAKE_CXX_FLAGS="-static-libgcc -static-libstdc++" -DCMAKE_EXE_LINKER_FLAGS="-Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic"
make && make install
```

## Staticization  
The backend ```Refresh Site``` function will generate static HTML files to the ```statichtml``` directory, along with ```gzip_static``` files. You need to copy the ```css, js, image``` of the currently used theme and the ```minragdatadir/public``` directory to the ```statichtml``` directory, or use Nginx reverse proxy to specify the directory without copying files.  
Nginx configuration example:
```conf
### CSS files of the current theme (default)
location ~ ^/css/ {
    #gzip_static on;
    root /data/minrag/minragdatadir/template/theme/default;  
}
### JS files of the current theme (default)
location ~ ^/js/ {
    #gzip_static on;
    root /data/minrag/minragdatadir/template/theme/default;  
}
### Image files of the current theme (default)
location ~ ^/image/ {
    root /data/minrag/minragdatadir/template/theme/default;  
}
### search-data.json FlexSearch JSON data
location ~ ^/public/search-data.json {
    #gzip_static on;
    root /data/minrag/minragdatadir;  
}
### Public files
location ~ ^/public/ {
    root /data/minrag/minragdatadir;  
}
    
### Admin backend management, request dynamic service
location ~ ^/admin/ {
    proxy_redirect     off;
    proxy_set_header   Host      $host;
    proxy_set_header   X-Real-IP $remote_addr;
    proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header   X-Forwarded-Proto $scheme;
    proxy_pass  http://127.0.0.1:738;  
}
### Static HTML directory
location / {
    proxy_redirect     off;
    proxy_set_header   Host      $host;
    proxy_set_header   X-Real-IP $remote_addr;
    proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header   X-Forwarded-Proto $scheme; 
    ## If there is a q query parameter, use the dynamic service. Also supports FlexSearch parsing public/search-data.json
    if ($arg_q) { 
       proxy_pass  http://127.0.0.1:738;  
       break;
    }

    ### Enable gzip static compression
    #gzip_static on;

    ### Nginx 1.26+ does not need to 302 redirect to the index.html under the directory, gzip_static will also take effect. This configuration is kept for record.
    ##if ( -d $request_filename ) {
        ## Not ending with /
    ##    rewrite [^\/]$ $uri/index.html redirect;
        ## Ending with /
    ##    rewrite ^(.*) ${uri}index.html redirect;      
    ##}
    
    ### Static file directory of the current theme (default)
    root   /data/minrag/minragdatadir/statichtml/default;
    
    ### if directive may conflict with try_files directive, causing try_files to be invalid
    ## Avoid directory 301 redirect, e.g., /about will 301 to /about/           
    try_files $uri $uri/index.html;
    
    index  index.html index.htm;
}

```  

## Backend Management Supports English
The minrag backend management currently supports both Chinese and English, with the capability to extend to other languages. Language files are located in ```minragdatadir/locales```. By default, the system uses Chinese (```zh-CN```) upon initial installation. If English is preferred, you can modify the ```"locale":"zh-CN"``` to ```"locale":"en-US"``` in the ```minragdatadir/install_config.json``` file before installation. Alternatively, after successful installation, you can change the ```Language``` setting to ```English``` in the ```Settings``` and restart the system to apply the changes.

## Table Structure  
ID defaults to timestamp (23 digits) + random number (9 digits), globally unique.  
Table creation statement ```minragdatadir/minrag.sql```          

### Configuration (Table Name: config)
Reads ```minragdatadir/install_config.json``` during installation.

| columnName  | Type        | Description         |  Remarks       | 
| ----------- | ----------- | ----------- | ----------- |
| id          | string      | Primary Key        |minrag_config |
| basePath    | string      | Base Path    |  Default /      |
| jwtSecret   | string      | JWT Secret     | Randomly generated     |
| jwttokenKey | string      | JWT Key    |  Default jwttoken  |
| serverPort  | string      | IP:Port     |  Default :738  |
| timeout     | int         | JWT Timeout Seconds|  Default 7200  |
| maxRequestBodySize | int  | Max Request Size     |  Default 20M  |
| locale      | string      | Language Pack       |  Default zh-CN,en-US |
| proxy       | string      | HTTP Proxy Address |             |
| createTime  | string      | Creation Time     |  2006-01-02 15:04:05  |
| updateTime  | string      | Update Time     |  2006-01-02 15:04:05  |
| createUser  | string      | Creator       |  Initialization system  |
| sortNo      | int         | Sort Order         |  Ascending  |
| status      | int         | Status     |  Link Access (0), Public (1), Top (2), Disable (3)  |

### User (Table Name: user)
There is only one user in the backend.

| columnName  | Type         | Description        |  Remarks       | 
| ----------- | ----------- | ----------- | ----------- |
| id          | string      | Primary Key        | minrag_admin |
| account     | string      | Login Name    |  Default admin  |
| passWord    | string      | Password        |    -  |
| userName    | string      | Description        |    -  |
| createTime  | string      | Creation Time     |  2006-01-02 15:04:05  |
| updateTime  | string      | Update Time     |  2006-01-02 15:04:05  |
| createUser  | string      | Creator       |  Initialization system  |
| sortNo      | int         | Sort Order         |  Ascending  |
| status      | int         | Status     |  Link Access (0), Public (1), Top (2), Disable (3)  |

### Site Information (Table Name:site)
Site information, such as title, logo, keywords, description, etc.

| columnName    | Type         | Description    |  Remarks       | 
| ----------- | ----------- | ----------- | ----------- |
| id          | string      | Primary Key        |minrag_site  |
| title       | string      | Site Name     |     -  |
| keyword     | string      | Keywords       |     -  |
| description | string      | Site Description    |     -  |
| theme       | string      | Default Theme     | Default uses default  |
| themePC     | string      | PC Theme      | First get from cookie, if not, get from Header, write to cookie, default uses default  |
| themeWAP    | string      | Mobile Theme    | First get from cookie, if not, get from Header, write to cookie, default uses default  |
| themeWX     | string      | WeChat Theme    | First get from cookie, if not, get from Header, write to cookie, default uses default  |
| logo        | string      | Logo       |     -  |
| favicon     | string      | Favicon    |     -  |
| createTime  | string      | Creation Time     |  2006-01-02 15:04:05  |
| updateTime  | string      | Update Time     |  2006-01-02 15:04:05  |
| createUser  | string      | Creator       |  Initialization system  |
| sortNo      | int         | Sort Order         |  Ascending  |
| status      | int         | Status     |  Link Access (0), Public (1), Top (2), Disable (3)  |

### Knowledge Base (Table Name: knowledgeBase)
| columnName    | Type         | Description    |  Remarks       | 
| ----------- | ----------- | ----------- | ----------- |
| id          | string      | Primary Key         | URL path, separated by /, e.g., /web/ |
| name        | string      | Knowledge Name     |    -  |
| hrefURL     | string      | Redirect Path     |    -  |
| hrefTarget  | string      | Redirect Method     | _self,_blank,_parent,_top|
| pid         | string      | Parent Knowledge ID     | Parent Knowledge ID  |
| templateFile  | string      | Template File       | Current knowledge page template  |
| childTemplateFile  | string | Child Theme Template File  | Default template for child pages, if not set, default uses this template |
| keyword     | string      | Knowledge Keywords   | Yes      |        |
| description | string      | Knowledge Description     | Yes      |        |
| createTime  | string      | Creation Time     |  2006-01-02 15:04:05  |
| updateTime  | string      | Update Time     |  2006-01-02 15:04:05  |
| createUser  | string      | Creator       |  Initialization system  |
| sortNo      | int         | Sort Order         |  Ascending  |
| status      | int         | Status     |  Link Access (0), Public (1), Top (2), Disable (3)  |

### Document (Table Name: document)
| columnName  | Type        | Description        | Whether to Tokenize |  Remarks                  | 
| ----------- | ----------- | ----------- | ------- | ---------------------- |
| id          | string      | Primary Key         |   No    | URL path, separated by /, e.g., /web/nginx-use-hsts |
| title       | string      | Title     | Yes      |     Uses jieba tokenizer    |
| keyword     | string      | Document Keywords   | Yes      |     Uses jieba tokenizer    |
| description | string      | Document Description     | Yes      |     Uses jieba tokenizer    |
| hrefURL     | string      | Self Page Path | No      |    -                    |
| subtitle    | string      | Subtitle       | Yes      |      Uses jieba tokenizer  |
| author      | string      | Author         | Yes      |      Uses jieba tokenizer  |
| tag         | string      | Tags         | Yes      |      Uses jieba tokenizer  |
| toc         | string      | Table of Documents         | Yes      |      Uses jieba tokenizer  |
| summary     | string      | Summary         | Yes      |      Uses jieba tokenizer  |
| knowledgeBaseName| string      | Knowledge Base, separated by comma (,)| Yes| Uses jieba tokenizer.      |
| knowledgeBaseID  | string      | Knowledge ID       | No      | -                       |
| templateFile| string      | Template File     | No      | Template                    |
| document     | string      | Document     | No      |                         |
| markdown    | string      | Markdown Document | No      |                         |
| thumbnail   | string      | Cover Image       | No      |                         |
| signature   | string      | Disable Key Signature of Document | No   |                         |
| signAddress | string      | Signature Address   | No   |                         |
| signChain   | string      | Chain of Address | No   |                         |
| txID        | string      | On-chain Transaction Hash  | No   |                         |
| createTime  | string      | Creation Time     | -       |  2006-01-02 15:04:05    |
| updateTime  | string      | Update Time     | -       |  2006-01-02 15:04:05    |
| createUser  | string      | Creator       | -       |  Initialization system          |
| sortNo      | int         | Sort Order         | -       |  Ascending                   |
| status      | int         | Status     | -       |  Link Access (0), Public (1), Top (2), Disable (3)  |