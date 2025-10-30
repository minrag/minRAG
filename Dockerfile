# 构建阶段
FROM golang:1.25.3-alpine3.22 AS builder
# 操作系统(linux/darwin/windows,默认linux)
ARG OS=linux     
# 架构(amd64/arm64,默认amd64)    
ARG ARCH=amd64        

# 安装编译依赖
RUN apk add --no-cache gcc g++ unzip

# 设置工作目录
WORKDIR /app

# 设置国内代理
#RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN go env -w GOPROXY=https://goproxy.cn,direct

# 复制项目代码
COPY . .

# 编译项目
RUN go build --tags "fts5" -ldflags "-w -s" -o minrag

# 初始化文件
RUN rm -rf /app/minragdatadir/dict && \
    unzip /app/minragdatadir/dict.zip -d /app/minragdatadir && \
    rm -rf /app/minragdatadir/dict.zip
   
### 这里可以增加编译 [markitdown](https://gitee.com/minrag/markitdown)  

# 运行阶段
FROM alpine:3.22.2

# 安装运行时依赖
RUN apk add --no-cache libgcc libstdc++ sqlite-libs


# 设置工作目录
WORKDIR /app

RUN mkdir -p ./minragdatadir

# 复制编译产物
COPY --from=builder /app/minrag .
COPY --from=builder /app/minragdatadir ./minragdatadir/

# 暴露端口
EXPOSE 738

# 设置数据卷
VOLUME ["/app/minragdatadir"]

# 启动命令
CMD ["./minrag"]