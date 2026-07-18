package dto

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAnnouncementFromServiceFormatsIDAsString(t *testing.T) {
	now := time.Now()

	out := AnnouncementFromService(&service.Announcement{
		ID:         1191948579941384200,
		Title:      "notice",
		Content:    "content",
		Status:     service.AnnouncementStatusActive,
		NotifyMode: service.AnnouncementNotifyModePopup,
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	require.NotNil(t, out)
	require.Equal(t, "1191948579941384200", out.ID)
}

func TestUserAnnouncementFromServiceFormatsIDAsString(t *testing.T) {
	now := time.Now()

	out := UserAnnouncementFromService(&service.UserAnnouncement{
		Announcement: service.Announcement{
			ID:         1191948579941384200,
			Title:      "notice",
			Content:    "content",
			NotifyMode: service.AnnouncementNotifyModePopup,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	})

	require.NotNil(t, out)
	require.Equal(t, "1191948579941384200", out.ID)
}
