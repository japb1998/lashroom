package websocket

import "fmt"

func getEmailFromContext(authorizerCtx interface{}) (string, error) {
	switch v := authorizerCtx.(type) {
	case map[string]interface{}:
		if email, ok := v["email"]; ok {
			return email.(string), nil
		}
		return "", fmt.Errorf("email not present in context")
	default:
		fmt.Printf("authorizer: %v \n", authorizerCtx)
		return "", fmt.Errorf("invalid email passed from authorizer")
	}
}
