package route

import (
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	"fmt"
	"net/http"
)

const (
	ragPrefix     = "/apis"
	ragAPIVersion = "v1"
	ragGroup      = "rag.openhydra.io"
)

var ragFullPath = fmt.Sprintf("%s/%s/%s", ragPrefix, ragGroup, ragAPIVersion)

func GetRagRoute(config *config.Config, stopChan <-chan struct{}) *ChiRouteBuilder {
	handler := getOrInitSouthRagHandler(config, stopChan)
	return &ChiRouteBuilder{
		PathPrefix: ragFullPath,
		MethodHandlers: []ChiSubRouteBuilder{
			{
				Method:  http.MethodGet,
				Handler: handler.GetConversation,
				Pattern: "/conversations",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			{
				Method:  http.MethodPost,
				Handler: handler.CreateConversation,
				Pattern: "/conversations",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			{
				Method:  http.MethodGet,
				Handler: handler.GetConversationById,
				Pattern: "/conversations/{conversationId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			{
				Method:  http.MethodDelete,
				Handler: handler.DeleteConversation,
				Pattern: "/conversations/{conversationId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagDelete,
				},
			},
			{
				Method:  http.MethodPatch,
				Handler: handler.PatchConversation,
				Pattern: "/conversations/{conversationId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagUpdate,
				},
			},
			{
				Method:  http.MethodGet,
				Handler: handler.GetConversationOfUser,
				Pattern: "/conversations/users/{userId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			{
				Method:  http.MethodDelete,
				Handler: handler.DeleteConversationOfUser,
				Pattern: "/conversations/users/{userId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagDelete,
				},
			},
			{
				Method:  http.MethodPost,
				Handler: handler.CreateKnowledgeBase,
				Pattern: "/knowledge_bases",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagCreate,
				},
			},
			// upload to knowledge base
			{
				Method:  http.MethodPost,
				Handler: handler.UploadFileKnowledgeBase,
				Pattern: "/knowledge_bases/{knowledgeBaseId}/upload",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagCreate,
				},
			},
			// delete knowledge base
			{
				Method:  http.MethodDelete,
				Handler: handler.DeleteKnowledgeBase,
				Pattern: "/knowledge_bases/{knowledgeBaseId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagDelete,
				},
			},
			// update knowledge base
			{
				Method:  http.MethodPatch,
				Handler: handler.PatchKnowledgeBase,
				Pattern: "/knowledge_bases/{knowledgeBaseId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagUpdate,
				},
			},
			{
				Method:  http.MethodGet,
				Handler: handler.GetPublicKnowledgeBases,
				Pattern: "/knowledge_bases",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			{
				Method:  http.MethodGet,
				Handler: handler.GetKnowledgeBases,
				Pattern: "/knowledge_bases/users/{userId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			// get kb detail
			{
				Method:  http.MethodGet,
				Handler: handler.GetKnowledgeBaseDetail,
				Pattern: "/knowledge_bases/{knowledgeBaseId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			{
				Method:  http.MethodPost,
				Handler: handler.CreateChat,
				Pattern: "/chats",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			// get conversation messages
			{
				Method:  http.MethodGet,
				Handler: handler.GetConversationMessages,
				Pattern: "/conversations/{conversationId}/messages",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			// get conversation message by id
			{
				Method:  http.MethodGet,
				Handler: handler.GetConversationMessageById,
				Pattern: "/conversations/{conversationId}/messages/{messageId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			// get chat quick starts
			{
				Method:  http.MethodGet,
				Handler: handler.GetChatQuickStarts,
				Pattern: "/chat_quick_starts",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			// upload file chat temp file
			{
				Method:  http.MethodPost,
				Handler: handler.FileChatHandler,
				Pattern: "/file_chat/{userId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			// upload file chat with quick start
			{
				Method:  http.MethodPost,
				Handler: handler.QuickFileChatHandler,
				Pattern: "/quick_file_chat",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			// file chat
			{
				Method:  http.MethodPost,
				Handler: handler.FileChat,
				Pattern: "/file_chat",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			// kb_chat
			{
				Method:  http.MethodPost,
				Handler: handler.KBChatHandler,
				Pattern: "/knowledge_bases/{knowledgeBaseId}/kb_chat",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			// get kb files
			{
				Method:  http.MethodGet,
				Handler: handler.GetKBFiles,
				Pattern: "/knowledge_bases/{knowledgeBaseId}/files",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			// delete kb files
			{
				Method:  http.MethodDelete,
				Handler: handler.DeleteKBFiles,
				Pattern: "/knowledge_bases/{knowledgeBaseId}/files",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagCreate,
				},
			},
		},
	}
}
