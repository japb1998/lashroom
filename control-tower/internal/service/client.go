package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"log/slog"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/dto"
	"github.com/japb1998/control-tower/internal/model"
)

// Errors

var (
	ErrInvalidDateString = errors.New("provided input string is not a valid date.")
)

type ClientRepository interface {
	GetClientsByCreator(string) ([]model.ClientItem, error)
	UpdateUser(createdBy string, userId string, client database.PatchClientItem) (model.ClientItem, error)
	CreateClient(model.ClientItem) (model.ClientItem, error)
	DeleteClient(createdBy, id string) error
	GetClientById(createdBy, id string) (*model.ClientItem, error)
	GetClientWithFilters(createdBy string, clientDto database.PatchClientItem, p *database.PaginationOps) ([]model.ClientItem, error)
	ClientCountWithFilters(createdBy string, clientPatch database.PatchClientItem) (int64, error)
}

type ClientService struct {
	Store ClientRepository
}

func NewClient(createdBy, firstName, lastName, createdAt, lastUpdatedAt, description, phone, email, id, lastSeen string, opt bool) *dto.ClientDto {
	return &dto.ClientDto{
		Id:           id,
		CreatedBy:    createdBy,
		Phone:        phone,
		Email:        email,
		FirstName:    firstName,
		LastName:     lastName,
		CreatedAt:    createdAt,
		LastUpdateAt: lastUpdatedAt,
		LastSeen:     &lastSeen,
		Description:  description,
		OptIn:        &opt,
	}
}

func NewClientFromItem(ci model.ClientItem) *dto.ClientDto {
	var lastSeen *string
	if ci.LastSeen != nil {
		lastSeen = aws.String(ci.LastSeen.Format(time.RFC3339))
	}
	return &dto.ClientDto{
		Id:           ci.SortKey,
		CreatedBy:    ci.PrimaryKey,
		Phone:        ci.Phone,
		Email:        ci.Email,
		FirstName:    ci.FirstName,
		LastName:     ci.LastName,
		CreatedAt:    ci.CreatedAt.Format(time.RFC3339),
		LastUpdateAt: ci.LastUpdateAt.Format(time.RFC3339),
		LastSeen:     lastSeen,
		Description:  ci.Description,
		OptIn:        &ci.OptIn,
	}
}
func NewClientSvc(s ClientRepository) *ClientService {

	return &ClientService{
		Store: s,
	}
}
func (c *ClientService) GetClientsByCreator(ctx context.Context, createdBy string) ([]dto.ClientDto, error) {
	clientList, err := c.Store.GetClientsByCreator(createdBy)

	if err != nil {
		return nil, err
	}

	dtos := make([]dto.ClientDto, 0, len(clientList))

	for _, c := range clientList {
		dtos = append(dtos, *NewClientFromItem(c))
	}
	return dtos, err
}

func (c *ClientService) UpdateUser(ctx context.Context, createdBy, clientId string, client dto.PatchClient) (dto.ClientDto, error) {

	patch := database.PatchClientItem{
		Phone:       client.Phone,
		Email:       client.Email,
		FirstName:   client.FirstName,
		LastName:    client.LastName,
		Description: client.Description,
		OptIn:       client.OptIn,
	}

	if client.LastSeen != nil {
		lastSeen, err := time.Parse(time.RFC3339, *client.LastSeen)

		if err != nil {
			clientLogger.Error(err.Error())
			return dto.ClientDto{}, fmt.Errorf("lastSeen could not me converted to date error='%s'", err)
		}
		patch.LastSeen = &lastSeen
	}

	clientLogger.Info("Updating User", slog.Any("patch", patch))

	item, err := c.Store.UpdateUser(createdBy, clientId, patch)

	if err != nil {
		clientLogger.Error(err.Error())
		return dto.ClientDto{}, err
	}

	return *NewClientFromItem(item), nil
}

func (c *ClientService) CreateClient(ctx context.Context, createdBy string, client dto.CreateClient) (dto.ClientDto, error) {
	lastSeen, err := time.Parse(time.RFC3339, *client.LastSeen)
	if err != nil {
		log.Printf("failed to convert lastSeen error:'%s'\n", err)
		return dto.ClientDto{}, ErrInvalidDateString
	}
	item := model.NewClientItem(createdBy, client.Phone, strings.ToLower(client.Email), strings.ToLower(client.FirstName), strings.ToLower(client.LastName), client.Description, &lastSeen)

	_, err = c.Store.CreateClient(*item)

	if err != nil {
		clientLogger.Error(err.Error())
		return dto.ClientDto{}, err
	}

	return *NewClientFromItem(*item), nil
}

func (c *ClientService) DeleteClient(createdBy, id string) error {
	err := c.Store.DeleteClient(createdBy, id)

	if err != nil {
		return err
	}

	return nil
}

func (c *ClientService) GetClientById(ctx context.Context, createdBy, id string) (*dto.ClientDto, error) {
	item, err := c.Store.GetClientById(createdBy, id)

	if err != nil {
		clientLogger.Error(err.Error())
		return nil, err
	}

	return NewClientFromItem(*item), nil
}

// GetClientWithFilters get clients with filters. Paginated, Zero indexed
func (c *ClientService) GetClientWithFilters(ctx context.Context, createdBy string, d dto.ClientPaginationDto) (dto.PaginatedResponse[dto.ClientDto], error) {

	var lastSeen *time.Time
	if d.LastSeen != nil {
		ls, err := time.Parse(time.RFC3339, *d.LastSeen)

		if err != nil {
			clientLogger.Error(err.Error())
			return dto.PaginatedResponse[dto.ClientDto]{}, fmt.Errorf("failed to convert lastSeen Date error='%s'", ErrInvalidDateString)
		}
		clientLogger.Info("lastSeen", slog.Time("at", ls))
		lastSeen = &ls
	}

	f := database.PatchClientItem{
		Phone:     d.Phone,
		Email:     d.Email,
		FirstName: d.FirstName,
		LastName:  d.LastName,
		LastSeen:  lastSeen,
	}

	paginatioOps := database.PaginationOps{
		Limit: *d.Limit,
		Skip:  *d.Limit * d.Page,
	}
	errChan := make(chan error, 2)
	itemCountChan := make(chan int64, 1)
	itemsListChan := make(chan []dto.ClientDto, 1)
	var itemCount int64
	var clientList []dto.ClientDto

	// get client count. TOTAL
	go func() {
		if count, err := c.Store.ClientCountWithFilters(createdBy, f); err != nil {
			errChan <- err
		} else {
			itemCountChan <- count
		}
	}()

	// get client items. Paginated, Zero Indexed.
	go func() {
		if items, err := c.Store.GetClientWithFilters(createdBy, f, &paginatioOps); err != nil {
			errChan <- err
		} else {
			dtoList := make([]dto.ClientDto, 0, len(items))

			for _, i := range items {
				dtoList = append(dtoList, *NewClientFromItem(i))
			}
			itemsListChan <- dtoList
		}
	}()

	for i := 0; i < 2; i++ {
		select {
		case t := <-itemCountChan:
			{
				itemCount = t
			}
		case t := <-itemsListChan:
			{
				clientList = t
			}
		case err := <-errChan:
			{
				return dto.PaginatedResponse[dto.ClientDto]{}, err
			}
		}
	}
	close(errChan)
	close(itemCountChan)
	close(itemsListChan)

	clientLogger.Info("Successfully retrieved clients.")
	return dto.PaginatedResponse[dto.ClientDto]{
		Total: itemCount,
		Data:  clientList,
		Page:  d.Page,
		Limit: *d.Limit,
	}, nil
}

func (c *ClientService) OptOut(ctx context.Context, createdBy, clientId string) error {
	patch := database.PatchClientItem{
		OptIn: aws.Bool(false),
	}
	if _, err := c.Store.UpdateUser(createdBy, clientId, patch); err != nil {
		clientLogger.Error("Unable to unsubscribe user createdBy='%s', clientId='%s' error=%s", createdBy, clientId, err)
		return fmt.Errorf("Unable to unsubscribe user")
	}

	return nil
}
