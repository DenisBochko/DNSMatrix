package route

import "github.com/gin-gonic/gin"

type ArticleHandler interface {
	CreateArticle(c *gin.Context)
	GetArticle(c *gin.Context)
	DeleteArticle(c *gin.Context)
	UpdateArticle(c *gin.Context)
	SearchArticles(c *gin.Context)
}

func RegisterArticleRoutes(g *gin.RouterGroup, h ArticleHandler, jwtAuthMiddleware gin.HandlerFunc, allowManagerAndAdminMiddleware gin.HandlerFunc) {
	protected := g.Group("", jwtAuthMiddleware, allowManagerAndAdminMiddleware)
	protected.POST("", h.CreateArticle)
	protected.GET("/:article_id", h.GetArticle)
	protected.DELETE("/:article_id", h.DeleteArticle)
	protected.PATCH("/:article_id", h.UpdateArticle)
	protected.GET("/search", h.SearchArticles)
}
