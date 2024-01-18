package mapper

import (
	"context"

	"github.com/japb1998/control-tower/internal/dto"
	"github.com/japb1998/control-tower/internal/model"
)

// maps template variables dto to model in order to store it in the DB.
func TemplateVarsModelToDto(ctx context.Context, m model.VariableType) dto.VariableTypeDto {
	return dto.VariableTypeDto{
		KeyType:   m.KeyType,
		Key:       m.Key,
		ValueType: m.ValueType,
		Value:     m.Value,
	}
}

func MapVarsSliceToDto(ctx context.Context, m []model.VariableType) []dto.VariableTypeDto {
	slc := make([]dto.VariableTypeDto, 0, len(m))

	for _, v := range m {
		slc = append(slc, TemplateVarsModelToDto(ctx, v))
	}

	return slc
}

// maps template vars dto to model type
func TemplateVarsDtoToModel(ctx context.Context, d dto.VariableTypeDto) model.VariableType {
	return model.VariableType{
		KeyType:   d.KeyType,
		Key:       d.Key,
		ValueType: d.ValueType,
		Value:     d.Value,
	}
}

func MapVarsSliceToModel(ctx context.Context, ds []dto.VariableTypeDto) []model.VariableType {
	slc := make([]model.VariableType, 0, len(ds))

	for _, v := range ds {
		slc = append(slc, TemplateVarsDtoToModel(ctx, v))
	}

	return slc
}

// maps template dto to model in order to store it in the db.
func MapTemplateDtoToModel(ctx context.Context, d dto.TemplateDto) model.TemplateItem {
	return model.TemplateItem{
		Name:         d.Name,
		TemplateId:   d.TemplateId,
		Html:         d.Html,
		Variables:    MapVarsSliceToModel(ctx, d.Variables),
		CreatedBy:    d.CreatedBy,
		TemplateType: d.TemplateType,
	}
}

// maps template dto to model in order to store it in the db.
func MapTemplateModelToDto(ctx context.Context, d model.TemplateItem) dto.TemplateDto {
	return dto.TemplateDto{
		Name:         d.Name,
		TemplateId:   d.TemplateId,
		Html:         d.Html,
		Variables:    MapVarsSliceToDto(ctx, d.Variables),
		CreatedBy:    d.CreatedBy,
		TemplateType: d.TemplateType,
	}
}

// maps dto.UpdateTemplateDto to  model.UpdateTemplate
func MapTemplateUpdateToModel(ctx context.Context, d dto.UpdateTemplateDto) model.UpdateTemplate {
	return model.UpdateTemplate{
		Variables:  MapVarsSliceToModel(ctx, d.Variables),
		Html:       d.Html,
		TemplateId: d.TemplateId,
	}
}

// maps createTemplateDto to templateItem
func MapCreateTemplateDtoToModel(ctx context.Context, creator string, d dto.CreateTemplateDto) model.TemplateItem {
	return model.TemplateItem{
		Name:         d.Name,
		CreatedBy:    creator,
		TemplateId:   d.TemplateId,
		Html:         d.Html,
		TemplateType: d.TemplateType,
		Variables:    MapVarsSliceToModel(ctx, d.Variables),
	}
}
