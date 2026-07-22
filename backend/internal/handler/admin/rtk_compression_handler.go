package admin

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tokensaver"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type RTKCompressionHandler struct {
	recorder *service.RTKCompressionRecorder
	cfg      *config.Config
}

func NewRTKCompressionHandler(cfg *config.Config) *RTKCompressionHandler {
	return &RTKCompressionHandler{recorder: service.DefaultRTKCompressionRecorder(), cfg: cfg}
}

func (h *RTKCompressionHandler) GetSnapshot(c *gin.Context) {
	if h == nil || h.recorder == nil {
		response.Success(c, service.RTKCompressionSnapshot{})
		return
	}
	enabled, strategy, minBytes, maxBytes := false, "", 0, 0
	outputStyle := service.RTKOutputStyleConfig{}
	if h.cfg != nil {
		enabled = h.cfg.Gateway.TokenSaver.Enabled
		opts := serviceOpenAITokenSaverOptionsForAdmin(h.cfg.Gateway.TokenSaver)
		strategy = opts.Strategy
		minBytes = opts.MinBytes
		maxBytes = opts.MaxBytes
		outputStyle.Enabled = h.cfg.Gateway.TokenSaver.OutputStyleEnabled
		outputStyle.Level = strings.TrimSpace(h.cfg.Gateway.TokenSaver.OutputStyleLevel)
	}
	response.Success(c, h.recorder.Snapshot(enabled, strategy, minBytes, maxBytes, outputStyle))
}

type updateRTKCompressionConfigRequest struct {
	Strategy    *string                       `json:"strategy"`
	OutputStyle *service.RTKOutputStyleConfig `json:"output_style"`
}

func (h *RTKCompressionHandler) UpdateConfig(c *gin.Context) {
	if h == nil || h.cfg == nil {
		response.Error(c, http.StatusServiceUnavailable, "RTK compression config is unavailable")
		return
	}
	var req updateRTKCompressionConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	if req.Strategy != nil {
		strategy := strings.ToLower(strings.TrimSpace(*req.Strategy))
		switch strategy {
		case "", tokensaver.Strategy9Router:
			h.cfg.Gateway.TokenSaver.Strategy = tokensaver.Strategy9Router
		case tokensaver.StrategyConservative:
			h.cfg.Gateway.TokenSaver.Strategy = tokensaver.StrategyConservative
		default:
			response.BadRequest(c, "strategy must be 9router or conservative")
			return
		}
	}
	if req.OutputStyle != nil {
		level := strings.ToLower(strings.TrimSpace(req.OutputStyle.Level))
		if level == "" {
			level = "concise"
		}
		switch level {
		case "concise", "terse":
		default:
			response.BadRequest(c, "output_style.level must be concise or terse")
			return
		}
		h.cfg.Gateway.TokenSaver.OutputStyleEnabled = req.OutputStyle.Enabled
		h.cfg.Gateway.TokenSaver.OutputStyleLevel = level
	}
	h.GetSnapshot(c)
}

func serviceOpenAITokenSaverOptionsForAdmin(cfg config.GatewayTokenSaverConfig) tokensaver.Options {
	strategy := strings.ToLower(strings.TrimSpace(cfg.Strategy))
	if strategy == "" {
		strategy = tokensaver.Strategy9Router
	}
	minBytes := cfg.MinBytes
	maxBytes := cfg.MaxBytes
	if strategy == tokensaver.Strategy9Router {
		minBytes = 500
		maxBytes = 10 << 20
	}
	return tokensaver.Options{Strategy: strategy, MinBytes: minBytes, MaxBytes: maxBytes}
}
