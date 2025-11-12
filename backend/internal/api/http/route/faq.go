// route/faq.go
package route

import (
	"github.com/gin-gonic/gin"
)

type FAQHandler interface {
	CreateFAQ(c *gin.Context)
	GetFAQ(c *gin.Context)
	UpdateFAQ(c *gin.Context)
	DeleteFAQ(c *gin.Context)
	ListFAQs(c *gin.Context)
	GetFAQsByCategory(c *gin.Context)
	GetCategories(c *gin.Context)
	GetCategoriesWithFAQs(c *gin.Context)
}

func RegisterFAQRoutes(g *gin.RouterGroup, h FAQHandler, jwtAuthMiddleware, allowManagerAndAdminMiddleware gin.HandlerFunc) {
	// Публичные маршруты
	public := g.Group("/faq")
	{
		public.GET("", h.ListFAQs)
		public.GET("/categories", h.GetCategories)
		public.GET("/grouped", h.GetCategoriesWithFAQs)
		public.GET("/category/:category", h.GetFAQsByCategory)
		public.GET("/:id", h.GetFAQ)
	}

	// Защищенные маршруты (требуют авторизации)
	protected := g.Group("/faq", jwtAuthMiddleware)
	{
		protected.POST("", allowManagerAndAdminMiddleware, h.CreateFAQ)
		protected.PATCH("/:id", allowManagerAndAdminMiddleware, h.UpdateFAQ)
		protected.DELETE("/:id", allowManagerAndAdminMiddleware, h.DeleteFAQ)
	}
}
