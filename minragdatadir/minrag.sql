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

CREATE TABLE IF NOT EXISTS document_chunk (
		id TEXT PRIMARY KEY    NOT NULL,
		documentID        TEXT NOT NULL,
		knowledgeBaseID   TEXT NOT NULL,
		markdown          TEXT,
		createTime        TEXT,
		updateTime        TEXT,
		createUser        TEXT,
		sortNo            INT NOT NULL,
		status            INT NOT NULL
	 ) strict ;


CREATE TABLE IF NOT EXISTS component (
		id TEXT PRIMARY KEY NOT NULL,
		componentType     TEXT NOT NULL,
		parameter         TEXT,
		createTime        TEXT,
		updateTime        TEXT,
		createUser        TEXT,
		sortNo            INT NOT NULL,
		status            INT NOT NULL
	 ) strict ;
INSERT INTO component (status,sortNo,createUser,updateTime,createTime,parameter,componentType,id) VALUES (1,1,'','2025-02-02 19:45:25','2025-02-02 19:45:25','{"splitBy":["\f", "\n\n", "\n", "。", "!", ".", ";", "，", ",", " "],"splitLength":500,"splitOverlap":0}','DocumentSplitter','DocumentSplitter');
INSERT INTO component (status,sortNo,createUser,updateTime,createTime,parameter,componentType,id) VALUES (1,2,'','2025-02-02 19:45:25','2025-02-02 19:45:25','{"apikey":"A4FTACZVPGAIV8PZCKIBEUGV7ZBMXTIBEGUGNC11","model":"bge-m3","apiBaseURL":"https://ai.gitee.com/v1","defaultHeaders":{"X-Failover-Enabled": "true", "X-Package": "1910"}}','OpenAITextEmbedder','OpenAITextEmbedder');
INSERT INTO component (status,sortNo,createUser,updateTime,createTime,parameter,componentType,id) VALUES (1,3,'','2025-02-02 19:45:25','2025-02-02 19:45:25','{"topK":5,"score":0.1}','VecEmbeddingRetriever','VecEmbeddingRetriever');
INSERT INTO component (status,sortNo,createUser,updateTime,createTime,parameter,componentType,id) VALUES (1,4,'','2025-02-02 19:45:25','2025-02-02 19:45:25','{"topK":5,"score":0.1}','FtsKeywordRetriever','FtsKeywordRetriever');


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


CREATE VIRTUAL TABLE IF NOT EXISTS fts_document_chunk USING fts5 (
    id UNINDEXED,
    documentID UNINDEXED,
    knowledgeBaseID UNINDEXED,
    markdown ,
    sortNo UNINDEXED,
    status UNINDEXED,
    tokenize = 'simple 0',
    content='document_chunk',
    content_rowid='rowid'
);

CREATE TRIGGER trigger_document_chunk_insert AFTER INSERT ON document_chunk
BEGIN
    INSERT INTO fts_document_chunk(rowid, id, documentID, knowledgeBaseID, markdown, sortNo, status)
    VALUES (new.rowid,new.id, new.documentID, new.knowledgeBaseID, new.markdown, new.sortNo, new.status);
END;

CREATE TRIGGER trigger_document_chunk_delete AFTER DELETE ON document_chunk
BEGIN
    INSERT INTO fts_document_chunk(fts_document_chunk, id, documentID, knowledgeBaseID, markdown, sortNo, status)
    VALUES ('delete', old.id, old.documentID, old.knowledgeBaseID, old.markdown, old.sortNo, old.status);
END;

CREATE TRIGGER trigger_document_chunk_update AFTER UPDATE ON document_chunk
BEGIN
    INSERT INTO fts_document_chunk(fts_document_chunk, rowid, id, documentID, knowledgeBaseID, markdown, sortNo, status)
    VALUES ('delete',old.rowid, old.id, old.documentID, old.knowledgeBaseID, old.markdown, old.sortNo, old.status);
    INSERT INTO fts_document_chunk(rowid, id, documentID, knowledgeBaseID, markdown, sortNo, status)
    VALUES (new.rowid, new.id, new.documentID, new.knowledgeBaseID, new.markdown, new.sortNo, new.status);
END;


CREATE VIRTUAL TABLE IF NOT EXISTS vec_document_chunk USING vec0(
	id TEXT,
    documentID TEXT,
    knowledgeBaseID TEXT,
    embedding float[1024],
    sortNo INT,
    status INT
);