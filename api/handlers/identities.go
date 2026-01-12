package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/auth/credentials/idtoken"
	db "github.com/YoshiTheExplorer/TipMNEE/db/sqlc"
	util "github.com/YoshiTheExplorer/TipMNEE/util"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)


type IdentitiesHandler struct {
	store     *db.Queries
	jwtSecret string
	googleAudiences []string
}

func NewIdentitiesHandler(store *db.Queries, jwtSecret string, googleAudiences []string) *IdentitiesHandler {
	raw := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_IDS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	}
	var auds []string
	for _, a := range strings.Split(raw, ",") {
		a = strings.TrimSpace(a)
		if a != "" {
			auds = append(auds, a)
		}
	}
	return &IdentitiesHandler{
		store:           store,
		jwtSecret:       jwtSecret,
		googleAudiences: auds,
	}
}

type loginResp struct {
	AccessToken string `json:"access_token"`
	UserID      int64  `json:"user_id"`
}

func (h *IdentitiesHandler) mintJWT(userID int64) (string, error) {
	claims := struct {
		UserID int64 `json:"user_id"`
		jwt.RegisteredClaims
	}{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.jwtSecret))
}

/*
Wallet login MVP:
- You SHOULD verify signature. For hackathon, you can start with "trust address" (not secure).
- Better: implement EIP-191 / personal_sign verification using go-ethereum crypto.
*/
type walletMessageReq struct {
	Address string `json:"address" binding:"required"`
}

type walletMessageResp struct {
	Address string `json:"address"`
	Message string `json:"message"`
}

func generateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func walletLoginMessage(addr, nonce string, expires time.Time) string {
	return fmt.Sprintf(
		"TipMNEE wants you to sign in with your Ethereum account.\n\nAddress: %s\nNonce: %s\nExpires: %s",
		addr,
		nonce,
		expires.UTC().Format(time.RFC3339),
	)
}

func (h *IdentitiesHandler) GetWalletLoginMessage(c *gin.Context) {
	var req walletMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	addr := strings.ToLower(strings.TrimSpace(req.Address))
	if addr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}

	nonce, err := generateNonce()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate nonce"})
		return
	}

	expires := time.Now().UTC().Add(10 * time.Minute).Truncate(time.Second)

	message := walletLoginMessage(addr, nonce, expires)

	ctx := c.Request.Context()
	if err := h.store.UpsertLoginNonce(ctx, db.UpsertLoginNonceParams{
		Address:   addr,
		Nonce:     nonce,
		ExpiresAt: expires,
		Message:   message,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store nonce"})
		return
	}

	c.JSON(http.StatusOK, walletMessageResp{Address: addr, Message: message})
}

type walletLoginReq struct {
  Address   string `json:"address" binding:"required"`
  Signature string `json:"signature" binding:"required"`
}

func (h *IdentitiesHandler) LoginWithWallet(c *gin.Context) {
	var req walletLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	addr := strings.ToLower(strings.TrimSpace(req.Address))
	if addr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}

	ctx := c.Request.Context()

	// 1) Must have a nonce issued for this address (forces /auth/wallet/message first)
	ln, err := h.store.GetLoginNonce(ctx, addr)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing nonce: call /api/auth/wallet/message first"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read nonce"})
		return
	}

	// 2) Check expiry (UTC + seconds)
	now := time.Now().UTC().Truncate(time.Second)
	if now.After(ln.ExpiresAt.UTC().Truncate(time.Second)) {
		_ = h.store.DeleteLoginNonce(ctx, addr)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "nonce expired: call /api/auth/wallet/message again"})
		return
	}

	// 3) Verify signature against the exact stored message
	recovered, err := util.RecoverAddressFromPersonalSign(ln.Message, req.Signature)
	if err != nil {
		_ = h.store.DeleteLoginNonce(ctx, addr) // optional but good
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	expected := strings.ToLower(strings.TrimSpace(ln.Address))
	if strings.ToLower(strings.TrimSpace(recovered)) != expected {
		_ = h.store.DeleteLoginNonce(ctx, addr) // optional but good
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":     "signature does not match address",
			"recovered": recovered,
			"expected":  expected,
			"canonical": ln.Message, // debug only; remove later
		})
		return
	}

	// 4) One-time use nonce (prevents replay)
	_ = h.store.DeleteLoginNonce(ctx, addr)

	// ---- existing logic unchanged below ----

	ident, err := h.store.GetIdentity(ctx, db.GetIdentityParams{
		Provider:       "wallet",
		ProviderUserID: addr,
	})

	var userID int64
	if err == nil {
		userID = ident.UserID
	} else {
		u, err2 := h.store.CreateUser(ctx)
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		_, err2 = h.store.CreateIdentity(ctx, db.CreateIdentityParams{
			UserID:         u.ID,
			Provider:       "wallet",
			ProviderUserID: addr,
		})
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create identity"})
			return
		}

		userID = u.ID
	}

	token, err := h.mintJWT(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mint token"})
		return
	}

	c.JSON(http.StatusOK, loginResp{AccessToken: token, UserID: userID})
}

type googleLoginReq struct {
	IDToken string `json:"id_token" binding:"required"`
}

func (h *IdentitiesHandler) validateGoogleIDToken(c *gin.Context, raw string) (*idtoken.Payload, error) {
	// idtoken.Validate requires an "audience" string. If you have multiple client IDs
	// (web + extension), we try each.
	if len(h.googleAudiences) == 0 {
		return nil, fmt.Errorf("missing GOOGLE_CLIENT_ID(S)")
	}

	ctx := c.Request.Context()
	var lastErr error
	for _, aud := range h.googleAudiences {
		p, err := idtoken.Validate(ctx, raw, aud)
		if err == nil {
			return p, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

/*
Google login:
- Verify ID token, extract "sub".
- Then same logic as wallet:
  identities(provider='google', provider_user_id=sub)
*/
func (h *IdentitiesHandler) LoginWithGoogle(c *gin.Context) {
	var req googleLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payload, err := h.validateGoogleIDToken(c, strings.TrimSpace(req.IDToken))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid google id_token"})
		return
	}

	sub := strings.TrimSpace(payload.Subject)
	if sub == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "google token missing subject"})
		return
	}

	ctx := c.Request.Context()

	ident, err := h.store.GetIdentity(ctx, db.GetIdentityParams{
		Provider:       "google",
		ProviderUserID: sub,
	})

	var userID int64
	if err == nil {
		userID = ident.UserID
	} else {
		u, err2 := h.store.CreateUser(ctx)
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		_, err2 = h.store.CreateIdentity(ctx, db.CreateIdentityParams{
			UserID:         u.ID,
			Provider:       "google",
			ProviderUserID: sub,
		})
		if err2 != nil {
			// Handle race: if identity was created concurrently, fetch it again.
			ident2, err3 := h.store.GetIdentity(ctx, db.GetIdentityParams{
				Provider:       "google",
				ProviderUserID: sub,
			})
			if err3 != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create identity"})
				return
			}
			userID = ident2.UserID
		} else {
			userID = u.ID
		}
	}

	token, err := h.mintJWT(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mint token"})
		return
	}

	c.JSON(http.StatusOK, loginResp{AccessToken: token, UserID: userID})
}
