package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type ArticleHandler struct {
	articleService service.ArticleService
}

func NewArticleHandler(articleService service.ArticleService) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

// List godoc
//
//	@Summary		List published articles (public)
//	@Tags			articles
//	@Produce		json
//	@Param			page		query		int		false	"Current page"	default(1)
//	@Param			page_size	query		int		false	"Records per page"	default(10)
//	@Param			type		query		string	false	"news or promotion"
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.ArticleResponse}}
//	@Router			/articles [get]
func (h *ArticleHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()
	articles, total, err := h.articleService.List(c.Request.Context(), true, c.Query("type"), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, articles, query.Page, query.PageSize, total)
}

// ListAdmin godoc
//
//	@Summary		List all articles, any status (admin)
//	@Tags			articles
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Current page"	default(1)
//	@Param			page_size	query		int		false	"Records per page"	default(10)
//	@Param			type		query		string	false	"news or promotion"
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.ArticleResponse}}
//	@Router			/admin/articles [get]
func (h *ArticleHandler) ListAdmin(c *gin.Context) {
	var query dto.PageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()
	articles, total, err := h.articleService.List(c.Request.Context(), false, c.Query("type"), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, articles, query.Page, query.PageSize, total)
}

// Show godoc
//
//	@Summary		Get a published article by slug (public)
//	@Tags			articles
//	@Produce		json
//	@Param			slug	path		string	true	"Article slug"
//	@Success		200		{object}	response.Body{data=dto.ArticleResponse}
//	@Failure		404		{object}	response.Body
//	@Router			/articles/{slug} [get]
func (h *ArticleHandler) Show(c *gin.Context) {
	article, err := h.articleService.GetBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, article)
}

// ShowAdmin godoc
//
//	@Summary		Get an article by ID, any status (admin)
//	@Tags			articles
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Article ID"
//	@Success		200	{object}	response.Body{data=dto.ArticleResponse}
//	@Failure		404	{object}	response.Body
//	@Router			/admin/articles/{id} [get]
func (h *ArticleHandler) ShowAdmin(c *gin.Context) {
	article, err := h.articleService.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, article)
}

// Create godoc
//
//	@Summary		Create an article
//	@Tags			articles
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.ArticleRequest	true	"Article"
//	@Success		201		{object}	response.Body{data=dto.ArticleResponse}
//	@Failure		400		{object}	response.Body
//	@Router			/admin/articles [post]
func (h *ArticleHandler) Create(c *gin.Context) {
	var req dto.ArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	article, err := h.articleService.Create(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, article)
}

// Update godoc
//
//	@Summary		Update an article
//	@Tags			articles
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Article ID"
//	@Param			payload	body		dto.UpdateArticleRequest	true	"Fields to change"
//	@Success		200		{object}	response.Body{data=dto.ArticleResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/articles/{id} [put]
func (h *ArticleHandler) Update(c *gin.Context) {
	var req dto.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	article, err := h.articleService.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, article)
}

// Delete godoc
//
//	@Summary		Delete an article (soft delete)
//	@Tags			articles
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Article ID"
//	@Success		204	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/admin/articles/{id} [delete]
func (h *ArticleHandler) Delete(c *gin.Context) {
	if err := h.articleService.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
