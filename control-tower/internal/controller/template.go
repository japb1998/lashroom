package controller

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/japb1998/control-tower/internal/dto"
	"github.com/japb1998/control-tower/internal/service"
)

// GetTemplates get templates by creator.
// @Summary get templates by creator..
// @Schemes
// @Description gets schedule by the user email obtained in the JWT token
// @Tags SCHEDULES
// @Param Authorization header string true "Bearer token"
// @Param page query integer false "Zero indexed" default(0)
// @Param limit query integer false "limit" default(10)
// @Accept json
// @Produce json
// @Success 200 {object}
// @Router /templates [get]
func GetTemplates(c *gin.Context) {

	userEmail := c.MustGet("email").(string)

	var ops = PaginationOps{
		Limit: aws.Int(10),
		Page:  0,
	}

	if err := c.ShouldBindWith(&ops, binding.Query); err != nil {
		notificationLogger.Error("error validating pagination filters", slog.String("error", err.Error()))
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make([]ErrMsg, len(ve))

			for i, fe := range ve {
				out[i] = ErrMsg{
					Message: getErrorMsg(fe),
					Field:   fe.Field(),
				}
			}
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"errors": out,
			})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to validate query parameters",
		})
		return
	}

	templateLogger.Info("pagination ops", slog.Int("page", ops.Page), slog.Int("limit", *ops.Limit))

	svcOps := dto.PaginationOps{
		Page: ops.Page,
	}
	if ops.Limit == nil {
		svcOps.Limit = aws.Int(10)
	} else {
		svcOps.Limit = ops.Limit
	}

	res, err := templateSvc.GetPaginatedTemplates(c.Request.Context(), userEmail, &svcOps)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// CreateTemplate create template.
/* TODO schema	*/
func CreateTemplate(c *gin.Context) {

	requestor := c.MustGet("email").(string)

	var template dto.CreateTemplateDto

	if err := c.ShouldBindJSON(&template); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make([]ErrMsg, len(ve))

			for i, fe := range ve {
				out[i] = ErrMsg{
					Message: getErrorMsg(fe),
					Field:   fe.Field(),
				}
			}
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"errors": out,
			})
			return
		}
		notificationLogger.Error("error on validation", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create template",
		})
		return
	}

	//  check for existing template
	if _, err := templateSvc.GetTemplate(c.Request.Context(), requestor, template.Name); err != nil {
		if !errors.Is(err, service.ErrTemplateNotFound) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		notificationLogger.Info("template not found", slog.String("template", template.Name))
	} else {
		notificationLogger.Error("template already exists")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("template with the provided name already exists name=%s", template.Name),
		})
		return
	}

	err := templateSvc.CreateTemplate(c.Request.Context(), requestor, template)

	if err != nil {
		notificationLogger.Error("error creating template", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

// GetTemplate get template by name.
/* TODO schema	*/
func GetTemplate(c *gin.Context) {
	requestor := c.MustGet("email").(string)

	templateName := c.Param("name")

	t, err := templateSvc.GetTemplate(c.Request.Context(), requestor, templateName)

	if err != nil {
		templateLogger.Error("error getting template", slog.String("error", err.Error()))
		if errors.Is(err, service.ErrTemplateNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, t)
}

// DeleteTemplate delete template by name.
func DeleteTemplate(c *gin.Context) {
	requestor := c.MustGet("email").(string)

	templateName := c.Param("name")

	err := templateSvc.DeleteTemplate(c.Request.Context(), requestor, templateName)

	if err != nil {
		templateLogger.Error("error deleting template", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

// UpdateTemplate update template by name.
func UpdateTemplate(c *gin.Context) {
	requestor := c.MustGet("email").(string)

	templateName := c.Param("name")

	var template dto.UpdateTemplateDto

	if err := c.ShouldBindJSON(&template); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make([]ErrMsg, len(ve))

			for i, fe := range ve {
				out[i] = ErrMsg{
					Message: getErrorMsg(fe),
					Field:   fe.Field(),
				}
			}
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"errors": out,
			})
			return
		}
		notificationLogger.Error("error on validation", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create template",
		})
		return
	}

	err := templateSvc.UpdateTemplate(c.Request.Context(), requestor, templateName, template)

	if err != nil {
		templateLogger.Error("error updating template", slog.String("error", err.Error()))
		var status = http.StatusInternalServerError
		if errors.Is(err, service.ErrTemplateNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}
