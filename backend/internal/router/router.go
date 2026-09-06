// Package router khai bao toan bo route cua service.
package router

import (
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"

	_ "github.com/Cinema-Project-Juann/BackEnd-CP/docs" // swagger docs sinh boi `make swag`
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/config"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/handlers"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/jwt"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// Handlers gom cac handler duoc wire o main.
type Handlers struct {
	Health *handlers.HealthHandler
	Auth   *handlers.AuthHandler
	User   *handlers.UserHandler
	Movie  *handlers.MovieHandler
}

// New dung gin.Engine da gan day du middleware va route.
func New(cfg *config.Config, jwtManager *jwt.Manager, h Handlers) *gin.Engine {
	if cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	registerValidators()

	engine := gin.New()
	engine.RedirectTrailingSlash = false
	engine.Use(
		middleware.RequestID(),
		middleware.Recovery(),
		middleware.Logger(),
		middleware.CORS(cfg.CORS.AllowedOrigins),
	)

	engine.NoRoute(func(c *gin.Context) {
		response.Error(c, apperrors.NotFound("route not found"))
	})
	engine.NoMethod(func(c *gin.Context) {
		response.Error(c, apperrors.BadRequest("method not allowed"))
	})

	// Probe cho ha tang (k8s, docker healthcheck).
	engine.GET("/health", h.Health.Check)

	if !cfg.App.IsProduction() {
		engine.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
	}

	v1 := engine.Group("/api/v1")
	v1.GET("/health", h.Health.Check)

	auth := v1.Group("/auth")
	{
		auth.POST("/register", h.Auth.Register)
		auth.POST("/login", h.Auth.Login)
		auth.POST("/refresh", h.Auth.Refresh)
	}

	protected := v1.Group("")
	protected.Use(middleware.Auth(jwtManager))
	{
		protected.GET("/users/me", h.User.Me)

		movies := protected.Group("/movies")
		{
			movies.GET("", h.Movie.List)
			movies.GET("/:id", h.Movie.Detail)

			// Thao tac ghi chi danh cho quan tri vien va nhan vien.
			writable := movies.Group("")
			writable.Use(middleware.RequireRoles(models.RoleAdmin, models.RoleStaff))
			{
				writable.POST("", h.Movie.Create)
				writable.PUT("/:id", h.Movie.Update)
				writable.DELETE("/:id", h.Movie.Delete)
			}
		}
	}

	return engine
}
