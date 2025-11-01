# 构建阶段
FROM rockylinux/rockylinux:10.0 AS builder

# 安装编译依赖
RUN dnf update -y && dnf install -y gcc g++ unzip wget

# 设置工作目录
WORKDIR /app

RUN wget https://golang.google.cn/dl/go1.25.3.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go1.25.3.linux-amd64.tar.gz

# 设置国内代理
#RUN /usr/local/go/bin/go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN /usr/local/go/bin/go env -w GOPROXY=https://goproxy.cn,direct

# 复制项目代码
COPY . .

# 编译项目
RUN /usr/local/go/bin/go build --tags "fts5" -ldflags "-w -s" -o minrag

# 初始化文件
RUN rm -rf /app/minragdatadir/dict && \
    unzip /app/minragdatadir/dict.zip -d /app/minragdatadir && \
    rm -rf /app/minragdatadir/dict.zip

# 构建markitdown
FROM python:3.12.12 AS markitdown

# 设置工作目录
WORKDIR /app
RUN apt install git -y && \
git clone https://gitee.com/minrag/markitdown.git && \
cd /app/markitdown && \
pip install PyInstaller && \
## 安装依赖
pip install -e 'packages/markitdown[all]' && \
## 编译打包
python3 build.py


# 运行阶段,vec0 需要依赖glibc,使用rockylinux镜像
FROM rockylinux/rockylinux:10.0

# 安装运行时依赖
RUN dnf update -y 
##RUN dnf install -y libgcc libstdc++ sqlite

# 设置工作目录
WORKDIR /app

RUN mkdir -p ./minragdatadir/markitdown

# 复制编译产物
COPY --from=builder /app/minrag .
COPY --from=builder /app/minragdatadir ./minragdatadir/
COPY --from=markitdown /app/markitdown/dist/markitdown ./minragdatadir/markitdown/

# 暴露端口
EXPOSE 738

# 设置数据卷
VOLUME ["/app/minragdatadir"]

# 启动命令
CMD ["./minrag"]