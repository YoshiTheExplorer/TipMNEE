package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/YoshiTheExplorer/TipMNEE/api/middleware"
	db "github.com/YoshiTheExplorer/TipMNEE/db/sqlc"

	"github.com/gin-gonic/gin"
)

type LedgerEventsHandler struct {
	store *db.Queries
}

func NewLedgerEventsHandler(store *db.Queries) *LedgerEventsHandler {
	return &LedgerEventsHandler{store: store}
}

func (h *LedgerEventsHandler) GetEarningsSummary(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	ctx := c.Request.Context()
	summary, err := h.store.GetEarningsSummaryForUser(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute earnings"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *LedgerEventsHandler) ListMyTips(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	limit := int32(50)
	offset := int32(0)

	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = int32(n)
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}

	ctx := c.Request.Context()
	events, err := h.store.ListTipsForUser(ctx, db.ListTipsForUserParams{
		UserID:  sql.NullInt64{Int64: userID, Valid: true},
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tips"})
		return
	}

	c.JSON(http.StatusOK, events)
}
