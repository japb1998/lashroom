package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"os"

	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/dto"
	"github.com/japb1998/control-tower/internal/mapper"
	"github.com/japb1998/control-tower/internal/model"
)

// errors
var (
	ErrTemplateNotFound      = errors.New("template not found")
	ErrorInvalidTemplateType = errors.New("invalid template type")
)

type TemplateRepository interface {
	Create(ctx context.Context, t *model.TemplateItem) error
	Update(ctx context.Context, name, createdBy string, t *model.UpdateTemplate) (*model.TemplateItem, error)
	GetByKey(ctx context.Context, creator string, name string) (*model.TemplateItem, error)
	GetByCreator(ctx context.Context, creator string, p *database.PaginationOps) ([]*model.TemplateItem, error)
	Delete(ctx context.Context, name, creator string) error
	GetTotalCount(ctx context.Context, creator string) (int64, error)
}

type TemplateSvc struct {
	store  TemplateRepository
	logger *slog.Logger
}

func NewTemplateSvc(store TemplateRepository) *TemplateSvc {
	l := slog.New(slog.NewTextHandler(os.Stdout, nil).WithAttrs([]slog.Attr{slog.String("name", "template-service")}))
	return &TemplateSvc{
		store,
		l,
	}
}

func (ts *TemplateSvc) CreateTemplate(ctx context.Context, creator string, template dto.CreateTemplateDto) error {
	// template types are directly correlated to notification types. /** template types table is meant to exist in the future */
	switch template.TemplateType {
	case int8(Email):
		break
	case int8(Whatsapp):
		break
	default:
		return ErrorInvalidTemplateType
	}

	t := mapper.MapCreateTemplateDtoToModel(ctx, creator, template)
	err := ts.store.Create(ctx, &t)

	if err != nil {
		ts.logger.Error("failed to create template", slog.String("error", err.Error()))
		return fmt.Errorf("failed to create template.")
	}

	return nil
}

func (ts *TemplateSvc) UpdateTemplate(ctx context.Context, name, creator string, update dto.UpdateTemplateDto) error {

	ts.logger.Info("updating template.", slog.Any("update", update))

	ts.logger.Info("checking if template exists")
	i, err := ts.store.GetByKey(ctx, creator, name)

	if err != nil {
		return fmt.Errorf("failed to retrieve template")
	}

	if i == nil {
		return ErrTemplateNotFound
	}

	m := mapper.MapTemplateUpdateToModel(ctx, update)

	_, err = ts.store.Update(ctx, name, creator, &m)

	if err != nil {
		ts.logger.Error("failed to update template", slog.String("error", err.Error()))
		return fmt.Errorf("failed to update template")
	}

	ts.logger.Info("successfully updated template")

	return nil
}

// GetPaginatedTemplates - retrieves templates for a specific creator/client with pagination options.
func (ts *TemplateSvc) GetPaginatedTemplates(ctx context.Context, creator string, ops *dto.PaginationOps) (dto.PaginatedResponse[dto.TemplateDto], error) {
	ts.logger.Info("getting templates by creator", slog.String("creator", creator))
	itemChan := make(chan []dto.TemplateDto)
	countChan := make(chan int64)
	errChan := make(chan error)

	// get items
	pg := database.PaginationOps{
		Limit: *ops.Limit,
		Skip:  *ops.Limit * ops.Page,
	}
	go func() {
		items, err := ts.store.GetByCreator(ctx, creator, &pg)

		if err != nil {
			errChan <- err
			return
		}

		templates := make([]dto.TemplateDto, 0, len(items))

		for _, i := range items {
			d := *i
			templates = append(templates, mapper.MapTemplateModelToDto(ctx, d))
		}
		itemChan <- templates
	}()

	go func() {
		count, err := ts.store.GetTotalCount(ctx, creator)

		if err != nil {
			errChan <- err
			return
		}

		countChan <- count
	}()

	var templates []dto.TemplateDto
	var count int64

	for i := 0; i < 2; i++ {
		select {
		case i := <-itemChan:
			templates = i
		case c := <-countChan:
			count = c
		case err := <-errChan:
			ts.logger.Error("failed to retrieve template for client", slog.String("client", creator), slog.String("error", err.Error()))
			return dto.PaginatedResponse[dto.TemplateDto]{}, fmt.Errorf("failed to retrieve template for client=%s", creator)
		}
	}

	return dto.PaginatedResponse[dto.TemplateDto]{
		Page:  ops.Page,
		Limit: *ops.Limit,
		Data:  templates,
		Total: count,
	}, nil
}

// Get template
func (ts *TemplateSvc) GetTemplate(ctx context.Context, creator, name string) (*dto.TemplateDto, error) {
	ts.logger.Info("getting template", slog.String("creator", creator), slog.String("name", name))
	i, err := ts.store.GetByKey(ctx, creator, name)

	if err != nil {
		ts.logger.Error("failed to get template", slog.Group("error", err.Error(), "creator", creator, "name", name))

		return nil, fmt.Errorf("failed to get template by key creator=%s name=%s", creator, name)
	}

	if i == nil {
		return nil, ErrTemplateNotFound
	}
	ts.logger.Info("template", slog.Any("template", i))
	d := mapper.MapTemplateModelToDto(ctx, *i)
	return &d, nil
}

// DeleteTemplate
func (ts *TemplateSvc) DeleteTemplate(ctx context.Context, creator string, name string) error {

	err := ts.store.Delete(ctx, name, creator)

	if err != nil {
		ts.logger.Error("failed to delete template", slog.String("error", err.Error()))
		return fmt.Errorf("failed to delete template creator='%s', name='%s'", creator, name)
	}
	ts.logger.Info("successfully delete template", slog.String("name", name), slog.String("creator", creator))
	return nil
}
