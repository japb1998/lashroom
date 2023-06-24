package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	data := url.Values{}

	data.Set("Authorization", base64.StdEncoding.Strict().EncodeToString([]byte(clientID+":"+clientSecret)))
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("scope", "https://938f97v570.execute-api.us-east-1.amazonaws.com/read https://938f97v570.execute-api.us-east-1.amazonaws.com/write")
	url := "https://lashroombyeli.auth.us-east-1.amazoncognito.com/oauth2/token"

	res, err := http.Post(url, "application/x-www-form-urlencoded", bytes.NewBuffer([]byte(data.Encode())))

	if err != nil {
		log.Fatal(err.Error())
		return
	}

	log.Println(res.StatusCode)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err.Error())
	}
	var response map[string]any
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err.Error())
	}
	token := fmt.Sprintf("%s %s", response["token_type"], response["access_token"])
	fmt.Println(token)
	client := http.Client{}
	post, err := http.NewRequest("POST", url, nil)
	post.Header.Add("Authorization", token)
	if err != nil {
		log.Fatal(err.Error())
	}
	// get, err := http.NewRequest("POST", "https://938f97v570.execute-api.us-east-1.amazonaws.com/dev/schedule", nil)

	// get.Header.Add("Authorization", token)

	res, err = client.Do(post)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf(res.Status)
	body, err = io.ReadAll(res.Body)
	var schedules any
	if err != nil {
		log.Fatal(err.Error())
	}
	json.Unmarshal(body, &schedules)
	fmt.Println(schedules)
	os.Exit(0)
}
