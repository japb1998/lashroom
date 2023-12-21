package credentials

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type CredentialsManager struct {
	manager *secretsmanager.SecretsManager
}

var (
	ErrSecretNotFound = errors.New("Secret not found")
)

func NewCredentialsManager(sess *session.Session) *CredentialsManager {
	sm := secretsmanager.New(sess)
	return &CredentialsManager{
		manager: sm,
	}
}

func (cm *CredentialsManager) GetSecret(secretArn string) (*string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     &secretArn,
		VersionStage: aws.String("AWSCURRENT"),
	}
	out, err := cm.manager.GetSecretValue(input)
	var t *secretsmanager.ResourceNotFoundException
	if errors.As(err, &t) {
		return nil, ErrSecretNotFound
	}
	return out.SecretString, err
}
