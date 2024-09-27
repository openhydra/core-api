package v1

type KnowledgeBase struct {
	KbId              string `json:"kb_id,omitempty"`
	UserID            string `json:"user_id,omitempty"`
	Username          string `json:"username,omitempty"`
	KnowledgeBaseName string `json:"kb_name,omitempty"`
	VectorStoreType   string `json:"vector_store_type,omitempty"`
	KBInfo            string `json:"kb_info,omitempty"`
	EmbedModel        string `json:"embed_model,omitempty"`
	VSType            string `json:"vs_type,omitempty"`
	FileCount         int    `json:"file_count,omitempty"`
	CreateTime        string `json:"create_time,omitempty"`
	IsPrivate         bool   `json:"is_private,omitempty"`
	ChunkSize         int    `json:"chunk_size,omitempty"`
	ChunkOverlap      int    `json:"chunk_overlap,omitempty"`
}

type KnowledgeBaseFilesToDelete struct {
	FileNames []string `json:"file_names,omitempty"`
}

type KnowledgeBaseFileList struct {
	Message string                    `json:"message,omitempty"`
	Data    []knowledgeBaseFileDetail `json:"data,omitempty"`
	Code    int                       `json:"code,omitempty"`
}

type knowledgeBaseFileDetail struct {
	KbName         string  `json:"kb_name,omitempty"`
	FileName       string  `json:"file_name,omitempty"`
	FileExt        string  `json:"file_ext,omitempty"`
	FileVersion    int     `json:"file_version,omitempty"`
	DocumentLoader string  `json:"document_loader,omitempty"`
	InFolder       bool    `json:"in_folder,omitempty"`
	TextSplitter   string  `json:"text_splitter,omitempty"`
	DocsCount      int     `json:"docs_count,omitempty"`
	CreateTime     string  `json:"create_time,omitempty"`
	InDb           bool    `json:"in_db,omitempty"`
	FileMTime      float32 `json:"file_mtime,omitempty"`
	FileSize       uint    `json:"file_size,omitempty"`
	CustomDocs     bool    `json:"custom_docs,omitempty"`
	No             int     `json:"no,omitempty"`
}

type KnowledgeBaseCommonResult struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
