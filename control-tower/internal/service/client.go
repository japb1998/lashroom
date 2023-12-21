package service

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/model"
)

var clientLogger = log.New(os.Stdout, "[Client Service]", log.Default().Flags())

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
type FiltersResponseDto struct {
	Data  []ClientDto `json:"data"`
	Limit int         `json:"limit"`
	Page  int         `json:"page"`
	Total int64       `json:"total"`
}
type ClientDto struct {
	Id           string `json:"id"`
	CreatedBy    string `json:"createdBy"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	CreatedAt    string `json:"createdAt"`
	LastUpdateAt string `json:"lastUpdateAt"`
	Description  string `json:"description"`
	OptIn        *bool  `json:"optIn"`
}

type CreateClient struct {
	Phone        string `json:"phone" binding:"omitempty,e164"`
	Email        string `json:"email" binding:"omitempty,email"`
	FirstName    string `json:"firstName" binding:"required"`
	LastName     string `json:"lastName" binding:"required"`
	CreatedAt    string `json:"createdAt" binding:"omitempty,rfc3339"`
	LastUpdateAt string `json:"lastUpdateAt" binding:"omitempty,rfc3339"`
	Description  string `json:"description" binding:"min=2,max=255"`
}
type PatchClient struct {
	Phone       string `json:"phone" form:"phone" binding:"omitempty,e164"`
	Email       string `json:"email" form:"email" binding:"omitempty,email"`
	FirstName   string `json:"firstName" form:"firstName" binding:"omitempty,min=1"`
	LastName    string `json:"lastName" form:"lastName" binding:"omitempty,min=1"`
	Description string `json:"description" form:"description" binding:"omitempty,min=2,max=255"`
	OptIn       *bool  `json:"optIn" form:"optIn" binding:"omitempty,boolean"`
}

type ClientPaginationDto struct {
	PatchClient
	PaginationOps
}

func NewClient(createdBy, firstName, lastName, createdAt, lastUpdatedAt, description, phone, email, id string, opt bool) *ClientDto {
	return &ClientDto{
		Id:           id,
		CreatedBy:    createdBy,
		Phone:        phone,
		Email:        email,
		FirstName:    firstName,
		LastName:     lastName,
		CreatedAt:    createdAt,
		LastUpdateAt: lastUpdatedAt,
		Description:  description,
		OptIn:        &opt,
	}
}

func NewClientFromItem(ci model.ClientItem) *ClientDto {
	return &ClientDto{
		Id:           ci.SortKey,
		CreatedBy:    ci.PrimaryKey,
		Phone:        ci.Phone,
		Email:        ci.Email,
		FirstName:    ci.FirstName,
		LastName:     ci.LastName,
		CreatedAt:    ci.CreatedAt.Format(time.RFC3339),
		LastUpdateAt: ci.LastUpdateAt.Format(time.RFC3339),
		Description:  ci.Description,
		OptIn:        &ci.OptIn,
	}
}
func NewClientSvc(s ClientRepository) *ClientService {

	return &ClientService{
		Store: s,
	}
}
func (c *ClientService) GetClientsByCreator(createdBy string) ([]ClientDto, error) {
	clientList, err := c.Store.GetClientsByCreator(createdBy)

	if err != nil {
		return nil, err
	}

	dtos := make([]ClientDto, 0, len(clientList))

	for _, c := range clientList {
		dtos = append(dtos, *NewClientFromItem(c))
	}
	return dtos, err
}

func (c *ClientService) UpdateUser(createdBy string, clientId string, client PatchClient) (ClientDto, error) {

	patch := database.PatchClientItem{
		Phone:       client.Phone,
		Email:       client.Email,
		FirstName:   client.FirstName,
		LastName:    client.LastName,
		Description: client.Description,
		OptIn:       client.OptIn,
	}
	clientLogger.Println("Updating User payload=", patch)

	item, err := c.Store.UpdateUser(createdBy, clientId, patch)

	if err != nil {
		return ClientDto{}, err
	}

	return *NewClientFromItem(item), nil
}

func (c *ClientService) CreateClient(createdBy string, client CreateClient) (ClientDto, error) {
	item := model.NewClientItem(createdBy, client.Phone, client.Email, client.FirstName, client.LastName, client.Description)
	_, err := c.Store.CreateClient(*item)

	if err != nil {
		return ClientDto{}, err
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

func (c *ClientService) GetClientById(createdBy, id string) (*ClientDto, error) {
	item, err := c.Store.GetClientById(createdBy, id)

	if err != nil {
		return nil, err
	}

	return NewClientFromItem(*item), nil
}

// GetClientWithFilters get clients with filters. Paginated, Zero indexed
func (c *ClientService) GetClientWithFilters(createdBy string, dto ClientPaginationDto) (FiltersResponseDto, error) {

	f := database.PatchClientItem{
		Phone:     dto.Phone,
		Email:     dto.Email,
		FirstName: dto.FirstName,
		LastName:  dto.LastName,
	}

	paginatioOps := database.PaginationOps{
		Limit: dto.Limit,
		Skip:  dto.Limit * dto.Page,
	}
	errChan := make(chan error, 2)
	itemCountChan := make(chan int64, 1)
	itemsListChan := make(chan []ClientDto, 1)
	var itemCount int64
	var clientList []ClientDto

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
			dtoList := make([]ClientDto, 0, len(items))

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
				return FiltersResponseDto{}, err
			}
		}
	}
	close(errChan)
	close(itemCountChan)
	close(itemsListChan)

	return FiltersResponseDto{
		Total: itemCount,
		Data:  clientList,
		Page:  dto.Page,
		Limit: dto.Limit,
	}, nil
}

func (c *ClientService) OptOut(createdBy, clientId string) error {
	patch := database.PatchClientItem{
		OptIn: aws.Bool(false),
	}
	if _, err := c.Store.UpdateUser(createdBy, clientId, patch); err != nil {
		clientLogger.Printf("Unable to unsubscribe user createdBy='%s', clientId='%s' error=%s", createdBy, clientId, err)
		return fmt.Errorf("Unable to unsubscribe user")
	}

	return nil
}
