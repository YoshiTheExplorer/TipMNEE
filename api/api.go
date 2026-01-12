package api

import (
	"log"
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
	//googleAudiences []string
}

// func parseCSVEnv(key string) []string {
// 	raw := strings.TrimSpace(os.Getenv(key))
// 	if raw == "" { return nil }
// 	parts := strings.Split(raw, ",")
// 	out := make([]string, 0, len(parts))
// 	for _, p := range parts {
// 		p = strings.TrimSpace(p)
// 		if p != "" { out = append(out, p) }
// 	}
// 	return out
//  }

func NewServer(store *db.Queries) *Server {
	s := &Server{
		store:     store,
		router:    gin.New(),
		jwtSecret: os.Getenv("JWT_SECRET"),
		// googleAudiences: func() []string {
		// 	if auds := parseCSVEnv("GOOGLE_CLIENT_IDS"); len(auds) > 0 {
		// 		return auds
		// 	}
		// 	return parseCSVEnv("GOOGLE_CLIENT_ID")
		// }(),
	}

	// Global middleware
	s.router.Use(gin.Logger(), gin.Recovery())

	s.router.Use(middleware.CORS())

	// Health
	s.router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	// Instantiate handlers
	usersH := handlers.NewUsersHandler(store)
	identitiesH := handlers.NewIdentitiesHandler(store, s.jwtSecret/*, s.googleAudiences*/)
	socialH := handlers.NewSocialLinksHandler(store)
	payoutsH := handlers.NewPayoutsHandler(store)
	ledgerH := handlers.NewLedgerEventsHandler(store)
	ledgerIngestH, err := handlers.NewLedgerIngestHandler(store)
	if err != nil {
		log.Fatal(err)
	}
	claimsH, err := handlers.NewClaimsHandler(store)
	if err != nil {
		log.Fatal(err)
	}


	// Public routes
	public := s.router.Group("/api")
	{
		// Resolve (public) - used by extension
		public.GET("/resolve/youtube/:channelId", payoutsH.ResolveYouTubeChannelPayout)

		// Transactions
		public.POST("/ledger/deposit", ledgerIngestH.RecordDeposit)
	}

	// Auth routes
	auth := s.router.Group("/api/auth")
	{
		auth.POST("/wallet/message", identitiesH.GetWalletLoginMessage)
		auth.POST("/wallet", identitiesH.LoginWithWallet)
		//Will add back in future
		//auth.POST("/google", identitiesH.LoginWithGoogle)
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
		protected.POST("/payouts", payoutsH.UpsertPayout)

		// Earnings
		protected.GET("/me/earnings", ledgerH.GetEarningsSummary)
		protected.GET("/me/tips", ledgerH.ListMyTips)

		// Transactions
		protected.POST("/ledger/withdrawal", ledgerIngestH.RecordWithdrawal)

		// Claims
		protected.POST("/social/youtube/verify", socialH.VerifyYouTubeChannel)
		protected.POST("/claims/youtube", claimsH.SignYouTubeClaim)
	}

	return s
}

func (s *Server) Start(addr string) error {
	return s.router.Run(addr)
}
