package service

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const codexAutoPromptCacheKeyPrefix = "codex_cache_"

type codexPromptCacheResolution struct {
	Key    string
	Source string
}

func resolveCodexOAuthPromptCacheKey(c *gin.Context, account *Account, reqBody map[string]any, model string) codexPromptCacheResolution {
	if existing, ok := reqBody["prompt_cache_key"].(string); ok {
		if key := strings.TrimSpace(existing); key != "" {
			return codexPromptCacheResolution{Key: key, Source: "body"}
		}
	}
	if c != nil {
		if key := strings.TrimSpace(c.GetHeader("session_id")); key != "" {
			return codexPromptCacheResolution{Key: deriveCodexPromptCacheKeyFromSeed(c, account, model, "header_session_id", key), Source: "header_session_id"}
		}
		if key := strings.TrimSpace(c.GetHeader("conversation_id")); key != "" {
			return codexPromptCacheResolution{Key: deriveCodexPromptCacheKeyFromSeed(c, account, model, "header_conversation_id", key), Source: "header_conversation_id"}
		}
	}
	if key := deriveCodexPromptCacheKeyFromBody(c, account, reqBody, model); key != "" {
		return codexPromptCacheResolution{Key: key, Source: "derived_body"}
	}
	return codexPromptCacheResolution{}
}

func deriveCodexPromptCacheKeyFromBody(c *gin.Context, account *Account, reqBody map[string]any, model string) string {
	if len(reqBody) == 0 {
		return ""
	}
	seedParts := []string{
		"model=" + normalizeCodexModel(strings.TrimSpace(model)),
		"instructions=" + stableSeedValue(reqBody["instructions"]),
		"tools=" + stableSeedValue(reqBody["tools"]),
		"tool_choice=" + stableSeedValue(reqBody["tool_choice"]),
		"reasoning=" + stableSeedValue(reqBody["reasoning"]),
	}
	if firstUser := firstCodexInputUserSeed(reqBody["input"]); firstUser != "" {
		seedParts = append(seedParts, "first_user="+firstUser)
	}
	if len(seedParts) == 0 {
		return ""
	}
	return deriveCodexPromptCacheKeyFromSeed(c, account, model, "body", strings.Join(seedParts, "|"))
}

func deriveCodexPromptCacheKeyFromSeed(c *gin.Context, account *Account, model, source, seed string) string {
	trimmed := strings.TrimSpace(seed)
	if trimmed == "" {
		return ""
	}
	accountID := int64(0)
	if account != nil {
		accountID = account.ID
	}
	apiKeyID := int64(0)
	if c != nil {
		apiKeyID = getAPIKeyIDFromContext(c)
	}
	raw := strings.Join([]string{
		"account=" + strconv.FormatInt(accountID, 10),
		"api_key=" + strconv.FormatInt(apiKeyID, 10),
		"model=" + normalizeCodexModel(strings.TrimSpace(model)),
		"source=" + source,
		"seed=" + trimmed,
	}, "\x00")
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%s%x", codexAutoPromptCacheKeyPrefix, sum[:16])
}

func stableSeedValue(value any) string {
	if value == nil {
		return ""
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(raw)
}

func firstCodexInputUserSeed(raw any) string {
	items, ok := raw.([]any)
	if !ok {
		return stableSeedValue(raw)
	}
	for _, itemRaw := range items {
		item, ok := itemRaw.(map[string]any)
		if !ok {
			continue
		}
		role, _ := item["role"].(string)
		if strings.TrimSpace(role) != "user" {
			continue
		}
		return stableSeedValue(item["content"])
	}
	if len(items) > 0 {
		return stableSeedValue(items[0])
	}
	return ""
}
