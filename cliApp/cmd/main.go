package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/japb1998/lashroom/cliApp/internals/booksy"
	"github.com/japb1998/lashroom/cliApp/internals/utils"
	"github.com/japb1998/lashroom/scheduleEmail/pkg/client"
	"github.com/japb1998/lashroom/shared/pkg/database"
)

func main() {

	fullPath := flag.String("path", "", "Full Path to json file to upload")
	createdBy := flag.String("creator", "", "Email to attach the clients to")
	flag.Parse()
	fmt.Println("path", *fullPath, "creator", *createdBy)

	if len(*fullPath) < 1 || len(*createdBy) < 1 {
		panic("Full path and creator are required")
	}

	data, err := os.ReadFile(*fullPath)

	if err != nil {
		log.Fatal(err.Error())
		return
	}

	var customers []booksy.BooksyClient
	if err := json.Unmarshal(data, &customers); err != nil {
		panic("Unable to parse clients from provided json")
	}

	store := database.NewClientRepository()
	clientService := client.NewClientService(store)

	for _, customer := range customers {

		clientPhone := utils.ExtractNumbers(customer.CellPhone)

		newClient := client.ClientDto{
			CreatedBy:  *createdBy,
			Phone:      clientPhone,
			Email:      &customer.Email,
			ClientName: strings.Trim(fmt.Sprintf("%s %s", customer.FirstName, customer.LastName), ""),
		}

		newClient.Phone = utils.ExtractNumbers(*newClient.Phone)

		c, err := clientService.CreateClient(newClient)

		if err != nil {
			fmt.Print(err.Error())
		}

		log.Println("New Customer:", c.ClientName)
	}

}
