package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/japb1998/control-tower/internal/dto"
	"github.com/japb1998/control-tower/internal/service"
	_ "github.com/swaggo/files"       // swagger embed files
	_ "github.com/swaggo/gin-swagger" // gin-swagger middleware
)

var (
	TableName           = os.Getenv("EMAIL_TABLE")
	ClientTable         = os.Getenv("CLIENT_TABLE")
	notificationService *service.NotificationService
)

type PaginatedNotifications struct {
	Limit int            `json:"limit"`
	Page  int            `json:"page"`
	Total int64          `json:"total"`
	Data  []Notification `json:"data"`
}

type PaginationOps struct {
	Page  int  `form:"page" json:"page" binding:"omitempty,min=0"`
	Limit *int `form:"limit" json:"limit" binding:"omitempty,min=1"`
}
type Notification struct {
	ID              string        `json:"id,omitempty"`
	Status          string        `json:"status"`
	Date            string        `json:"date"`
	CreatedBy       string        `json:"createdBy"`
	ClientToken     string        `json:"-"`
	DeliveryMethods []int8        `json:"deliveryMethods"`
	Client          dto.ClientDto `json:"client"`
}

// @BasePath /

// GetSchedules gets schedule by the user email obtained in the JWT token
// @Summary get schedules by creator.
// @Schemes
// @Description gets schedule by the user email obtained in the JWT token
// @Tags SCHEDULES
// @Param Authorization header string true "Bearer token"
// @Param page query integer false "Zero indexed" default(0)
// @Param limit query integer false "limit" default(10)
// @Accept json
// @Produce json
// @Success 200 {object} PaginatedNotifications
// @Router /schedule [get]
func GetSchedules(c *gin.Context) {

	userEmail := c.MustGet("email").(string)

	var ops = PaginationOps{
		Limit: aws.Int(10),
		Page:  0,
	}

	if err := c.ShouldBindWith(&ops, binding.Query); err != nil {
		notificationLogger.Error("error validating schedule filters", slog.String("error", err.Error()))
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

	clientLogger.Info("pagination ops", slog.Int("page", ops.Page), slog.Int("limit", *ops.Limit))

	svcOps := dto.PaginationOps{
		Page: ops.Page,
	}
	if ops.Limit == nil {
		svcOps.Limit = aws.Int(10)
	} else {
		svcOps.Limit = ops.Limit
	}

	res, err := notificationService.GetNotificationsByCreator(userEmail, &svcOps)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	nl, err := aggregateNotifications(c.Request.Context(), res.Data)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	sort.Slice(nl, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, nl[i].Date)
		timeJ, _ := time.Parse(time.RFC3339, nl[j].Date)

		return timeJ.After(timeI)
	})

	c.JSON(http.StatusOK, PaginatedNotifications{
		Data:  nl,
		Limit: *svcOps.Limit,
		Page:  ops.Page,
		Total: res.Total,
	})
}

// GetSchedule gets schedule by the id provided in the path parameters.
// @Summary get schedules by creator.
// @Schemes
// @Param Authorization header string true "Bearer token"
// @Tags SCHEDULES
// @Param id path string false "schedule ID"
// @Accept json
// @Produce json
// @Success 200 {object} PaginatedNotifications
// @Router /schedule/{id} [get]
func GetSchedule(c *gin.Context) {

	id, _ := c.Params.Get("id")
	userEmail := c.MustGet("email").(string)

	notificationLogger.Info("GetSchedule By", slog.String("id", id))

	notification, err := notificationService.GetNotification(userEmail, id)

	if err != nil {
		if errors.Is(err, service.ErrNotificationNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("notification ID='%s' not found.", id),
			})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("unable to retrieve notification ID='%s'", id),
		})
		return
	}

	c.JSON(http.StatusOK, notification)
}

// PostSchedule creates schedule.
// @Summary create schedule.
// @Schemes
// @Description create schedule.
// @Tags SCHEDULES
// @Param Authorization header string true "Bearer token"
// @Param request body service.NotificationInput true "body"
// @Accept json
// @Success 204
// @Router /schedule [post]
func PostSchedule(c *gin.Context) {
	ctx, startSpan := tracer.Start(c.Request.Context(), "post-schedule")
	defer startSpan.End()
	defer c.Request.Body.Close()
	userEmail := c.MustGet("email").(string)
	var schedule dto.NotificationInput

	if err := c.ShouldBindJSON(&schedule); err != nil {
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
			"error": "failed to create schedule",
		})
		return
	}
	clientCtx, clientSpan := tracer.Start(ctx, "get-client")

	user, err := clientService.GetClientById(clientCtx, userEmail, schedule.ClientId)

	if err != nil {
		notificationLogger.Error("Error getting client", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create schedule",
		})
		clientSpan.End()
		return
	}
	clientSpan.End()
	if *user.OptIn == false {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Client ID='%s' has notifications disabled.", user.Id),
		})
		return
	}

	_, scheduleSpan := tracer.Start(ctx, "create-schedule")

	id, err := notificationService.ScheduleNotification(userEmail, &schedule)

	if err != nil {
		if errors.Is(err, service.ErrInvalidDate) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create schedule",
		})
		scheduleSpan.End()
		return
	}
	scheduleSpan.End()

	var wg sync.WaitGroup
	/*
		TODO: move ws notification to eventBridge event handler.
	*/
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := sendUserNotification(c.Request.Context(), userEmail, id, service.NotificationCreatedAction); err != nil {
			notificationLogger.Error("error sending ws message", slog.String("error", err.Error()))
		}

	}()
	c.Writer.WriteHeader(204)
	wg.Wait()
}

// UpdateSchedule updates schedule.
// @Summary patch existing schedule by id.
// @Schemes
// @Description patch existing schedule by id.
// @Tags SCHEDULES
// @Param id path string false "Schedule ID"
// @Param Authorization header string true "Bearer token"
// @Param request body service.PatchNotification true "body"
// @Accept json
// @Produce json
// @Success 200 {object} service.Notification
// @Router /schedule/{id} [patch]
func UpdateSchedule(c *gin.Context) {

	userEmail := c.MustGet("email").(string)

	id := c.Params.ByName("id")
	var notification dto.PatchNotification
	if err := c.ShouldBindJSON(&notification); err != nil {
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
		notificationLogger.Error("error on validation", slog.String("error", err.Error()), "")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update schedule",
		})
		return
	}
	n, err := notificationService.UpdateNotification(userEmail, id, notification)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, n)
}

// DeleteSchedule deletes a schedule from both the scheduler service and db. if schedule has been remove from scheduler then it just removes it from the DB.
// DeleteSchedule deletes a schedule from both the scheduler service and db.
// @Summary deletes a schedule from both the scheduler service and db.
// @Schemes
// @Description deletes a schedule from both the scheduler service and db.
// @Tags SCHEDULES
// @Param Authorization header string true "Bearer token"
// @Param id path string false "Schedule ID"
// @Success 204
// @Router /schedule/{id} [delete]
func DeleteSchedule(c *gin.Context) {

	userEmail := c.MustGet("email").(string)
	notificationId, _ := c.Params.Get("id")
	err := notificationService.DeleteNotification(userEmail, notificationId)

	if err != nil {

		if errors.Is(err, service.ErrNotificationNotFound) {
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

	c.AbortWithStatus(http.StatusNoContent)
}

// aggregateNotifications receives a service.Notification List and retrieves a Notification with the full client struct.
func aggregateNotifications(ctx context.Context, nl []dto.Notification) ([]Notification, error) {

	notificationList := make([]Notification, 0, len(nl))
	errChan := make(chan error, 1)
	nChan := make(chan Notification, len(nl))

	for _, n := range nl {

		go func(n dto.Notification) {

			c, err := clientService.GetClientById(ctx, n.CreatedBy, n.ClientId)

			if err != nil {
				notificationLogger.Error("error getting user", slog.String("error", err.Error()))
				errChan <- fmt.Errorf("failed to get schedules")
				return
			}

			nChan <- Notification{
				ID:              n.ID,
				Status:          n.Status,
				Date:            n.Date,
				CreatedBy:       n.CreatedBy,
				ClientToken:     n.ClientToken,
				DeliveryMethods: n.DeliveryMethods,
				Client:          *c,
			}

		}(n)
	}

	for i := 0; i < len(nl); i++ {
		select {
		case n := <-nChan:
			notificationList = append(notificationList, n)
		case err := <-errChan:
			return nil, err
		}
	}

	return notificationList, nil
}

func sendUserNotification(ctx context.Context, email, notificationId, action string) error {

	msg, err := service.NewNotificationUpdateMsg(email, notificationId).WithAction(action)

	if err != nil {
		notificationLogger.Error("failed to notify using ws", slog.String("error", err.Error()))
		return err
	}
	err = connectionSvc.SendWsMessageByEmail(ctx, msg)

	if err != nil {
		notificationLogger.Error("failed to notify using", slog.String("error", err.Error()))
		return err
	}
	return nil
}
