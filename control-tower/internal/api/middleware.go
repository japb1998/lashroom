package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func getUserEmailFromToken(tokenString string) (string, error) {
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
		routerLogger.Println("Extracting User")

		token, ok := c.Request.Header["Authorization"]

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}
		email, err := getUserEmailFromToken(token[0])

		if err != nil {
			routerLogger.Println(err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}
		c.Set("email", email)

		c.Next()
	}
}
