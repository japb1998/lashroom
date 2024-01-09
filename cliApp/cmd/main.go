package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/japb1998/lashroom/cliApp/internals/booksy"
	"github.com/japb1998/lashroom/cliApp/internals/utils"
	"github.com/japb1998/lashroom/scheduleEmail/internal/client"
	"github.com/japb1998/lashroom/shared/internal/database"
)

var wg sync.WaitGroup

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
		wg.Add(1)
		go func(customer booksy.BooksyClient) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Println("recovered from error:", r)
				}
			}()

			clientPhone := utils.ExtractNumbers(customer.CellPhone)

			newClient := client.ClientDto{
				CreatedBy:  *createdBy,
				Phone:      clientPhone,
				Email:      &customer.Email,
				ClientName: strings.Trim(fmt.Sprintf("%s %s", customer.FirstName, customer.LastName), ""),
			}

			clientList, err := clientService.Store.GetClientWithFilters(*createdBy, newClient)

			if err != nil {
				fmt.Print(err.Error())
				return
			}

			if len(clientList) > 0 {
				log.Printf("Client name: %s, phone number: %s, exists", newClient.ClientName, *newClient.Phone)
				return
			}

			newClient.Phone = utils.ExtractNumbers(*newClient.Phone)

			c, err := clientService.CreateClient(newClient)

			if err != nil {
				fmt.Print(err.Error())
				return
			}

			log.Println("New Customer:", c.ClientName)
		}(customer)
	}
	wg.Wait()
}
