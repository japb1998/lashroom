package dto

type CreateTemplateDto struct {
	Name         string            `json:"name" binding:"required,noSpaces,min=2"`
	TemplateId   *string           `json:"templateId,omitempty" binding:"required,min=2"`
	Html         *string           `json:"html" binding:"omitempty,min=2"`
	Variables    []VariableTypeDto `json:"variables" binding:"omitempty"`
	TemplateType int8              `json:"templateType" binding:"min=0"` // directly linked to delivery methods in Notifications.
}

type TemplateDto struct {
	Name         string            `json:"name"`
	TemplateId   *string           `json:"templateId,omitempty"`
	Html         *string           `json:"html,omitempty"`
	Variables    []VariableTypeDto `json:"variables"`
	CreatedBy    string            `json:"createdBy"`
	TemplateType int8              `json:"templateType"` // directly linked to delivery methods in Notifications.
}

type UpdateTemplateDto struct {
	Variables  []VariableTypeDto `json:"variables"`
	TemplateId *string           `json:"templateId,omitempty"`
	Html       *string           `json:"html,omitempty"`
}

type VariableTypeDto struct {
	KeyType   string `json:"keyType" binding:"required"`
	Key       string `json:"key" binding:"required"`
	ValueType string `json:"valueType" binding:"required"`
	Value     string `json:"value" binding:"required"`
}
