package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func getUserEmailFromToken(ctx context.Context, tokenString string) (string, error) {
	_, span := tracer.Start(ctx, "getUserEmailFromToken")
	defer span.End()
	claims := jwt.MapClaims{}
	tokenSlice := strings.Split(tokenString, " ")
	if len(tokenSlice) < 2 {
		return "", fmt.Errorf("Bearer token has incorrect format")
	}
	jwt.ParseWithClaims(tokenSlice[1], claims, func(t *jwt.Token) (interface{}, error) {
		return nil, nil
	})
	if email, ok := claims["email"]; !ok {

		return "", errors.New("error while getting user email from token")

	} else if emailString, ok := email.(string); !ok {

		return "", errors.New("email is not a string")
	} else {
		return emailString, nil
	}
}

func currentUserMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, userSpan := tracer.Start(c.Request.Context(), "currentUserMiddleWare")

		routerLogger.Println("Extracting User")

		token, ok := c.Request.Header["Authorization"]

		if !ok {
			userSpan.SetStatus(codes.Error, "Unauthorized")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}
		email, err := getUserEmailFromToken(ctx, token[0])

		if err != nil {
			routerLogger.Println(err)
			userSpan.SetStatus(codes.Error, "Unauthorized")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}
		userSpan.SetAttributes(attribute.String("email", email))
		c.Set("email", email)
		userSpan.End()
		c.Next()
	}
}

func init() {
	tracer = otel.Tracer(ScopeName)
}
