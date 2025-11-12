package route

import (
	"crypto/ecdsa"
	"hackathon-back/internal/api/http/handler"
	"hackathon-back/internal/api/http/middleware"
	"io"

	"hackathon-back/internal/config"
	"hackathon-back/internal/model"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const maxMultipartMemory = 1 << 30

func SetupRouter(
	log *zap.Logger,
	cfg *config.Config,
	publicKey *ecdsa.PublicKey,
	healthHdl HealthHandler,
	authHdl AuthHandler,
	userHdl UserHandler,
	articleHdl ArticleHandler,
	apiKeyRepo middleware.APIKeyRepositoryInterface,
	faqHdl FAQHandler,
	reqHdl RequestHandler,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	router := gin.Default()
	router.MaxMultipartMemory = maxMultipartMemory

	// middleware
	router.Use(middleware.Logger(log))
	router.Use(middleware.RequestTimeout(cfg.HTTPServer.Timeout.Request))
	router.Use(middleware.CORS(cfg.CORS))

	jwtAuthMiddleware := middleware.JWTAuth(publicKey)
	allowManagerAndAdminMiddleware := middleware.RequireRoles(model.RoleManager, model.RoleAdmin)
	apiKeyMiddleware := middleware.APIKeyAuthMiddleware(apiKeyRepo)

	router.HandleMethodNotAllowed = true
	router.NoMethod(handler.NoMethod)
	router.NoRoute(handler.NoRoute)

	basePath := router.Group(cfg.BasePath)

	docsPath := basePath.Group("/docs")
	RegisterDock(docsPath)

	healthPath := basePath.Group("/health")
	RegisterHealth(healthPath, healthHdl, jwtAuthMiddleware)

	authPath := basePath.Group("/auth")
	RegisterAuth(authPath, authHdl)

	userPath := basePath.Group("/user")
	RegisterAdminUserRoutes(userPath, userHdl, jwtAuthMiddleware, allowManagerAndAdminMiddleware)

	requestPath := basePath.Group("/check")
	RegisterRequestRoutes(requestPath, reqHdl)

	articleGroup := basePath.Group("/article")
	RegisterArticleRoutes(articleGroup, articleHdl, jwtAuthMiddleware, allowManagerAndAdminMiddleware)

	// 🔑 API Key защищенные маршруты (для приложений)
	apiGroup := basePath.Group("/api-key")
	apiGroup.Use(apiKeyMiddleware)
	{
		// Доступ к API через API Key
		apiGroup.GET("/articles", articleHdl.SearchArticles)
		apiGroup.GET("/articles/:id", articleHdl.GetArticle)
		apiGroup.POST("/articles", articleHdl.CreateArticle)
		apiGroup.PATCH("/articles/:id", articleHdl.UpdateArticle)
		apiGroup.DELETE("/articles/:id", articleHdl.DeleteArticle)
	}

	faqPath := basePath.Group("/faq")
	RegisterFAQRoutes(faqPath, faqHdl, jwtAuthMiddleware, allowManagerAndAdminMiddleware)

	return router
}
