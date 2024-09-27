package south

import (
	"bytes"
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	coreApiLog "core-api/pkg/logger"
	chatV1 "core-api/pkg/north/api/chat/core/v1"
	conversationV1 "core-api/pkg/north/api/conversation/core/v1"
	knowledgeBaseV1 "core-api/pkg/north/api/knowledge_base/core/v1"
	coreUserV1 "core-api/pkg/north/api/user/core/v1"
	"core-api/pkg/util/common"
	customErr "core-api/pkg/util/error"
	httpHelper "core-api/pkg/util/http"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

type RAGSouthApiHandler struct {
	config         *config.Config
	groupedKBCache *sync.Map
	userCache      *sync.Map
	stopChan       <-chan struct{}
	innerStopChan  chan struct{}
}

func NewRAGSouthApiHandler(config *config.Config, stopChan <-chan struct{}) *RAGSouthApiHandler {
	return &RAGSouthApiHandler{
		config:         config,
		groupedKBCache: &sync.Map{},
		userCache:      &sync.Map{},
		stopChan:       stopChan,
		innerStopChan:  make(chan struct{}, 1),
	}
}

func (h *RAGSouthApiHandler) StopBackgroundCache() {
	if h.stopChan == nil {
		coreApiLog.Logger.Warn("stop channel is nil, background cache not running")
		return
	}
	h.innerStopChan <- struct{}{}
}

func (h *RAGSouthApiHandler) RunBackgroundCache() {
	if h.stopChan == nil {
		coreApiLog.Logger.Warn("stop channel is nil, background cache will not run")
		return
	}

	go func() {
		h.renewGroupedKBCache()
		ticker := time.Tick(10 * time.Second)
		for range ticker {
			// clear all cache
			//coreApiLog.Logger.Debug("background worker renewing grouped cache1")
			h.renewGroupedKBCache()
		}
	}()

	select {
	case <-h.stopChan:
		coreApiLog.Logger.Info("stop channel is closed, stopping background cache")
		return
	case <-h.innerStopChan:
		coreApiLog.Logger.Info("inner stop channel is closed, stopping background cache")
		return
	}
}

// note because rag app do not have group we have to aggregate all kb that user can access by their group
func (h *RAGSouthApiHandler) renewGroupedKBCache() {
	body, _, code, err := common.CommonRequest(fmt.Sprintf("%s/knowledge_base/list_knowledge_bases", h.config.Rag.Endpoint), http.MethodGet, "", nil, nil, false, true, 3*time.Second)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get knowledge bases, aborting renew grouped kb cache", "error", err)
		return
	}

	if code != http.StatusOK {
		coreApiLog.Logger.Error("Failed to get knowledge bases, aborting renew grouped kb cache", "code", code, "body", string(body))
		return
	}

	allKBsWrapper := &struct {
		Data []knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}
	err = json.Unmarshal(body, &allKBsWrapper)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal knowledge bases, aborting renew grouped kb cache", "error", err)
		return
	}

	userProvider, err := initOrGetUserProvider(h.config)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get user provider, aborting renew grouped kb cache", "error", err)
		return
	}

	allUsers, err := userProvider.GetUsers(nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get users, aborting renew grouped kb cache", "error", err)
		return
	}

	flatUsers := map[string]coreUserV1.CoreUser{}
	for _, user := range allUsers {
		flatUsers[user.Id] = user
	}

	tempUserCache := &sync.Map{}
	tempUserCache.Store("users", flatUsers)
	coreApiLog.Logger.Debug("renewed user cache", "len", len(flatUsers))
	// replace the old cache with the new one
	h.userCache = tempUserCache

	userGroupedKBs := map[string][]knowledgeBaseV1.KnowledgeBase{}
	// note groupedKBs for fast identification of kb that already exist
	groupedKBs := map[string]map[string]struct{}{}

	for _, kb := range allKBsWrapper.Data {
		if !kb.IsPrivate {
			// skip public kb because it will be included in all user's kb
			continue
		}
		// find who created this kb
		userFound, ok := flatUsers[kb.UserID]
		if !ok {
			coreApiLog.Logger.Warn("Find un-ref kb, kb will be ignored", "userId", kb.UserID, "kbName", kb.KnowledgeBaseName)
			continue
		}
		if len(userFound.Groups) == 0 {
			continue
		}

		flatUserGroups := map[string]struct{}{}
		for _, group := range userFound.Groups {
			flatUserGroups[group.Id] = struct{}{}
		}

		for _, user := range flatUsers {
			if userFound.Id == user.Id {
				// skip self because this kb will always be included in user's kb
				continue
			}
			groupedKBs[user.Id] = map[string]struct{}{}
			if len(user.Groups) == 0 {
				continue
			}
			for _, group := range user.Groups {
				if _, ok := flatUserGroups[group.Id]; ok {
					if _, ok := groupedKBs[user.Id][kb.KnowledgeBaseName]; !ok {
						groupedKBs[user.Id][kb.KnowledgeBaseName] = struct{}{}
						kb.Username = userFound.Name
						userGroupedKBs[user.Id] = append([]knowledgeBaseV1.KnowledgeBase{}, kb)
						break
					}
				}
			}
		}

	}

	tempSyncMap := &sync.Map{}
	tempSyncMap.Store("groupedKBs", userGroupedKBs)

	coreApiLog.Logger.Debug("renewed grouped kb cache", "data", userGroupedKBs)
	// replace the old cache with the new one
	h.groupedKBCache = tempSyncMap
}

func (h *RAGSouthApiHandler) GetConversation(w http.ResponseWriter, r *http.Request) {

	chatTypeRequested := map[string]struct{}{}
	chatType := r.URL.Query().Get("chatTypes")
	if chatType == "" {
		coreApiLog.Logger.Debug("chatType is empty, set it to all")
		chatTypeRequested["llm_chat"] = struct{}{}
		chatTypeRequested["file_chat"] = struct{}{}
		chatTypeRequested["kb_chat"] = struct{}{}
	} else {
		slicedChatType := strings.Split(chatType, ",")
		for _, chatType := range slicedChatType {
			if chatType != "all" && chatType != "llm_chat" && chatType != "file_chat" && chatType != "kb_chat" {
				httpHelper.WriteCustomErrorAndLog(w, "chatType must be either all or llm_chat or file_chat or kb_chat", http.StatusBadRequest, "", fmt.Errorf("chatType must be either all or llm_chat or file_chat or kb_chat"))
				return
			}
			chatTypeRequested[chatType] = struct{}{}
		}
	}

	body, headers, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations", h.config.Rag.Endpoint), r.Method, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation list", http.StatusInternalServerError, "", err)
		return
	}

	if status != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation list", status, "", fmt.Errorf("failed to get conversation list due to error: %s", string(body)))
		return
	}

	ForwardResponseHeader(w, headers)
	if chatType == "" {
		w.Write(body)
	} else {
		var conversations []conversationV1.Conversation
		err = json.Unmarshal(body, &conversations)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal conversation list", http.StatusInternalServerError, "", err)
			return
		}
		var result []conversationV1.Conversation
		for _, conversation := range conversations {
			if _, ok := chatTypeRequested[conversation.ChatType]; ok {
				result = append(result, conversation)
			}
		}

		if len(result) == 0 {
			w.Header().Set("Content-Length", "2")
			w.Write([]byte("[]"))
			return
		}

		resultBytes, err := json.Marshal(result)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal conversation list", http.StatusInternalServerError, "", err)
			return
		}
		// reset content-length
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(resultBytes)))
		w.Write(resultBytes)
	}

	w.WriteHeader(status)

}

func (h *RAGSouthApiHandler) createConversation(conversation *conversationV1.Conversation, r *http.Request) ([]byte, map[string][]string, int, error) {
	uerProvider, err := initOrGetUserProvider(h.config)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}

	_, err = uerProvider.GetUser(conversation.UserID, nil)
	if err != nil {
		// check is not found
		if customErr.IsNotFound(err) {
			return nil, nil, http.StatusNotFound, fmt.Errorf("user not found: %s", conversation.UserID)
		} else {
			return nil, nil, http.StatusInternalServerError, err
		}
	}

	if conversation.ChatType != "llm_chat" && conversation.ChatType != "file_chat" && conversation.ChatType != "kb_chat" {
		return nil, nil, http.StatusBadRequest, fmt.Errorf("chatType must be either llm_chat or file_chat or kb_chat")
	}

	bodyBytes, err := json.Marshal(conversation)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}

	typedConversationBytes, _, _, err := h.getConversationOfUser(conversation.UserID, conversation.ChatType, nil, r)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}

	typedConversation := []conversationV1.Conversation{}
	err = json.Unmarshal(typedConversationBytes, &typedConversation)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}

	limit := h.config.Rag.MaximumChatHistoryRecord
	switch conversation.ChatType {
	case "llm_chat":
		limit = h.config.Rag.MaximumChatHistoryRecord
	case "file_chat":
		limit = h.config.Rag.MaximumFileChatHistoryRecord
	case "kb_chat":
		limit = h.config.Rag.MaximumKbChatHistoryRecord
	}

	if len(typedConversation) >= limit {
		// remove last record
		coreApiLog.Logger.Debug("exceed the limit of chat history, remove the last record", "chatType", conversation.ChatType, "conversationId", conversation.ID)
		_, err = h.DeleteConversationNoForward(typedConversation[len(typedConversation)-1].ID)
		if err != nil {
			return nil, nil, http.StatusInternalServerError, fmt.Errorf("failed to delete the oldest conversation due to error: %s", err)
		}
	}

	body, header, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations", h.config.Rag.Endpoint), http.MethodPost, "", bodyBytes, httpHelper.GetCommonHttpHeader(nil), false, true, 3*time.Second)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}

	if status != http.StatusCreated {
		return nil, nil, status, fmt.Errorf("failed to create conversation due to error: %s", string(body))
	}

	return body, header, http.StatusCreated, nil
}

func (h *RAGSouthApiHandler) CreateConversationNoForward(conversation *conversationV1.Conversation, r *http.Request) (*conversationV1.Conversation, error) {

	newConversationBtyes, _, _, err := h.createConversation(conversation, r)
	if err != nil {
		return nil, err
	}

	newConversation := &conversationV1.Conversation{}
	err = json.Unmarshal(newConversationBtyes, newConversation)
	if err != nil {
		return nil, err
	}

	return newConversation, nil
}

func (h *RAGSouthApiHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	// parse request body to []byte
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusInternalServerError, "", err)
		return
	}

	conversation := &conversationV1.Conversation{}
	err = json.Unmarshal(body, conversation)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
		return
	}

	bodyResp, headers, status, err := h.createConversation(conversation, r)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create conversation", status, "", err)
		return
	}

	ForwardResponseHeader(w, headers)
	w.Write(bodyResp)
	w.WriteHeader(http.StatusCreated)
}

func (h *RAGSouthApiHandler) GetConversationById(w http.ResponseWriter, r *http.Request) {

	conversationId := chi.URLParam(r, "conversationId")
	if conversationId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Conversation id is required", http.StatusBadRequest, "", fmt.Errorf("conversation id is required"))
		return
	}

	body, headers, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations/%s", h.config.Rag.Endpoint, conversationId), r.Method, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation by id", http.StatusInternalServerError, "", err)
		return
	}

	if status != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation by id", status, "", fmt.Errorf("failed to get conversation by id due to error: %s", string(body)))
		return
	}

	conversation := &conversationV1.Conversation{}
	err = json.Unmarshal(body, conversation)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal conversation", http.StatusInternalServerError, "", err)
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, conversation.UserID, "conversation")
		if err != nil {
			return
		}
	}

	if conversation.TempKBId != "" && conversation.ChatType == "file_chat" {
		fileName, err := GetFileChatFileName(h.config.Rag.FileChatPath, conversation.TempKBId)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get file chat file name", http.StatusInternalServerError, "", err)
			return
		}
		conversation.TempFileName = fileName
	}

	ForwardResponseHeader(w, headers)
	w.Header().Del("Content-Length")
	httpHelper.WriteResponseEntity(w, conversation)
}

func (h *RAGSouthApiHandler) GetConversationByIdToModel(conversationId string) (*conversationV1.Conversation, error) {
	body, _, _, err := common.CommonRequest(fmt.Sprintf("%s/conversations/%s", h.config.Rag.Endpoint, conversationId), http.MethodGet, "", nil, nil, false, true, 3*time.Second)
	if err != nil {
		return nil, err
	}

	conversation := &conversationV1.Conversation{}
	err = json.Unmarshal(body, conversation)
	if err != nil {
		return nil, err
	}
	return conversation, nil
}

func (h *RAGSouthApiHandler) DeleteConversation(w http.ResponseWriter, r *http.Request) {

	conversationFound, err := h.GetConversationByIdToModel(chi.URLParam(r, "conversationId"))
	if err != nil {
		if customErr.IsNotFound(err) {
			httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Conversation with id: '%s' not found", chi.URLParam(r, "conversationId")), http.StatusNotFound, "", err)
			return
		}
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation by id", http.StatusInternalServerError, "", err)
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, conversationFound.UserID, "conversation")
		if err != nil {
			return
		}
	}

	body, headers, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations/%s", h.config.Rag.Endpoint, chi.URLParam(r, "conversationId")), r.Method, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete conversation", http.StatusInternalServerError, "", err)
		return
	}

	if status != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete conversation", status, "", fmt.Errorf("failed to delete conversation due to error: %s", string(body)))
		return
	}

	if conversationFound.TempKBId != "" && conversationFound.ChatType == "file_chat" {
		// delete temp chat file of file chat conversation
		err := os.RemoveAll(filepath.Join(h.config.Rag.FileChatPath, "data", "temp", conversationFound.TempKBId))
		if err != nil {
			coreApiLog.Logger.Warn("Failed to delete temp file, not much we can do here anymore", "error", err, "tempKBId", conversationFound.TempKBId, "conversationId", conversationFound.ID, "userId", conversationFound.UserID)
		}
	}

	ForwardResponseHeader(w, headers)
	w.Write(body)
	w.WriteHeader(status)
}

func (h *RAGSouthApiHandler) DeleteConversationNoForward(id string) (*conversationV1.Conversation, error) {
	_, _, _, err := common.CommonRequest(fmt.Sprintf("%s/conversations/%s", h.config.Rag.Endpoint, id), http.MethodDelete, "", nil, httpHelper.GetCommonHttpHeader(nil), false, true, 3*time.Second)
	if err != nil {
		return nil, err
	}

	conversation := &conversationV1.Conversation{
		ID: id,
	}
	return conversation, nil
}

func (h *RAGSouthApiHandler) PatchConversation(w http.ResponseWriter, r *http.Request) {

	conversationId := chi.URLParam(r, "conversationId")
	if conversationId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Conversation id is required", http.StatusBadRequest, "", fmt.Errorf("conversation id is required"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusInternalServerError, "", err)
		return
	}

	conversationFound, err := h.GetConversationByIdToModel(conversationId)
	if err != nil {
		if customErr.IsNotFound(err) {
			httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Conversation with id: '%s' not found", conversationId), http.StatusNotFound, "", err)
			return
		}
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation by id", http.StatusInternalServerError, "", err)
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, conversationFound.UserID, "conversation")
		if err != nil {
			return
		}
	}

	body, headers, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations/%s", h.config.Rag.Endpoint, conversationId), r.Method, "", body, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to patch conversation", http.StatusInternalServerError, "", err)
		return
	}

	ForwardResponseHeader(w, headers)
	w.Write(body)
	w.WriteHeader(status)
}

func (h *RAGSouthApiHandler) getConversationOfUser(userId, chatType string, header map[string][]string, r *http.Request) ([]byte, map[string][]string, int, error) {
	chatTypeRequested := map[string]struct{}{}
	if chatType == "" {
		coreApiLog.Logger.Debug("chatType is empty, set it to all")
		chatTypeRequested["llm_chat"] = struct{}{}
		chatTypeRequested["file_chat"] = struct{}{}
		chatTypeRequested["kb_chat"] = struct{}{}
	} else {
		slicedChatType := strings.Split(chatType, ",")
		for _, chatType := range slicedChatType {
			if chatType != "all" && chatType != "llm_chat" && chatType != "file_chat" && chatType != "kb_chat" {
				return nil, nil, http.StatusBadRequest, fmt.Errorf("chatType must be either all, llm_chat, file_chat, kb_chat")
			}
			chatTypeRequested[chatType] = struct{}{}
		}
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, canAccess, err := SouthAuthorizationControlWithUser(r, nil, "rag", privileges.PermissionRagManageOtherUserResource, userId, "conversation")
		if err != nil {
			return nil, nil, http.StatusInternalServerError, err
		}

		if !canAccess {
			return nil, nil, http.StatusForbidden, fmt.Errorf("user id is not match")
		}
	}

	body, headers, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations/users/%s", h.config.Rag.Endpoint, userId), http.MethodGet, "", nil, header, false, true, 3*time.Second)
	if err != nil {
		return nil, nil, http.StatusBadRequest, err
	}

	if status != http.StatusOK {
		return nil, nil, status, fmt.Errorf("failed to get conversation of user due to error: %s", string(body))
	}

	if chatType == "" || chatType == "all" {
		return body, headers, http.StatusOK, nil
	}
	var conversations []conversationV1.Conversation
	err = json.Unmarshal(body, &conversations)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}
	var result []conversationV1.Conversation
	for _, conversation := range conversations {
		if _, ok := chatTypeRequested[conversation.ChatType]; ok {
			result = append(result, conversation)
		}
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}
	return resultBytes, headers, http.StatusOK, nil
}

// get conversation of a user
func (h *RAGSouthApiHandler) GetConversationOfUser(w http.ResponseWriter, r *http.Request) {

	resultBytes, headers, status, err := h.getConversationOfUser(chi.URLParam(r, "userId"), r.URL.Query().Get("chatType"), r.Header, r)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation of user", http.StatusInternalServerError, "", err)
		return
	}

	ForwardResponseHeader(w, headers)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(resultBytes)))
	w.Write(resultBytes)
	w.WriteHeader(status)
}

func (h *RAGSouthApiHandler) GetConversationOfUserToModel(userId string) ([]conversationV1.Conversation, error) {
	body, _, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations/users/%s", h.config.Rag.Endpoint, userId), http.MethodGet, "", nil, nil, false, true, 3*time.Second)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		if status == http.StatusNotFound {
			err := fmt.Errorf("user with id: '%s' not found", userId)
			httpHelper.WriteCustomErrorAndLog(nil, fmt.Sprintf("user with id: '%s' not found", userId), http.StatusNotFound, "", err)
			return nil, err
		}
		err := fmt.Errorf("failed to get conversation of user with id: '%s'", userId)
		httpHelper.WriteCustomErrorAndLog(nil, fmt.Sprintf("failed to get conversation of user with id: '%s'", userId), http.StatusInternalServerError, "", err)
		return nil, err
	}

	conversations := []conversationV1.Conversation{}
	err = json.Unmarshal(body, &conversations)
	if err != nil {
		return nil, err
	}
	return conversations, nil
}

// delete conversation of a user
func (h *RAGSouthApiHandler) DeleteConversationOfUser(w http.ResponseWriter, r *http.Request) {

	// get all conversation of user
	conversations, err := h.GetConversationOfUserToModel(chi.URLParam(r, "userId"))
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation of user", http.StatusInternalServerError, "", err)
		return
	}

	if len(conversations) == 0 {
		w.Write([]byte("[]"))
		w.WriteHeader(http.StatusOK)
		return
	}

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "User id is required", http.StatusBadRequest, "", fmt.Errorf("user id is required"))
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, userId, "conversation")
		if err != nil {
			return
		}
	}

	body, headers, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations/users/%s", h.config.Rag.Endpoint, userId), r.Method, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete conversation of user", http.StatusInternalServerError, "", err)
		return
	}

	// delete temp chat file of file chat conversation
	// one shot delete all temp file this is painful
	// if failed then these file will be left forever cause we will not be able to get all conversation of user again
	// because it's been deleted
	for _, conversation := range conversations {
		if conversation.TempKBId != "" && conversation.ChatType == "file_chat" {
			err := os.RemoveAll(filepath.Join(h.config.Rag.FileChatPath, "data", "temp", conversation.TempKBId))
			if err != nil {
				coreApiLog.Logger.Warn("Failed to delete temp file, not much we can do here anymore", "error", err, "tempKBId", conversation.TempKBId, "conversationId", conversation.ID, "userId", conversation.UserID)
			}
		}
	}

	ForwardResponseHeader(w, headers)
	w.Write(body)
	w.WriteHeader(status)
}

func (h *RAGSouthApiHandler) PatchKnowledgeBase(w http.ResponseWriter, r *http.Request) {

	kbId := chi.URLParam(r, "knowledgeBaseId")
	if kbId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Knowledge base id is required", http.StatusBadRequest, "", fmt.Errorf("knowledge base id is required"))
		return
	}

	kbBody, _, status, err := h.getKnowledgeBaseDetailToModel(kbId)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge base detail", status, "", err)
		return
	}

	kb := &struct {
		knowledgeBaseV1.KnowledgeBaseCommonResult
		Data *knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}
	err = json.Unmarshal(kbBody, kb)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal knowledge base detail", http.StatusInternalServerError, "", err)
		return
	}

	if kb.Code != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge base detail", kb.Code, "", fmt.Errorf("failed to get knowledge base detail due to error: %s", string(kbBody)))
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, kb.Data.UserID, "knowledge_base")
		if err != nil {
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusInternalServerError, "", err)
		return
	}

	kbPost := &knowledgeBaseV1.KnowledgeBase{}
	err = json.Unmarshal(body, kbPost)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
		return
	}

	postBody := &struct {
		KbId      string `json:"kb_id"`
		KbInfo    string `json:"kb_info"`
		IsPrivate bool   `json:"is_private"`
	}{
		KbId:      kb.Data.KbId,
		KbInfo:    kbPost.KBInfo,
		IsPrivate: kbPost.IsPrivate,
	}

	bodyBytes, err := json.Marshal(postBody)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal request body", http.StatusInternalServerError, "", err)
		return
	}

	body, _, status, err = common.CommonRequest(fmt.Sprintf("%s/knowledge_base/update_info", h.config.Rag.Endpoint), http.MethodPost, "", bodyBytes, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to patch knowledge base", http.StatusInternalServerError, "", err)
		return
	}

	if status != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to patch knowledge base", status, "", fmt.Errorf("failed to patch knowledge base due to error: %s", string(body)))
		return
	}

	httpHelper.WriteResponseEntity(w, postBody)
}

func (h *RAGSouthApiHandler) DeleteKnowledgeBase(w http.ResponseWriter, r *http.Request) {

	kbId := chi.URLParam(r, "knowledgeBaseId")
	if kbId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Knowledge base id is required", http.StatusBadRequest, "", fmt.Errorf("knowledge base id is required"))
		return
	}

	body, _, status, err := h.getKnowledgeBaseDetailToModel(kbId)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge base detail", status, "", err)
		return
	}

	kb := &struct {
		knowledgeBaseV1.KnowledgeBaseCommonResult
		Data *knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}
	err = json.Unmarshal(body, kb)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal knowledge base detail", http.StatusInternalServerError, "", err)
		return
	}

	if kb.Code != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge base detail", kb.Code, "", fmt.Errorf("failed to get knowledge base detail due to error: %s", string(body)))
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, kb.Data.KbId, "knowledge_base")
		if err != nil {
			return
		}
	}

	body, _, status, err = common.CommonRequest(fmt.Sprintf("%s/knowledge_base/delete_knowledge_base/%s", h.config.Rag.Endpoint, kbId), r.Method, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete knowledge base", http.StatusInternalServerError, "", err)
		return
	}

	if status != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete knowledge base", status, "", fmt.Errorf("failed to delete knowledge base due to error: %s", string(body)))
		return
	}

	httpHelper.WriteResponseEntity(w, kb.Data)
	w.WriteHeader(status)
}

// create a knowledge base
func (h *RAGSouthApiHandler) CreateKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusInternalServerError, "", err)
		return
	}

	kb := &knowledgeBaseV1.KnowledgeBase{}
	err = json.Unmarshal(body, kb)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
		return
	}

	userProvider, err := initOrGetUserProvider(h.config)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get user provider", http.StatusInternalServerError, "", err)
		return
	}

	_, err = userProvider.GetUser(kb.UserID, nil)
	if err != nil {
		// check is not found
		if customErr.IsNotFound(err) {
			httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("User with id: %s not found", kb.UserID), http.StatusNotFound, "", err)
			return
		}
		httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Failed to get user by id: '%s'", kb.UserID), http.StatusInternalServerError, "", err)
		return
	}

	body, _, status, err := common.CommonRequest(fmt.Sprintf("%s/knowledge_base/create_knowledge_base", h.config.Rag.Endpoint), r.Method, "", body, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create knowledge base", http.StatusInternalServerError, "", err)
		return
	}

	if status != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create knowledge base", status, "", fmt.Errorf("failed to create knowledge base due to error: %s", string(body)))
		return
	}

	result := &struct {
		knowledgeBaseV1.KnowledgeBaseCommonResult
		Data string `json:"data"`
	}{}

	err = json.Unmarshal(body, result)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal knowledge base", http.StatusInternalServerError, "", err)
		return
	}

	if result.Code != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Failed to create knowledge base due to %s", result.Message), result.Code, "", fmt.Errorf("failed to create knowledge base due to error: %s", result.Message))
		return
	}

	returnKb := &knowledgeBaseV1.KnowledgeBase{
		KbId:              result.Data,
		KnowledgeBaseName: kb.KnowledgeBaseName,
		KBInfo:            kb.KBInfo,
		EmbedModel:        kb.EmbedModel,
		VectorStoreType:   kb.VectorStoreType,
	}

	httpHelper.WriteResponseEntity(w, returnKb)
}

// get knowledge base
func (h *RAGSouthApiHandler) GetKnowledgeBases(w http.ResponseWriter, r *http.Request) {

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "User id is required", http.StatusBadRequest, "", fmt.Errorf("user id is required"))
		return
	}

	userProvider, err := initOrGetUserProvider(h.config)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get user provider", http.StatusInternalServerError, "", err)
		return
	}

	user, err := userProvider.GetUser(userId, nil)
	if err != nil {
		// check is not found
		if customErr.IsNotFound(err) {
			httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("User with id: %s not found", userId), http.StatusNotFound, "", err)
			return
		}
		httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Failed to get user by id: '%s'", userId), http.StatusInternalServerError, "", err)
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, userId, "knowledge_base")
		if err != nil {
			return
		}
	}

	body, headers, status, err := common.CommonRequest(fmt.Sprintf("%s/knowledge_base/list_knowledge_bases/users/%s", h.config.Rag.Endpoint, userId), r.Method, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge bases", http.StatusInternalServerError, "", err)
		return
	}

	tmpWrapper := &struct {
		Data []knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}

	err = json.Unmarshal(body, tmpWrapper)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal knowledge bases", http.StatusInternalServerError, "", err)
		return
	}

	for index := range tmpWrapper.Data {
		tmpWrapper.Data[index].Username = user.Name
	}

	additional := r.URL.Query().Get("appendKB")
	if additional != "" {
		appendKbSet := strings.Split(additional, ",")
		for _, kbName := range appendKbSet {
			if kbName == "publicKB" {
				publicKbs, err := h.GetPublicKnowledgeBasesToModel(map[string]struct{}{userId: {}})
				if err != nil {
					httpHelper.WriteCustomErrorAndLog(w, "Failed to get public knowledge bases", http.StatusInternalServerError, "", err)
					return
				}
				tmpWrapper.Data = append(tmpWrapper.Data, publicKbs...)
				continue
			}

			if kbName == "groupedKB" {
				groupedKBs, ok := h.groupedKBCache.Load("groupedKBs")
				if !ok {
					httpHelper.WriteCustomErrorAndLog(w, "Failed to get grouped knowledge bases", http.StatusInternalServerError, "", fmt.Errorf("failed to get grouped knowledge bases"))
					return
				}

				convertedGroupedKBs, ok := groupedKBs.(map[string][]knowledgeBaseV1.KnowledgeBase)
				if !ok {
					httpHelper.WriteCustomErrorAndLog(w, "Failed to convert grouped knowledge bases", http.StatusInternalServerError, "", fmt.Errorf("failed to convert grouped knowledge bases"))
					return
				}

				if len(convertedGroupedKBs) > 0 {
					if _, ok := convertedGroupedKBs[userId]; ok {
						tmpWrapper.Data = append(tmpWrapper.Data, convertedGroupedKBs[userId]...)
					}
				}
			}
		}
	}

	kbList, err := json.Marshal(tmpWrapper.Data)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal knowledge bases", http.StatusInternalServerError, "", err)
		return
	}

	// reset content-length
	headers.Set("Content-Length", fmt.Sprintf("%d", len(kbList)))

	ForwardResponseHeader(w, headers)
	w.Write(kbList)
	w.WriteHeader(status)
}

// get public knowledge base to mode
func (h *RAGSouthApiHandler) GetPublicKnowledgeBasesToModel(excludesByUserId map[string]struct{}) ([]knowledgeBaseV1.KnowledgeBase, error) {
	body, _, status, err := common.CommonRequest(fmt.Sprintf("%s/knowledge_base/list_public_knowledge_base", h.config.Rag.Endpoint), http.MethodGet, "", nil, nil, false, true, 3*time.Second)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, fmt.Errorf("failed to get public knowledge bases")
	}

	tmpWrapper := &struct {
		Data []knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}

	err = json.Unmarshal(body, tmpWrapper)
	if err != nil {
		return nil, err
	}

	resultWrapper := &struct {
		Data []knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}

	userCache, ok := h.userCache.Load("users")
	if !ok {
		return nil, fmt.Errorf("failed to get user cache")
	}

	// note we use cache here, because public kb can be invoked by any user
	// which cause algorithm complexity to be O(n^2) if we don't use cache
	convertedUserCache, ok := userCache.(map[string]coreUserV1.CoreUser)
	if !ok {
		return nil, fmt.Errorf("failed to convert user cache")
	}

	for index := range tmpWrapper.Data {
		if _, ok := excludesByUserId[tmpWrapper.Data[index].UserID]; ok {
			continue
		}

		if user, ok := convertedUserCache[tmpWrapper.Data[index].UserID]; ok {
			tmpWrapper.Data[index].Username = user.Name
		} else {
			if tmpWrapper.Data[index].UserID == "build-in" {
				tmpWrapper.Data[index].Username = "build-in"
			} else {
				tmpWrapper.Data[index].Username = "un-referenced"
			}
		}
		resultWrapper.Data = append(resultWrapper.Data, tmpWrapper.Data[index])
	}

	return resultWrapper.Data, nil
}

// get public knowledge base
func (h *RAGSouthApiHandler) GetPublicKnowledgeBases(w http.ResponseWriter, r *http.Request) {
	body, headers, status, err := common.CommonRequest(fmt.Sprintf("%s/knowledge_base/list_public_knowledge_base", h.config.Rag.Endpoint), r.Method, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge bases", http.StatusInternalServerError, "", err)
		return
	}

	tmpWrapper := &struct {
		Data []knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}

	err = json.Unmarshal(body, tmpWrapper)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal knowledge bases", http.StatusInternalServerError, "", err)
		return
	}

	// init user provider
	uerProvider, err := initOrGetUserProvider(h.config)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get user provider", http.StatusInternalServerError, "", err)
		return
	}

	for index, kb := range tmpWrapper.Data {
		if kb.UserID == "build-in" {
			tmpWrapper.Data[index].Username = "build-in"
			continue
		}
		user, err := uerProvider.GetUser(kb.UserID, nil)
		if err != nil {
			coreApiLog.Logger.Warn("Failed to get user, will fall back to un-referenced", "error", err, "userId", kb.UserID)
			tmpWrapper.Data[index].Username = "un-referenced"
			continue
		}
		tmpWrapper.Data[index].Username = user.Name
	}

	kbList, err := json.Marshal(tmpWrapper.Data)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal knowledge bases", http.StatusInternalServerError, "", err)
		return
	}

	// reset content-length
	headers.Set("Content-Length", fmt.Sprintf("%d", len(kbList)))

	ForwardResponseHeader(w, headers)
	w.Write(kbList)
	w.WriteHeader(status)
}

func (h *RAGSouthApiHandler) CreateChat(w http.ResponseWriter, r *http.Request) {

	// parse body
	chatPost := &chatV1.ChatPost{}
	err := json.NewDecoder(r.Body).Decode(chatPost)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to decode chat post", http.StatusInternalServerError, "", err)
		return
	}

	if chatPost.Query == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Query is required", http.StatusBadRequest, "", nil)
		return
	}

	if len(chatPost.Messages) > 1000 {
		httpHelper.WriteCustomErrorAndLog(w, "Messages length must be less than or equal to 1000", http.StatusBadRequest, "", fmt.Errorf("messages length must be less than or equal to 1000"))
		return
	}

	if chatPost.ConversationId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "ConversationId is required", http.StatusBadRequest, "", nil)
		return
	}

	_, _, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations/%s", h.config.Rag.Endpoint, chatPost.ConversationId), http.MethodGet, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation message by id", http.StatusInternalServerError, "", err)
		return
	}

	if status == http.StatusNotFound {
		httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Conversation id: '%s' not found", chatPost.ConversationId), http.StatusNotFound, "", nil)
		return
	}

	if status != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation message by id", http.StatusInternalServerError, "", err)
		return
	}

	chatPost.Messages = append(chatPost.Messages, chatV1.Message{
		Role:    "user",
		Content: chatPost.Query,
	})

	if chatPost.HistoryLength == 0 {
		coreApiLog.Logger.Debug("HistoryLength is 0, fall back to 10")
		chatPost.HistoryLength = 10
	}

	// marshal chat post
	chatPostBytes, err := json.Marshal(chatPost)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal chat post", http.StatusInternalServerError, "", err)
		return
	}

	err = common.CommonStreamRequestRedirect(fmt.Sprintf("%s/chat/chat/completions", h.config.Rag.Endpoint), r.Method, http.StatusOK, bytes.NewReader(chatPostBytes), w)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create chat", http.StatusInternalServerError, "", err)
		return
	}
}

// get conversation messages
func (h *RAGSouthApiHandler) GetConversationMessages(w http.ResponseWriter, r *http.Request) {

	conversationId := chi.URLParam(r, "conversationId")
	if conversationId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "ConversationId is required", http.StatusBadRequest, "", nil)
		return
	}

	conversationFound, err := h.GetConversationByIdToModel(conversationId)
	if err != nil {
		if customErr.IsNotFound(err) {
			httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Conversation id: '%s' not found", conversationId), http.StatusNotFound, "", nil)

		} else {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation by id", http.StatusInternalServerError, "", err)
		}
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, conversationFound.UserID, "conversation")
		if err != nil {
			return
		}
	}

	body, headers, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations/messages/%s", h.config.Rag.Endpoint, chi.URLParam(r, "conversationId")), r.Method, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation messages", http.StatusInternalServerError, "", err)
		return
	}

	ForwardResponseHeader(w, headers)
	w.Write(body)
	w.WriteHeader(status)
}

// get message by id
func (h *RAGSouthApiHandler) GetConversationMessageById(w http.ResponseWriter, r *http.Request) {

	conversationId := chi.URLParam(r, "conversationId")
	if conversationId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "ConversationId is required", http.StatusBadRequest, "", nil)
		return
	}

	messageId := chi.URLParam(r, "messageId")
	if messageId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "MessageId is required", http.StatusBadRequest, "", nil)
		return
	}

	conversationFound, err := h.GetConversationByIdToModel(conversationId)
	if err != nil {
		if customErr.IsNotFound(err) {
			httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Conversation id: '%s' not found", conversationId), http.StatusNotFound, "", nil)

		} else {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation by id", http.StatusInternalServerError, "", err)
		}
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, conversationFound.UserID, "conversation")
		if err != nil {
			return
		}
	}

	body, headers, status, err := common.CommonRequest(fmt.Sprintf("%s/conversations/message/%s", h.config.Rag.Endpoint, chi.URLParam(r, "messageId")), r.Method, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation message by id", http.StatusInternalServerError, "", err)
		return
	}

	ForwardResponseHeader(w, headers)
	w.Write(body)
	w.WriteHeader(status)
}

func (h *RAGSouthApiHandler) GetChatQuickStarts(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(h.config.Rag.ChatQuickStarts)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal chat quick starts", http.StatusInternalServerError, "", err)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Write(data)
}

func (h *RAGSouthApiHandler) FileChat(w http.ResponseWriter, r *http.Request) {
	// parse body
	chatPost := &chatV1.FileChatPost{}
	err := json.NewDecoder(r.Body).Decode(chatPost)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to decode file chat post", http.StatusInternalServerError, "", err)
		return
	}

	if chatPost.Query == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Query is required", http.StatusBadRequest, "", fmt.Errorf("query is required"))
		return
	}

	if len(chatPost.Messages) > 1000 {
		httpHelper.WriteCustomErrorAndLog(w, "Messages length must be less than or equal to 1000", http.StatusBadRequest, "", fmt.Errorf("messages length must be less than or equal to 1000"))
		return
	}

	if chatPost.ConversationId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "ConversationId is required", http.StatusBadRequest, "", fmt.Errorf("conversation id is required"))
		return
	}

	if chatPost.TempKBId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "TempKBId is required", http.StatusBadRequest, "", fmt.Errorf("temp kb id is required"))
		return
	}

	conversation, err := h.GetConversationByIdToModel(chatPost.ConversationId)
	if err != nil {
		if customErr.IsNotFound(err) {
			httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Conversation id: '%s' not found", chatPost.ConversationId), http.StatusNotFound, "", err)
			return
		}
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation by id", http.StatusInternalServerError, "", err)
		return
	}

	// block user to access other user resource
	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, conversation.UserID, "conversation")
		if err != nil {
			return
		}
	}

	if conversation.ChatType != "file_chat" {
		httpHelper.WriteCustomErrorAndLog(w, "Conversation type must be file_chat", http.StatusBadRequest, "", fmt.Errorf("conversation type must be file_chat"))
		return
	}

	chatPost.Messages = append(chatPost.Messages, chatV1.Message{
		Role:    "user",
		Content: chatPost.Query,
	})

	if chatPost.HistoryLength == 0 {
		coreApiLog.Logger.Debug("HistoryLength is 0, fall back to 10")
		chatPost.HistoryLength = 10
	}

	// marshal chat post
	chatPostBytes, err := json.Marshal(chatPost)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal file chat post", http.StatusInternalServerError, "", err)
		return
	}

	err = common.CommonStreamRequestRedirect(fmt.Sprintf("%s/chat/file_chat", h.config.Rag.Endpoint), r.Method, http.StatusOK, bytes.NewReader(chatPostBytes), w)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create chat", http.StatusInternalServerError, "", err)
		return
	}
}

func (h *RAGSouthApiHandler) UploadFileKnowledgeBase(w http.ResponseWriter, r *http.Request) {

	kbId := chi.URLParam(r, "knowledgeBaseId")
	if kbId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Knowledge base name is required", http.StatusBadRequest, "", fmt.Errorf("knowledge base name is required"))
		return
	}

	bodyKB, _, status, err := h.getKnowledgeBaseDetailToModel(kbId)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge base detail", status, "", err)
		return
	}

	kbInfo := &struct {
		knowledgeBaseV1.KnowledgeBaseCommonResult
		Data knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}
	err = json.Unmarshal(bodyKB, kbInfo)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal knowledge base detail", http.StatusInternalServerError, "", err)
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, kbInfo.Data.UserID, "knowledge_base")
		if err != nil {
			return
		}
	}

	body, _, code, err := common.CommonRequestForwardBody(fmt.Sprintf("%s/knowledge_base/upload_docs", h.config.Rag.Endpoint), r.Method, "", r.Body, r.Header, false, true, 600*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to upload file knowledge base", http.StatusInternalServerError, "", err)
		return
	}

	if code == http.StatusInternalServerError {
		httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Got unexpected response code %d", code), code, "", fmt.Errorf("got unexpected response code %d with error: %s", code, string(body)))
		return
	}

	if code != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Got unexpected response code %d", code), code, "", fmt.Errorf("got unexpected response code %d", code))
		return
	}

	resultDetail := &httpHelper.FileUploadFailedResponse{}

	err = json.Unmarshal(body, resultDetail)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal response", http.StatusInternalServerError, "", err)
		return
	}

	if len(resultDetail.FailedFiles) > 0 {
		httpHelper.WriteCustomErrorAndLogHandler(w, "Failed to upload file knowledge base", http.StatusInternalServerError, "", nil, func(customError *httpHelper.CustomError) {
			customError.FileUploadFailedMessage = resultDetail
		})
		return
	}

	kbInfo.Data.FileCount += len(resultDetail.SucceededFiles)
	httpHelper.WriteResponseEntity(w, kbInfo.Data)
}

func (h *RAGSouthApiHandler) FileChatHandler(w http.ResponseWriter, r *http.Request) {
	// check userId is in path
	userId := chi.URLParam(r, "userId")
	if userId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Missing user id", http.StatusBadRequest, "", fmt.Errorf("missing user id in path"))
		return
	}

	// validate user exist
	userProvider, err := initOrGetUserProvider(h.config)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create user provider", http.StatusInternalServerError, "", err)
		return
	}

	_, err = userProvider.GetUser(userId, nil)
	if err != nil {
		if customErr.IsNotFound(err) {
			httpHelper.WriteCustomErrorAndLog(w, "User not found", http.StatusNotFound, "", err)
			return
		}
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusInternalServerError, "", err)
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, userId, "file_chat")
		if err != nil {
			return
		}
	}

	body, _, status, err := common.CommonRequestForwardBody(fmt.Sprintf("%s/knowledge_base/upload_temp_docs", h.config.Rag.Endpoint), r.Method, "", r.Body, r.Header, false, true, 10*time.Minute)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to upload file to remote server", status, "", err)
		return
	}

	if status != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Got unexpected status code %d expected %d", status, http.StatusOK), status, "", fmt.Errorf("unexpected status code with error: %s", string(body)))
		return
	}

	result := &struct {
		Data *httpHelper.FileUploadFailedResponse `json:"data"`
	}{}
	err = json.Unmarshal(body, result)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal response body", http.StatusInternalServerError, "", err)
		return
	}

	if result.Data.Id == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Got empty id from rag api", http.StatusInternalServerError, "", fmt.Errorf("got empty id from rag api"))
		return
	}

	if len(result.Data.FailedFiles) > 0 {

		failedMsg := ""

		// loop once , only one file allowed to upload for file chat
		for _, file := range result.Data.FailedFiles {
			failedMsg += fmt.Sprintf("%s: %s\n", file.FileName, file.Msg)
			break
		}

		httpHelper.WriteCustomErrorAndLogHandler(w, fmt.Sprintf("Failed to upload file to remote server due to '%s'", failedMsg), http.StatusInternalServerError, "", fmt.Errorf("failed to upload file to remote server due to '%s'", failedMsg), func(customError *httpHelper.CustomError) {
			customError.FileUploadFailedMessage = result.Data
		})
		return
	}

	if len(result.Data.SucceededFiles) == 0 {
		httpHelper.WriteCustomErrorAndLog(w, "Got empty succeeded files from rag api", http.StatusInternalServerError, "", fmt.Errorf("got empty succeeded files from rag api"))
		return
	}

	fileName := result.Data.SucceededFiles[0].FileName
	fileName = filepath.Base(fileName)

	// validate conversation exist

	conversation, err := h.CreateConversationNoForward(&conversationV1.Conversation{
		UserID:   userId,
		ChatType: "file_chat",
		Name:     fmt.Sprintf("%s-%s", fileName, time.Now().Format("2006-01-02 15:04:05")),
		TempKBId: result.Data.Id,
	}, r)

	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create conversation", http.StatusInternalServerError, "", err)
		return
	}

	conversation.TempFileName = fileName

	httpHelper.WriteResponseEntity(w, conversation)
}

func (h *RAGSouthApiHandler) QuickFileChatHandler(w http.ResponseWriter, r *http.Request) {
	// check post body
	bodyPost, err := io.ReadAll(r.Body)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusInternalServerError, "", err)
		return
	}

	quickFileChat := &conversationV1.Conversation{}
	err = json.Unmarshal(bodyPost, quickFileChat)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
		return
	}

	if quickFileChat.TempFileName == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Missing temp file name", http.StatusBadRequest, "", fmt.Errorf("missing temp file name"))
		return
	}

	if quickFileChat.UserID == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Missing user id", http.StatusBadRequest, "", fmt.Errorf("missing user id"))
		return
	}

	// validate user exist
	userProvider, err := initOrGetUserProvider(h.config)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create user provider", http.StatusInternalServerError, "", err)
		return
	}

	_, err = userProvider.GetUser(quickFileChat.UserID, nil)
	if err != nil {
		if customErr.IsNotFound(err) {
			httpHelper.WriteCustomErrorAndLog(w, "User not found", http.StatusNotFound, "", err)
			return
		}
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusInternalServerError, "", err)
		return
	}

	// create a multipart from local file /mnt/quick_file_chat
	bodyToPost := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyToPost)
	// add user id to multipart
	_ = writer.WriteField("user_id", quickFileChat.UserID)

	targetFile := filepath.Join(h.config.Rag.QuickFileChatPath, quickFileChat.TempFileName)
	part, err := writer.CreateFormFile("files", quickFileChat.TempFileName)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create form file", http.StatusInternalServerError, "", err)
		return
	}

	f, err := os.Open(targetFile)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to open file", http.StatusInternalServerError, "", err)
		return
	}
	defer f.Close()

	_, err = io.Copy(part, f)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to copy file", http.StatusInternalServerError, "", err)
		return
	}

	err = writer.Close()
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to close writer", http.StatusInternalServerError, "", err)
		return
	}

	respBody, _, code, err := common.CommonRequestForwardBody(fmt.Sprintf("%s/knowledge_base/upload_temp_docs", h.config.Rag.Endpoint), http.MethodPost, "", bodyToPost, map[string][]string{
		"Content-Type": {writer.FormDataContentType()},
	}, true, false, 600*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Failed to upload temp file of file chat %s", quickFileChat.TempFileName), http.StatusInternalServerError, "", err)
		return
	}

	if code != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Failed to upload temp file of file chat %d", code), code, "", fmt.Errorf("failed to upload temp file of file chat %s with error: %s", quickFileChat.TempFileName, string(respBody)))
		return
	}

	result := &struct {
		Data *httpHelper.FileUploadFailedResponse `json:"data"`
	}{}
	err = json.Unmarshal(respBody, result)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal response body", http.StatusInternalServerError, "", err)
		return
	}

	if result.Data.Id == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Got empty id from rag api", http.StatusInternalServerError, "", fmt.Errorf("got empty id from rag api"))
		return
	}

	if len(result.Data.FailedFiles) > 0 {

		failedMsg := ""

		// loop once , only one file allowed to upload for file chat
		for _, file := range result.Data.FailedFiles {
			failedMsg += fmt.Sprintf("%s: %s\n", file.FileName, file.Msg)
			break
		}

		httpHelper.WriteCustomErrorAndLogHandler(w, fmt.Sprintf("Failed to upload file to remote server due to '%s'", failedMsg), http.StatusInternalServerError, "", fmt.Errorf("failed to upload file to remote server due to '%s'", failedMsg), func(customError *httpHelper.CustomError) {
			customError.FileUploadFailedMessage = result.Data
		})
		return
	}

	if len(result.Data.SucceededFiles) == 0 {
		httpHelper.WriteCustomErrorAndLog(w, "Got empty succeeded files from rag api", http.StatusInternalServerError, "", fmt.Errorf("got empty succeeded files from rag api"))
		return
	}

	fileName := result.Data.SucceededFiles[0].FileName
	fileName = filepath.Base(fileName)

	// validate conversation exist

	conversation, err := h.CreateConversationNoForward(&conversationV1.Conversation{
		UserID:   quickFileChat.UserID,
		ChatType: "file_chat",
		Name:     fmt.Sprintf("%s-%s", fileName, time.Now().Format("2006-01-02 15:04:05")),
		TempKBId: result.Data.Id,
	}, r)

	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create conversation", http.StatusInternalServerError, "", err)
		return
	}

	conversation.TempFileName = fileName

	httpHelper.WriteResponseEntity(w, conversation)
}

func (h *RAGSouthApiHandler) getKnowledgeBaseDetailToModel(kbId string) ([]byte, map[string][]string, int, error) {

	body, header, status, err := common.CommonRequest(fmt.Sprintf("%s/knowledge_base/get_knowledge_base_detail/%s", h.config.Rag.Endpoint, kbId), http.MethodGet, "", nil, nil, false, true, 3*time.Second)
	if err != nil {
		return nil, nil, status, err
	}

	if status != http.StatusOK {
		if status == http.StatusNotFound {
			return nil, nil, status, fmt.Errorf("knowledge id: '%s' not found", kbId)
		}
		return nil, nil, status, fmt.Errorf("failed to get knowledge base detail")
	}
	return body, header, status, nil
}

func (h *RAGSouthApiHandler) GetKnowledgeBaseDetailToModel(kbId string, r *http.Request, w http.ResponseWriter) (*knowledgeBaseV1.KnowledgeBase, error) {

	body, _, _, err := h.getKnowledgeBaseDetailToModel(kbId)
	if err != nil {
		return nil, err
	}

	kbInfo := &struct {
		knowledgeBaseV1.KnowledgeBaseCommonResult
		Data *knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}
	err = json.Unmarshal(body, kbInfo)
	if err != nil {
		return nil, err
	}

	if !h.config.CoreApiConfig.DisableAuth && kbInfo.Data.IsPrivate {
		loginUser, canAccess, err := SouthAuthorizationControlWithUser(r, nil, "rag", privileges.PermissionRagManageOtherUserResource, kbInfo.Data.UserID, "knowledge_base")
		if err != nil {
			return nil, err
		}

		if !canAccess {
			userProvider, err := initOrGetUserProvider(h.config)
			if err != nil {
				return nil, err
			}
			user, err := userProvider.GetUser(kbInfo.Data.UserID, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to get user to check permission")
			}

			// check whether both user in a same group
			for _, group := range user.Groups {
				for _, loginUserGroup := range loginUser.Groups {
					if group == loginUserGroup {
						return kbInfo.Data, nil
					}
				}
			}
			return nil, fmt.Errorf("no permission to access knowledge base %s", kbId)
		}
	}

	return kbInfo.Data, nil
}

func (h *RAGSouthApiHandler) GetKnowledgeBaseDetail(w http.ResponseWriter, r *http.Request) {
	kbId := chi.URLParam(r, "knowledgeBaseId")
	if kbId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Knowledge base name is required", http.StatusBadRequest, "", fmt.Errorf("knowledge base name is required"))
		return
	}

	body, headers, _, err := h.getKnowledgeBaseDetailToModel(kbId)
	if err != nil {
		return
	}

	result := &struct {
		knowledgeBaseV1.KnowledgeBaseCommonResult
		Data *knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}

	err = json.Unmarshal(body, result)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal knowledge base detail", http.StatusInternalServerError, "", err)
		return
	}

	if result.Code != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge base detail", result.Code, result.Message, nil)
		return
	}

	if result.Data.UserID == "build-in" {
		result.Data.Username = "build-in"
	} else {
		userProvider, err := initOrGetUserProvider(h.config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create user provider", http.StatusInternalServerError, "", err)
			return
		}

		user, err := userProvider.GetUser(result.Data.UserID, nil)
		if err != nil {
			coreApiLog.Logger.Warn("Failed to get user, will fall back to un-referenced", "error", err, "userId", result.Data.UserID)
			result.Data.Username = "un-referenced"
		} else {
			result.Data.Username = user.Name
		}
	}

	delete(headers, "Content-Length")
	ForwardResponseHeader(w, headers)
	httpHelper.WriteResponseEntity(w, result.Data)
}

func (h *RAGSouthApiHandler) KBChatHandler(w http.ResponseWriter, r *http.Request) {

	// check userId is in path
	kbId := chi.URLParam(r, "knowledgeBaseId")
	if kbId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Missing knowledgeBaseId", http.StatusBadRequest, "", fmt.Errorf("missing knowledgeBaseId in path"))
		return
	}

	// parse body
	chatPost := &chatV1.KbChatPost{
		KBId: kbId,
	}
	err := json.NewDecoder(r.Body).Decode(chatPost)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to decode kb chat post", http.StatusInternalServerError, "", err)
		return
	}

	if chatPost.Query == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Query is required", http.StatusBadRequest, "", fmt.Errorf("query is required"))
		return
	}

	if len(chatPost.Messages) > 1000 {
		httpHelper.WriteCustomErrorAndLog(w, "Messages length must be less than or equal to 1000", http.StatusBadRequest, "", fmt.Errorf("messages length must be less than or equal to 1000"))
		return
	}

	if chatPost.ConversationId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "ConversationId is required", http.StatusBadRequest, "", fmt.Errorf("conversation id is required"))
		return
	}

	conversation, err := h.GetConversationByIdToModel(chatPost.ConversationId)
	if err != nil {
		if customErr.IsNotFound(err) {
			httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Conversation id: '%s' not found", chatPost.ConversationId), http.StatusNotFound, "", err)
			return
		}
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get conversation by id", http.StatusInternalServerError, "", err)
		return
	}

	if conversation.ChatType != "kb_chat" {
		httpHelper.WriteCustomErrorAndLog(w, "Conversation type must be kb_chat", http.StatusBadRequest, "", fmt.Errorf("conversation type must be kb_chat"))
		return
	}

	kbInfo, err := h.GetKnowledgeBaseDetailToModel(kbId, r, w)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge base detail", http.StatusInternalServerError, "", err)
		return
	}

	// if kb not belong to user request
	// if !h.config.CoreApiConfig.DisableAuth {
	// 	canAccess, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, kbInfo.UserID, "knowledge_base")
	// 	if err != nil {
	// 		return
	// 	}
	// 	if !canAccess {
	// 		httpHelper.WriteCustomErrorAndLog(w, "No permission to access knowledge base", http.StatusForbidden, "", fmt.Errorf("no permission to access knowledge base"))
	// 		return
	// 	}
	// }

	// todo: check is group or public

	if kbInfo.FileCount == 0 {
		httpHelper.WriteCustomErrorAndLog(w, "Knowledge base has no files", http.StatusBadRequest, "", fmt.Errorf("knowledge base %s has no files", kbId))
		return
	}

	chatPost.Messages = append(chatPost.Messages, chatV1.Message{
		Role:    "user",
		Content: chatPost.Query,
	})

	chatPost.KBName = kbId

	if chatPost.HistoryLength == 0 {
		coreApiLog.Logger.Debug("HistoryLength is 0, fall back to 10")
		chatPost.HistoryLength = 10
	}

	// marshal chat post
	chatPostBytes, err := json.Marshal(chatPost)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal chat post", http.StatusInternalServerError, "", err)
		return
	}

	err = common.CommonStreamRequestRedirect(fmt.Sprintf("%s/chat/kb_chat", h.config.Rag.Endpoint), r.Method, http.StatusOK, bytes.NewReader(chatPostBytes), w)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create chat", http.StatusInternalServerError, "", err)
		return
	}

}

func (h *RAGSouthApiHandler) GetKBFiles(w http.ResponseWriter, r *http.Request) {
	// check userId is in path
	kbId := chi.URLParam(r, "knowledgeBaseId")
	if kbId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Missing knowledgeBaseId", http.StatusBadRequest, "", fmt.Errorf("missing knowledgeBaseId in path"))
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		kbBody, _, status, err := h.getKnowledgeBaseDetailToModel(kbId)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge base detail", status, "", err)
			return
		}

		kbInfo := &knowledgeBaseV1.KnowledgeBase{}
		err = json.Unmarshal(kbBody, kbInfo)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal knowledge base detail", http.StatusInternalServerError, "", err)
			return
		}

		_, _, err = SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, kbInfo.UserID, "knowledge_base")
		if err != nil {
			return
		}
	}

	body, header, code, err := common.CommonRequest(fmt.Sprintf("%s/knowledge_base/list_files/%s", h.config.Rag.Endpoint, kbId), r.Method, "", nil, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get kb files", http.StatusInternalServerError, "", err)
		return
	}

	if code != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get kb files", code, "", fmt.Errorf("failed to get kb files due unexpected status code %d", code))
		return
	}

	result := &knowledgeBaseV1.KnowledgeBaseFileList{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal kb files", http.StatusInternalServerError, "", err)
		return
	}

	ForwardResponseHeader(w, header)
	w.WriteHeader(code)
	w.Write(body)
}

// delete kb files
func (h *RAGSouthApiHandler) DeleteKBFiles(w http.ResponseWriter, r *http.Request) {
	kbId := chi.URLParam(r, "knowledgeBaseId")
	if kbId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "Missing knowledgeBaseId", http.StatusBadRequest, "", fmt.Errorf("missing knowledgeBaseId in path"))
		return
	}

	kbBody, _, status, err := h.getKnowledgeBaseDetailToModel(kbId)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge base detail", status, "", err)
		return
	}

	kbInfo := &struct {
		knowledgeBaseV1.KnowledgeBaseCommonResult
		Data *knowledgeBaseV1.KnowledgeBase `json:"data"`
	}{}
	err = json.Unmarshal(kbBody, kbInfo)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal knowledge base detail", http.StatusInternalServerError, "", err)
		return
	}

	if kbInfo.Code != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get knowledge base detail", kbInfo.Code, kbInfo.Message, nil)
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {
		_, _, err := SouthAuthorizationControlWithUser(r, w, "rag", privileges.PermissionRagManageOtherUserResource, kbInfo.Data.UserID, "knowledge_base")
		if err != nil {
			return
		}
	}

	bodyFileList := []string{}
	// ready body
	err = json.NewDecoder(r.Body).Decode(&bodyFileList)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to decode body", http.StatusInternalServerError, "", err)
		return
	}

	deleteFiles := &struct {
		KbId              string   `json:"kb_id"`
		KnowledgeBaseName string   `json:"kb_name"`
		FileNames         []string `json:"file_names"`
		DeleteContent     bool     `json:"delete_content"`
		NotRefreshVSCache bool     `json:"not_refresh_vs_cache"`
	}{
		KbId:              kbId,
		KnowledgeBaseName: kbInfo.Data.KnowledgeBaseName,
		FileNames:         bodyFileList,
		DeleteContent:     true,
		NotRefreshVSCache: false,
	}

	bodyToPost, err := json.Marshal(deleteFiles)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal body", http.StatusInternalServerError, "", err)
		return
	}

	body, header, code, err := common.CommonRequest(fmt.Sprintf("%s/knowledge_base/delete_docs", h.config.Rag.Endpoint), r.Method, "", bodyToPost, r.Header, false, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get kb files", http.StatusInternalServerError, "", err)
		return
	}

	if code != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete kb files", code, "", fmt.Errorf("failed to delete kb files due unexpected status code %d with error: '%s'", code, string(body)))
		return
	}

	ForwardResponseHeader(w, header)
	w.WriteHeader(code)
	w.Write(body)
}
