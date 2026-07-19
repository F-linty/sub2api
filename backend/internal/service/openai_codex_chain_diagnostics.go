package service

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func recordCodexChainDiagnostics(c *gin.Context, account *Account, model string, body []byte, compact bool, transport OpenAIUpstreamTransport, requestHadPreviousResponseID bool) {
	if account == nil {
		return
	}
	promptCacheKey := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	apiKeyID := getAPIKeyIDFromContext(c)
	headerSessionID := ""
	headerConversationID := ""
	if c != nil {
		headerSessionID = strings.TrimSpace(c.GetHeader("session_id"))
		headerConversationID = strings.TrimSpace(c.GetHeader("conversation_id"))
	}
	upstreamSessionID := ""
	upstreamConversationID := ""
	if promptCacheKey != "" {
		isolated := isolateOpenAISessionID(apiKeyID, promptCacheKey)
		upstreamSessionID = isolated
		upstreamConversationID = isolated
	} else if compact {
		upstreamSessionID = isolateOpenAISessionID(apiKeyID, resolveOpenAICompactSessionID(c))
	}

	DefaultCodexChainRecorder().Record(CodexChainEvent{
		AccountID:                    account.ID,
		AccountType:                  string(account.Type),
		Model:                        model,
		Transport:                    string(transport),
		Compact:                      compact,
		StoreFalse:                   gjson.GetBytes(body, "store").Exists() && !gjson.GetBytes(body, "store").Bool(),
		StreamTrue:                   gjson.GetBytes(body, "stream").Bool(),
		HasPreviousResponseID:        gjson.GetBytes(body, "previous_response_id").Exists(),
		PreviousResponseIDRemoved:    requestHadPreviousResponseID && !gjson.GetBytes(body, "previous_response_id").Exists(),
		HasPromptCacheKey:            promptCacheKey != "",
		PromptCacheKeyHash:           hashSensitiveValueForLog(promptCacheKey),
		HasReasoning:                 gjson.GetBytes(body, "reasoning").Exists(),
		ReasoningEffort:              strings.TrimSpace(gjson.GetBytes(body, "reasoning.effort").String()),
		HasReasoningEncryptedContent: codexBodyIncludesReasoningEncryptedContent(body),
		HeaderSessionIDHash:          hashSensitiveValueForLog(headerSessionID),
		HeaderConversationIDHash:     hashSensitiveValueForLog(headerConversationID),
		UpstreamSessionIDHash:        hashSensitiveValueForLog(upstreamSessionID),
		UpstreamConversationIDHash:   hashSensitiveValueForLog(upstreamConversationID),
	})
}

func codexBodyIncludesReasoningEncryptedContent(body []byte) bool {
	include := gjson.GetBytes(body, "include")
	if !include.Exists() {
		return false
	}
	if include.IsArray() {
		for _, value := range include.Array() {
			if value.String() == "reasoning.encrypted_content" {
				return true
			}
		}
		return false
	}
	return include.String() == "reasoning.encrypted_content"
}
