package model

// TemplateItem templates are part of every notification mode.
type TemplateItem struct {
	CreatedBy    string         `json:"createdBy"`
	Name         string         `json:"name"`
	Variables    []VariableType `json:"variables"`
	TemplateId   *string        `json:"templateId,omitempty"`
	Html         *string        `json:"html,omitempty"`
	TemplateType int8           `json:"templateType"` // directly linked to delivery methods in Notifications.
}

type UpdateTemplate struct {
	Variables  []VariableType `json:"variables"`
	TemplateId *string        `json:"templateId,omitempty"`
	Html       *string        `json:"html,omitempty"`
}

type VariableType struct {
	KeyType   string `json:"keyType"`
	Key       string `json:"key"`
	ValueType string `json:"valueType"`
	Value     string `json:"value"`
}
