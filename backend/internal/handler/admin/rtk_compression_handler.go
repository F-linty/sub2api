package admin

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
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
	enabled, minBytes, maxBytes := false, 0, 0
	outputStyle := service.RTKOutputStyleConfig{}
	if h.cfg != nil {
		enabled = h.cfg.Gateway.TokenSaver.Enabled
		minBytes = h.cfg.Gateway.TokenSaver.MinBytes
		maxBytes = h.cfg.Gateway.TokenSaver.MaxBytes
		outputStyle.Enabled = h.cfg.Gateway.TokenSaver.OutputStyleEnabled
		outputStyle.Level = strings.TrimSpace(h.cfg.Gateway.TokenSaver.OutputStyleLevel)
	}
	response.Success(c, h.recorder.Snapshot(enabled, minBytes, maxBytes, outputStyle))
}

type updateRTKCompressionConfigRequest struct {
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
