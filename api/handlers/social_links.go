package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	channelID := strings.TrimSpace(req.ChannelID)
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_id required"})
		return
	}

	// Takeover protection
	existing, err := h.store.GetSocialLinkByPlatformUser(ctx, db.GetSocialLinkByPlatformUserParams{
		Platform:       "youtube",
		PlatformUserID: channelID,
	})
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read existing link"})
		return
	}
	if err == nil && existing.UserID != userID && existing.VerifiedAt.Valid {
        c.JSON(http.StatusConflict, gin.H{"error": "channel already verified by another user"})
        return
    }

	// Already linked to this user
	if err == nil && existing.UserID == userID {
		c.JSON(http.StatusOK, gin.H{"linked": true, "verified": existing.VerifiedAt.Valid})
		return
	}

	// Create link as UNVERIFIED
	_, err = h.store.CreateSocialLink(ctx, db.CreateSocialLinkParams{
		UserID:         userID,
		Platform:       "youtube",
		PlatformUserID: channelID,
		VerifiedAt:     sql.NullTime{Valid: false},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to link channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"linked": true, "verified": false})
}

type verifyYouTubeReq struct {
	ChannelID   string `json:"channel_id" binding:"required"`
	AccessToken string `json:"access_token" binding:"required"`
}

func (h *SocialLinksHandler) VerifyYouTubeChannel(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req verifyYouTubeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channelID := strings.TrimSpace(req.ChannelID)
	accessToken := strings.TrimSpace(req.AccessToken)
	
	ctx := c.Request.Context()

	// 1. Check DB for existing link (stores result in 'err')
	existing, err := h.store.GetSocialLinkByPlatformUser(ctx, db.GetSocialLinkByPlatformUserParams{
		Platform:       "youtube",
		PlatformUserID: channelID,
	})
    
    // SAFETY: Catch real DB errors (not just 'no rows')
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read existing link from DB"})
		return
	}

    // 2. Takeover protection
	if err == nil && existing.UserID != userID && existing.VerifiedAt.Valid {
		c.JSON(http.StatusConflict, gin.H{"error": "channel already verified by another user"})
		return
	}

	// 3. OAuth ownership check (uses 'verifyErr' so we don't overwrite 'err')
	ok, verifyErr := verifyYouTubeOwnership(ctx, accessToken, channelID)
	if verifyErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": verifyErr.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "oauth token does not own this channel_id"})
		return
	}

	now := time.Now()

    // 4. Use the ORIGINAL 'err' from step 1 to decide what to do
    if err == sql.ErrNoRows {
        // NO existing link: finally CreateSocialLink will be called!
        if _, err := h.store.CreateSocialLink(ctx, db.CreateSocialLinkParams{
            UserID:         userID,
            Platform:       "youtube",
            PlatformUserID: channelID,
            VerifiedAt:     sql.NullTime{Time: now, Valid: true},
        }); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "failed to create social link"})
            return
        }
    } else {
        // Existing link found (err was nil)
        if existing.UserID == userID {
            if !existing.VerifiedAt.Valid {
                if _, err := h.store.UpdateSocialLinkVerifiedAt(ctx, db.UpdateSocialLinkVerifiedAtParams{
                    ID:         existing.ID,
                    VerifiedAt: sql.NullTime{Time: now, Valid: true},
                }); err != nil {
                    c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark verified"})
                    return
                }
            }
        } else {
            // Takeover + verify
            if _, err := h.store.TransferSocialLinkToUser(ctx, db.TransferSocialLinkToUserParams{
                ID:         existing.ID,
                UserID:     userID,
                VerifiedAt: sql.NullTime{Time: now, Valid: true},
            }); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to takeover social link"})
                return
            }
        }
    }

	// Backfill ledger
	_ = h.store.BackfillLedgerEventsUserIDForChannel(ctx, db.BackfillLedgerEventsUserIDForChannelParams{
		UserID:         sql.NullInt64{Int64: userID, Valid: true},
		Platform:       "youtube",
		PlatformUserID: channelID,
	})

	c.JSON(http.StatusOK, gin.H{"verified": true})
}

// Calls YouTube Data API: GET /youtube/v3/channels?part=id&mine=true
func verifyYouTubeOwnership(ctx context.Context, accessToken, channelID string) (bool, error) {
	url := "https://www.googleapis.com/youtube/v3/channels?part=id&mine=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("youtube api request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 400 {
			msg = msg[:400] + "..."
		}
		if msg == "" {
			msg = resp.Status
		}
		return false, fmt.Errorf("youtube api error: %s", msg)
	}

	type ytResp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	var out ytResp
	if err := json.Unmarshal(body, &out); err != nil {
		return false, fmt.Errorf("failed to parse youtube response: %w", err)
	}
	if len(out.Items) == 0 {
		return false, errors.New("no channels returned from youtube; token may lack youtube scopes")
	}

	for _, it := range out.Items {
		if strings.TrimSpace(it.ID) == channelID {
			return true, nil
		}
	}
	return false, nil
}
