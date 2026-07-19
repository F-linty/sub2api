package admin

import (
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
	if h.cfg != nil {
		enabled = h.cfg.Gateway.TokenSaver.Enabled
		minBytes = h.cfg.Gateway.TokenSaver.MinBytes
		maxBytes = h.cfg.Gateway.TokenSaver.MaxBytes
	}
	response.Success(c, h.recorder.Snapshot(enabled, minBytes, maxBytes))
}
