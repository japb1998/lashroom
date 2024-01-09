package controller

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/japb1998/control-tower/internal/service"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer trace.Tracer
)
var clientService *service.ClientService

func GetClientsByCreator(c *gin.Context) {

	userEmail := c.MustGet("email").(string)

	if clientDtoList, err := clientService.GetClientsByCreator(c.Request.Context(), userEmail); err != nil {

		clientLogger.Error("GetClientsByCreator", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{
			"message": "Error while retrieving clients",
		})

	} else {

		c.JSON(http.StatusOK, gin.H{
			"records": clientDtoList,
			"count":   len(clientDtoList),
		})
	}
}

// CreateClient create client.
// @Tags CLIENT
// @Summary create client.
// @Schemes
// @Description create client.
// @Param Authorization header string true "Bearer token"
// @Param request body service.CreateClient true "create client dto"
// @Accept json
// @Produce json
// @Success 200 {object} service.ClientDto
// @Router /clients [post]
func CreateClient(c *gin.Context) {
	userEmail := c.MustGet("email").(string)
	var clientDto service.CreateClient

	if err := c.ShouldBindJSON(&clientDto); err != nil {
		clientLogger.Error("CreateClient validation error", slog.String("error", err.Error()))
		var ve validator.ValidationErrors

		if errors.As(err, &ve) {
			output := make([]ErrMsg, len(ve))
			fmt.Println(ve)
			for i, fe := range ve {
				output[i] = ErrMsg{
					Message: getErrorMsg(fe),
					Field:   fe.Field(),
				}
				clientLogger.Error("CreateClient validation error", slog.String("error", fe.Error()))
			}
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"errors": output,
			})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "error creating user",
		})
		return
	}

	if client, err := clientService.CreateClient(c.Request.Context(), userEmail, clientDto); err != nil {
		clientLogger.Error("Error creating user", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "error creating user"})
		return
	} else {
		c.JSON(http.StatusOK, client)
	}

}

// UpdateClient update client.
// @Tags CLIENT
// @Summary update client.
// @Schemes
// @Description update client.
// @Param Authorization header string true "Bearer token"
// @Param request body service.PatchClient true "patch client dto"
// @Produce json
// @Success 200 {object} service.ClientDto
// @Router /clients/{id} [patch]
func UpdateClient(c *gin.Context) {
	userEmail := c.MustGet("email").(string)
	clientId, _ := c.Params.Get("id")
	var clientDto service.PatchClient

	if err := c.ShouldBindJSON(&clientDto); err != nil {
		clientLogger.Error("Error validating UpdateClient payload", slog.String("error", err.Error()))
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
			"error": "failed to validate input.",
		})
		return
	}

	if client, err := clientService.UpdateUser(c.Request.Context(), userEmail, clientId, clientDto); err != nil {
		clientLogger.Error("Error updating user", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	} else {
		c.JSON(http.StatusOK, client)
	}
}

// GetClientByID Get client by ID
// @Tags CLIENT
// @Summary Get client by ID
// @Schemes
// @Description Get client by ID
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Client ID"
// @Produce json
// @Success 200 {object} service.ClientDto
// @Router /clients/{id} [get]
func GetClientByID(c *gin.Context) {
	userEmail := c.MustGet("email").(string)
	clientId, _ := c.Params.Get("id")

	client, err := clientService.GetClientById(c.Request.Context(), userEmail, clientId)

	if err != nil {
		clientLogger.Error("Error Getting client by ID", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "error getting user"})
		return
	} else if client == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, client)
}

// DeleteClient delete client by ID.
// @Tags CLIENT
// @Summary delete client by ID.
// @Schemes
// @Description delete client by ID.
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Client ID"
// @Produce json
// @Success 200 {object} service.ClientDto
// @Router /clients/{id} [delete]
func DeleteClient(c *gin.Context) {
	userEmail := c.MustGet("email").(string)
	clientId, _ := c.Params.Get("id")

	err := clientService.DeleteClient(userEmail, clientId)

	if err != nil {
		clientLogger.Error("Error deleting client", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "error deleting user"})
		return
	}

	c.AbortWithStatus(http.StatusNoContent)
}

// DeleteClient mark client as not available for notifications.
// @Tags CLIENT
// @Summary mark client as not available for notifications.
// @Schemes
// @Description mark client as not available for notifications.
// @Param id path string true "Client ID"
// @Param createdBy path string true "Client ID"
// @Success 301
// @Router /clients/{createdBy}/{id} [get]
func OptOut(c *gin.Context) {
	creator, ok := c.Params.Get("creator")
	if !ok || creator == "" {
		fmt.Printf("creator not present")
		return
	}

	client, ok := c.Params.Get("userID")
	if !ok || client == "" {
		fmt.Printf("client not present")
		return
	}

	if err := clientService.OptOut(c.Request.Context(), creator, client); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.Redirect(http.StatusMovedPermanently, "https://lashroombyeli.com")
}

// ClientsWithFilters get clients by creator with filters.
// @Tags CLIENT
// @Summary get clients by creator with filters.
// @Schemes
// @Description get clients by creator with filters.
// @Param Authorization header string true "Bearer token"
// @Param phone query string false "Phone number to filter by"
// @Param email query string false "email to filter by"
// @Param firstName query string false "First Name to filter by"
// @Param lastName query string false "Last Name to filter by"
// @Param page query int false "page number. Zero Indexed" default(0)
// @Param limit query int false "max number of records." default(10)
// @Produce json
// @Success 200 {object} service.FiltersResponseDto
// @Router /clients [get]
func ClientsWithFilters(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "client-with-filter-controller")
	defer span.End()

	requestorEmail := c.MustGet("email").(string)
	var paginationDto service.ClientPaginationDto
	if err := c.ShouldBindWith(&paginationDto, binding.Query); err != nil {
		clientLogger.Error("Error on filters validation", slog.String("error", err.Error()))
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

	// default pagination value
	if paginationDto.Limit == 0 {
		paginationDto.Limit = 10
	}

	childCtx, childSpan := tracer.Start(ctx, "client-service")
	childSpan.AddEvent("get-client")

	defer childSpan.End()

	if response, err := clientService.GetClientWithFilters(childCtx, requestorEmail, paginationDto); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	} else {
		c.JSON(http.StatusOK, response)
		return
	}
}
