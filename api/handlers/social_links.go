package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/YoshiTheExplorer/TipMNEE/api/middleware"
	db "github.com/YoshiTheExplorer/TipMNEE/db/sqlc"

	"github.com/gin-gonic/gin"
)

type SocialLinksHandler struct {
	store *db.Queries
}

func NewSocialLinksHandler(store *db.Queries) *SocialLinksHandler {
	return &SocialLinksHandler{store: store}
}

type linkYouTubeReq struct {
	ChannelID string `json:"channel_id" binding:"required"`
	// handle optional if you want later
}

func (h *SocialLinksHandler) LinkYouTubeChannel(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req linkYouTubeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Safety: prevent takeover if channel already linked to someone else
	existing, err := h.store.GetSocialLinkByPlatformUser(ctx, db.GetSocialLinkByPlatformUserParams{
		Platform:       "youtube",
		PlatformUserID: req.ChannelID,
	})
	if err == nil && existing.UserID != userID {
		c.JSON(http.StatusConflict, gin.H{"error": "channel already linked to another user"})
		return
	}

	// Create link (if already linked to same user, you can no-op or update verified_at)
	_, err = h.store.CreateSocialLink(ctx, db.CreateSocialLinkParams{
		UserID:         userID,
		Platform:       "youtube",
		PlatformUserID: req.ChannelID,
		VerifiedAt:     sql.NullTime{Time: time.Now(), Valid: true},
	})
	if err != nil {
		// If duplicate because it's already linked to same user, you can ignore or return ok
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to link channel"})
		return
	}

	// Backfill old unclaimed events for this channel
	_ = h.store.BackfillLedgerEventsUserIDForChannel(ctx, db.BackfillLedgerEventsUserIDForChannelParams{
		UserID: sql.NullInt64{Int64: userID, Valid: true},
		Platform: "youtube",
		PlatformUserID: req.ChannelID,
	})

	c.JSON(http.StatusOK, gin.H{"linked": true})
}
