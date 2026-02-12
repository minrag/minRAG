CREATE TABLE IF NOT EXISTS config (
		id TEXT PRIMARY KEY   NOT NULL,
		base_path         TEXT NOT NULL,
		jwt_secret        TEXT NOT NULL,
		jwt_token_key      TEXT NOT NULL,
		server_port       TEXT NOT NULL,
		timeout          INT  NOT NULL,
		max_request_body_size INT NOT NULL,
		locale           TEXT NOT NULL,
		proxy            TEXT,
		ai_base_url        TEXT,
		ai_api_key         TEXT,
		llm_model         TEXT,
		create_time       TEXT,
		update_time       TEXT,
		create_user       TEXT,
		sortno           int,
		status           INT  
	 ) strict ;

CREATE TABLE IF NOT EXISTS user (
		id TEXT PRIMARY KEY   NOT NULL,
		account          TEXT NOT NULL,
		password         TEXT NOT NULL,
		user_name         TEXT NOT NULL,
		create_time       TEXT,
		update_time       TEXT,
		create_user       TEXT,
		sortno           int,
		status           INT  
	 ) strict ;

CREATE TABLE IF NOT EXISTS knowledge_base (
		id TEXT PRIMARY KEY     NOT NULL,
		name              TEXT  NOT NULL,
		pid               TEXT,
        knowledge_base_type INT NOT NULL,
		create_time        TEXT,
		update_time        TEXT,
		create_user        TEXT,
		sortno            INT NOT NULL,
		status            INT NOT NULL
	 ) strict ;

INSERT INTO knowledge_base (status,sortno,create_user,update_time,create_time,knowledge_base_type,pid,name,id) VALUES (1,1,'','2025-01-31 10:24:00','2025-01-31 10:24:00',0,'','默认知识库','/default/');

CREATE TABLE IF NOT EXISTS document (
		id TEXT PRIMARY KEY    NOT NULL,
		name              TEXT NOT NULL,
		knowledge_base_id   TEXT,
		knowledge_base_name TEXT,
		toc               TEXT,
		summary           TEXT,
		markdown          TEXT,
		file_path          TEXT,
		file_size          INT,
		file_ext           TEXT,
		create_time        TEXT,
		update_time        TEXT,
		create_user        TEXT,
		sortno            INT NOT NULL,
		status            INT NOT NULL
	 ) strict ;

CREATE TABLE IF NOT EXISTS document_chunk (
		id TEXT PRIMARY KEY    NOT NULL,
		document_id        TEXT NOT NULL,
		knowledge_base_id   TEXT NOT NULL,
		title             TEXT,
		markdown          TEXT,
		parent_id          TEXT,
		pre_id             TEXT,
		next_id            TEXT,
		level             INT,
		create_time        TEXT,
		update_time        TEXT,
		create_user        TEXT,
		sortno            INT NOT NULL,
		status            INT NOT NULL
	 ) strict ;


CREATE TABLE IF NOT EXISTS component (
		id TEXT PRIMARY KEY NOT NULL,
		component_type     TEXT NOT NULL,
		parameter         TEXT,
		create_time        TEXT,
		update_time        TEXT,
		create_user        TEXT,
		sortno            INT NOT NULL,
		status            INT NOT NULL
	 ) strict ;
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,1,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"tikaURL":"http://localhost:9998/tika"}','TikaConverter','TikaConverter');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,2,'','2025-10-24 10:24:00','2025-10-24 10:24:00','','MarkdownConverter','MarkdownConverter');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,3,'','2025-10-24 10:24:00','2025-10-24 10:24:00','','WebScraper','WebScraper');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,4,'','2025-10-24 10:24:00','2025-10-24 10:24:00','','HtmlCleaner','HtmlCleaner');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,5,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"splitBy":["\f", "\n\n", "\n", "。", "!", ".", ";", "，", ",", " "],"splitLength":500,"splitOverlap":0}','DocumentSplitter','DocumentSplitter');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,6,'','2025-10-24 10:24:00','2025-10-24 10:24:00','','MarkdownIndex','MarkdownIndex');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,7,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"webURL":"https://www.bing.com/search?q=","querySelector":["li.b_algo div.b_tpcn"],"depth":2,"top_n":3}','WebSearch','WebSearch');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,8,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"model":"lke-text-embedding-v2"}','LKEDocumentEmbedder','LKEDocumentEmbedder');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,9,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"model":"Qwen3-Embedding-8B"}','OpenAIDocumentEmbedder','OpenAIDocumentEmbedder');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,10,'','2025-10-24 10:24:00','2025-10-24 10:24:00','','SQLiteVecDocumentStore','SQLiteVecDocumentStore');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,11,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"model":"lke-text-embedding-v2"}','LKETextEmbedder','LKETextEmbedder');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,12,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"model":"Qwen3-Embedding-8B"}','OpenAITextEmbedder','OpenAITextEmbedder');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,13,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"top_n":5,"score":0.0}','VecEmbeddingRetriever','VecEmbeddingRetriever');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,14,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"top_n":5,"score":0.0}','FtsKeywordRetriever','FtsKeywordRetriever');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,15,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"promptTemplate":"配合search_document_toc_by_id工具函数进行调用,文档列表:\n {{if .documents}} {{ range $i,$v := .documents }} 文档ID:{{ $v.Id }} \n 文档名称:{{ $v.Name }} \n 文档摘要:{{ $v.Summary }}  \n\n {{end }}{{end}}"}','MarkdownRetriever','MarkdownRetriever');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,16,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"model":"lke-reranker-base","top_n":5,"score":0.1}','LKEDocumentChunkReranker','LKEDocumentChunkReranker');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,17,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"model":"qwen3-reranker-8b","top_n":5,"score":0.1}','QianFanDocumentChunkReranker','QianFanDocumentChunkReranker');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,18,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"base_url":"https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank","model":"qwen3-rerank","return_documents":true,"top_n":5,"score":0.1}','BaiLianDocumentChunkReranker','BaiLianDocumentChunkReranker');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,19,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"model":"Qwen3-Reranker-8B","top_n":5,"score":0.1}','DocumentChunkReranker','DocumentChunkReranker');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,20,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"promptTemplate":"{{if .webSerachDocuments}} 联网搜索的相关网页内容,忽略内容中无关广告信息.{{ range $i,$v := .webSerachDocuments }} {{ $v.Name }} \n {{ $v.Id }} \n {{ $v.Markdown }} \n\n {{end }}{{end}} {{if .documentChunks}}根据文档,请回答问题,回复中不要说提供了文档.\n 文档: \n {{ range $i,$v := .documentChunks }} {{ $v.Markdown }} \n {{end }} \n{{end}}问题: {{ .query }} \n回答:"}','PromptBuilder','PromptBuilder');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,21,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"memory_length":3}','OpenAIChatMemory','OpenAIChatMemory');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,22,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"maxDeep":10}','OpenAIChatGenerator','OpenAIChatGenerator');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,23,'','2025-10-24 10:24:00','2025-10-24 10:24:00','','ChatMessageLogStore','ChatMessageLogStore');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,24,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"id":"indexPipeline","downStream":[{"id":"MarkdownConverter","downStream":[{"id":"DocumentSplitter"}]},{"id":"DocumentSplitter","downStream":[{"id":"OpenAIDocumentEmbedder"}]},{"id":"OpenAIDocumentEmbedder","downStream":[{"id":"SQLiteVecDocumentStore"}]},{"id":"SQLiteVecDocumentStore"}]}','Pipeline','indexPipeline');
INSERT INTO component (status,sortno,create_user,update_time,create_time,parameter,component_type,id) VALUES (1,25,'','2025-10-24 10:24:00','2025-10-24 10:24:00','{"id":"default","downStream":[{"id":"OpenAITextEmbedder","downStream":[{"id":"VecEmbeddingRetriever"}]},{"id":"VecEmbeddingRetriever","downStream":[{"id":"FtsKeywordRetriever"}]},{"id":"FtsKeywordRetriever","downStream":[{"id":"DocumentChunkReranker"}]},{"id":"DocumentChunkReranker","downStream":[{"id":"PromptBuilder"}]},{"id":"PromptBuilder","downStream":[{"id":"OpenAIChatMemory"}]},{"id":"OpenAIChatMemory","downStream":[{"id":"OpenAIChatGenerator"}]},{"id":"OpenAIChatGenerator","downStream":[{"id":"ChatMessageLogStore"}]},{"id":"ChatMessageLogStore"}]}','Pipeline','default');


CREATE TABLE IF NOT EXISTS agent (
		id TEXT PRIMARY KEY NOT NULL,
		name              TEXT NOT NULL,
		knowledge_base_id   TEXT NOT NULL,
		pipeline_id        TEXT NOT NULL,
		default_reply      TEXT NOT NULL,
		agent_type         INT  NOT NULL,
		agent_prompt       TEXT NOT NULL,
		avatar            TEXT,
		welcome           TEXT,
		tools             TEXT,
		memory_length      INT,
		create_time        TEXT,
		update_time        TEXT,
		create_user        TEXT,
		sortno            INT NOT NULL,
		status            INT NOT NULL
	 ) strict ;
INSERT INTO agent (status,sortno,create_user,update_time,create_time,memory_length,tools,welcome,avatar,agent_prompt,agent_type,default_reply,pipeline_id,knowledge_base_id,name,id) VALUES (1,1,'','2025-10-24 10:24:00','2025-10-24 10:24:00',0,'','您好,有什么可以帮助您吗?','','我是一个AI私人助手',0,'非常抱歉,可用聊其他话题吗?','default','/default/','默认智能体','default');

CREATE TABLE IF NOT EXISTS chat_room (
		id TEXT PRIMARY KEY NOT NULL,
		name              TEXT NOT NULL,
		agent_id           TEXT NOT NULL,
		pipeline_id        TEXT NOT NULL,
		knowledge_base_id   TEXT NOT NULL,
		user_id            TEXT,
		create_time        TEXT NOT NULL
	 ) strict ;

CREATE TABLE IF NOT EXISTS message_log (
		id TEXT PRIMARY KEY NOT NULL,
		agent_id           TEXT NOT NULL,
		room_id            TEXT NOT NULL,
		pipeline_id        TEXT NOT NULL,
		knowledge_base_id   TEXT NOT NULL,
		user_message       TEXT NOT NULL,
		ai_message         TEXT NOT NULL,
		user_id            TEXT,
		create_time        TEXT NOT NULL
	 ) strict ;


CREATE TABLE IF NOT EXISTS site (
		id TEXT PRIMARY KEY NOT NULL,
		title         TEXT NOT NULL,
		name          TEXT NOT NULL,
		domain        TEXT,
		keyword       TEXT,
		description   TEXT,
		theme         TEXT NOT NULL,
		theme_pc       TEXT,
		theme_wap      TEXT,
		theme_wx       TEXT,
		logo          TEXT,
		favicon       TEXT,
		footer        TEXT,
		create_time    TEXT,
		update_time    TEXT,
		create_user    TEXT,
		sortno        INT NOT NULL,
		status        INT NOT NULL
	 ) strict ;
INSERT INTO site (status,sortno,create_user,update_time,create_time,footer,favicon,logo,theme_wx,theme_wap,theme_pc,theme,description,keyword,domain,name,title,id)VALUES (1,1,NULL,NULL,NULL,'<div class="copyright"><span class="copyright-year">&copy; 2008 - 2025 <span class="author">jiagou.com 版权所有 <a href=''https://beian.miit.gov.cn'' target=''_blank''>豫ICP备xxxxx号</a>   <a href=''http://www.beian.gov.cn/portal/registerSystemInfo?recordcode=xxxx''  target=''_blank''><img src=''/public/gongan.png''>豫公网安备xxxxx号</a></span></span></div>','public/favicon.png','public/logo.png','default','default','default','default','Web3内容平台,Hertz + Go template + FTS5全文检索,支持以太坊和百度超级链,兼容Hugo、WordPress生态,使用Wasm扩展插件,只需200M内存','minrag,web3,Hugo,WordPress,以太坊,百度超级链','https://jiagou.com','架构','jiagou','minrag_site');


CREATE VIRTUAL TABLE IF NOT EXISTS fts_document_chunk USING fts5 (
    id UNINDEXED,
    document_id UNINDEXED,
    knowledge_base_id UNINDEXED,
	title,
    markdown,
    sortno UNINDEXED,
    status UNINDEXED,
    tokenize = 'simple 0',
    content='document_chunk',
    content_rowid='rowid'
);

CREATE TRIGGER trigger_document_chunk_insert AFTER INSERT ON document_chunk
BEGIN
    INSERT INTO fts_document_chunk(rowid, id, document_id, knowledge_base_id, title, markdown, sortno, status)
    VALUES (new.rowid,new.id, new.document_id, new.knowledge_base_id, new.title, new.markdown, new.sortno, new.status);
END;

CREATE TRIGGER trigger_document_chunk_delete AFTER DELETE ON document_chunk
BEGIN
    INSERT INTO fts_document_chunk(fts_document_chunk, id, document_id, knowledge_base_id, title, markdown, sortno, status)
    VALUES ('delete', old.id, old.document_id, old.knowledge_base_id, old.title, old.markdown, old.sortno, old.status);
END;

CREATE TRIGGER trigger_document_chunk_update AFTER UPDATE ON document_chunk
BEGIN
    INSERT INTO fts_document_chunk(fts_document_chunk, rowid, id, document_id, knowledge_base_id, title, markdown, sortno, status)
    VALUES ('delete',old.rowid, old.id, old.document_id, old.knowledge_base_id, old.title, old.markdown, old.sortno, old.status);
    INSERT INTO fts_document_chunk(rowid, id, document_id, knowledge_base_id, title, markdown, sortno, status)
    VALUES (new.rowid, new.id, new.document_id, new.knowledge_base_id, new.title, new.markdown, new.sortno, new.status);
END;


CREATE VIRTUAL TABLE IF NOT EXISTS vec_document_chunk USING vec0(
	id TEXT,
    document_id TEXT,
    knowledge_base_id TEXT,
    embedding float[1024],
    sortno INT,
    status INT
);