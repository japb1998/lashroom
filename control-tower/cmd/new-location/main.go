package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/model"
	"github.com/japb1998/control-tower/pkg/awssess"
	"github.com/japb1998/control-tower/pkg/sms"
	"github.com/joho/godotenv"
)

var logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}).WithAttrs([]slog.Attr{slog.String("app", "new-location")})
var logger = slog.New(logHandler)

func main() {

	creator := flag.String("creator", "", "[required] user email that created the clients")
	flag.Parse()

	if *creator == "" {
		flag.PrintDefaults()
		return
	}

	if os.Getenv("stage") == "local" {
		cwd, err := os.Getwd()

		if err != nil {
			log.Fatalf("Error getting current working directory: %s", err)
		}

		p := path.Join(cwd, "./.env")
		err = godotenv.Load(p)

		if err != nil {
			log.Fatalf("Error loading env vars: %s", err)
		}
	}
	sess := awssess.MustGetSession()
	clientSvc := database.NewClientRepo(sess)

	svcID := os.Getenv("TWILIO_SERVICE_ID")

	if svcID == "" {
		log.Fatal("TWILIO_SERVICE_ID is required")
	}
	smsService := sms.MusInitMsgSvc(svcID)

	clients, err := clientSvc.GetClientsByCreator(*creator)

	if err != nil {
		log.Fatalf("Error getting clients: %s", err)
	}

	cc := make(chan os.Signal)
	signal.Notify(cc, os.Interrupt, syscall.SIGTERM)
	errChan := make(chan error, len(clients))
	cachedClients := sync.Map{}
	clientsQueue := make(chan model.ClientItem, len(clients))
	var wg sync.WaitGroup

	waitChan := make(chan bool, 1)
	templateVariables := map[string]string{
		"1": "8751 commodity circle suite 12/14, Orlando FL 32819",
		"2": "+1 (863) 521-3491",
	}

	v, err := json.Marshal(templateVariables)

	if err != nil {
		log.Fatalf("Error marshaling template variables: %s", err)
	}

	for _, client := range clients {
		clientsQueue <- client
	}
	close(clientsQueue)

	for c := range clientsQueue {
		if _, ok := cachedClients.Load(c.Phone); ok {
			logger.Debug("skipping client", slog.String("phone", c.Phone))
			continue
		}
		// mark before sending
		cachedClients.Store(c.Phone, true)

		wg.Add(1)
		go func(client model.ClientItem) {
			defer wg.Done()

			logger.Info("queuing message to", slog.String("phone", client.Phone))
			msg := &sms.Msg{
				To:                client.Phone,
				TemplateVariables: v,
				TemplateId:        os.Getenv("ADDR_TEMPLATE_ID"),
			}
			err := smsService.SendMessage(msg)
			if err != nil {
				errChan <- fmt.Errorf("client %s, failed with error=%s", client.Phone, err.Error())
				return
			}

		}(c)
	}

	go func() {
		wg.Wait()
		log.Println("DONE")
		close(errChan)
	}()
	go func() {

		for range errChan {
			select {
			case err := <-errChan:
				if err != nil {
					log.Printf("Error sending sms: %s", err)
				}
			case <-cc:
				log.Println("CANCELLED")
				waitChan <- true
				return
			}
		}
		waitChan <- true
	}()
	<-waitChan
}
