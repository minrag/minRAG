CREATE TABLE IF NOT EXISTS config (
		id TEXT PRIMARY KEY   NOT NULL,
		basePath         TEXT NOT NULL,
		jwtSecret        TEXT NOT NULL,
		jwttokenKey      TEXT NOT NULL,
		serverPort       TEXT NOT NULL,
		timeout          INT  NOT NULL,
		maxRequestBodySize INT NOT NULL,
		locale           TEXT NOT NULL,
		proxy            TEXT NULL,
		createTime       TEXT,
		updateTime       TEXT,
		createUser       TEXT,
		sortNo           int,
		status           INT  
	 ) strict ;

CREATE TABLE IF NOT EXISTS user (
		id TEXT PRIMARY KEY     NOT NULL,
		account         TEXT  NOT NULL,
		password         TEXT   NOT NULL,
		userName         TEXT NOT NULL,
		chainType        TEXT,
		chainAddress     TEXT,
		createTime       TEXT,
		updateTime       TEXT,
		createUser       TEXT,
		sortNo           int,
		status           INT  
	 ) strict ;

CREATE TABLE IF NOT EXISTS knowledgeBase (
		id TEXT PRIMARY KEY     NOT NULL,
		name          TEXT  NOT NULL,
		pid        TEXT,
        knowledgeBaseType     INT NOT NULL,
		createTime        TEXT,
		updateTime        TEXT,
		createUser        TEXT,
		sortNo            INT NOT NULL,
		status            INT NOT NULL
	 ) strict ;

INSERT INTO knowledgeBase (status,sortNo,createUser,updateTime,createTime,knowledgeBaseType,pid,name,id) VALUES (0,1,'','2025-01-31 10:24:00','2025-01-31 10:24:00',0,'','默认知识库','/default/');

CREATE TABLE IF NOT EXISTS document (
		id TEXT PRIMARY KEY     NOT NULL,
		name         TEXT   NOT NULL,
		knowledgeBaseID           TEXT,
		knowledgeBaseName           TEXT,
		toc           TEXT,
		summary           TEXT,
		markdown          TEXT,
		filePath          TEXT,
		fileSize          INT,
		fileExt           TEXT,
		createTime        TEXT,
		updateTime        TEXT,
		createUser        TEXT,
		sortNo            INT NOT NULL,
		status            INT NOT NULL
	 ) strict ;

CREATE TABLE IF NOT EXISTS site (
		id TEXT PRIMARY KEY     NOT NULL,
		title         TEXT  NOT NULL,
		name         TEXT   NOT NULL,
		domain         TEXT,
		keyword         TEXT,
		description         TEXT,
		theme         TEXT NOT NULL,
		themePC         TEXT,
		themeWAP         TEXT,
		themeWX        TEXT,
		logo         TEXT,
		favicon         TEXT,
		footer         TEXT,
		createTime        TEXT,
		updateTime        TEXT,
		createUser        TEXT,
		sortNo            INT NOT NULL,
		status            INT NOT NULL
	 ) strict ;
INSERT INTO site (status,sortNo,createUser,updateTime,createTime,footer,favicon,logo,themeWX,themeWAP,themePC,theme,description,keyword,domain,name,title,id)VALUES (1,1,NULL,NULL,NULL,'<div class="copyright"><span class="copyright-year">&copy; 2008 - 2025 <span class="author">jiagou.com 版权所有 <a href=''https://beian.miit.gov.cn'' target=''_blank''>豫ICP备xxxxx号</a>   <a href=''http://www.beian.gov.cn/portal/registerSystemInfo?recordcode=xxxx''  target=''_blank''><img src=''/public/gongan.png''>豫公网安备xxxxx号</a></span></span></div>','public/favicon.png','public/logo.png','default','default','default','default','Web3内容平台,Hertz + Go template + FTS5全文检索,支持以太坊和百度超级链,兼容Hugo、WordPress生态,使用Wasm扩展插件,只需200M内存','minrag,web3,Hugo,WordPress,以太坊,百度超级链','https://jiagou.com','架构','jiagou','minrag_site');
