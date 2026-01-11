package api

import (
	"net/http"
	"os"

	db "github.com/YoshiTheExplorer/TipMNEE/db/sqlc"

	"github.com/gin-gonic/gin"

	"github.com/YoshiTheExplorer/TipMNEE/api/handlers"
	"github.com/YoshiTheExplorer/TipMNEE/api/middleware"
)

type Server struct {
	store     *db.Queries
	router    *gin.Engine
	jwtSecret string
}

func NewServer(store *db.Queries) *Server {
	s := &Server{
		store:     store,
		router:    gin.New(),
		jwtSecret: os.Getenv("JWT_SECRET"),
	}

	// Global middleware
	s.router.Use(gin.Logger(), gin.Recovery())

	// Health
	s.router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	// Instantiate handlers
	usersH := handlers.NewUsersHandler(store)
	identitiesH := handlers.NewIdentitiesHandler(store, s.jwtSecret)
	socialH := handlers.NewSocialLinksHandler(store)
	payoutsH := handlers.NewPayoutsHandler(store)
	ledgerH := handlers.NewLedgerEventsHandler(store)

	// Public routes
	public := s.router.Group("/api")
	{
		// Resolve (public) - used by extension
		public.GET("/resolve/youtube/:channelId", payoutsH.ResolveYouTubeChannelPayout)
	}

	// Auth routes
	auth := s.router.Group("/api/auth")
	{
		auth.POST("/wallet/message", identitiesH.GetWalletLoginMessage)
		auth.POST("/wallet", identitiesH.LoginWithWallet)
		auth.POST("/google", identitiesH.LoginWithGoogle)
	}

	// Protected routes
	protected := s.router.Group("/api")
	protected.Use(middleware.AuthMiddleware(s.jwtSecret))
	{
		// User
		protected.GET("/me", usersH.GetMe)

		// Link socials
		protected.POST("/social/youtube/link", socialH.LinkYouTubeChannel)

		// Payouts
		protected.POST("/payouts", payoutsH.UpsertPayout) // set payout address

		// Earnings
		protected.GET("/me/earnings", ledgerH.GetEarningsSummary)
		protected.GET("/me/tips", ledgerH.ListMyTips)
	}

	return s
}

func (s *Server) Start(addr string) error {
	return s.router.Run(addr)
}
