package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"github.com/go-playground/validator/v10"
	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/dto"
	"github.com/japb1998/control-tower/internal/mapper"
	"github.com/japb1998/control-tower/pkg/awssess"
	"github.com/joho/godotenv"
)

var store *database.ClientRepository

func main() {
	pf, err := os.Create("./go-pp.prof")
	if err != nil {
		log.Fatal(err)
	}

	/* to be removed */
	pprof.StartCPUProfile(pf)
	defer pf.Close()
	defer pprof.StopCPUProfile()

	runtime.GOMAXPROCS(runtime.NumCPU())

	defer func() {
		if rec := recover(); rec != nil {
			fmt.Println("Recovered from %v", rec)
		}
	}()
	cc := make(chan os.Signal)
	signal.Notify(cc, os.Interrupt, syscall.SIGTERM)
	errChan := make(chan error)

	var f string
	flag.StringVar(&f, "file", "", "file were the booksy clients are located")

	flag.Parse()

	fmt.Println("file", f)
	if _, err := os.Stat(f); err != nil {
		if errors.Is(err, fs.ErrExist) {
			log.Fatalf("failed to open file error='%s'", err.Error())
		}
		log.Fatalf("failed to open file error='%s'", err.Error())
	}

	d, err := os.Open(f)
	defer d.Close()
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			log.Fatalf("failed to open file error='%s'", err.Error())
		}
		log.Fatalf("failed to open file error='%s'", err.Error())
	}

	var clients []dto.BooksyUserDto

	err = json.NewDecoder(d).Decode(&clients)

	if err != nil {
		log.Fatalf("failed unmarshall error='%s'", err.Error())
	}
	creator := "pratoelis@gmail.com"

	for i, client := range clients {
		fmt.Println("running client", i)
		c := mapper.BooksyUserToItem(creator, client)

		validate := validator.New(validator.WithRequiredStructEnabled())

		err := validate.Struct(c)

		if err != nil {

			for _, ve := range err.(validator.ValidationErrors) {
				fmt.Printf("%s validation: %s failed. value='%s', param='%s'\n", ve.Namespace(), ve.Tag(), ve.Value(), ve.Param())
			}
			log.Fatal(c, err)

		}

		filters := database.PatchClientItem{
			Email: client.Email,
		}
		ops := &database.PaginationOps{
			Limit: 1,
			Skip:  0,
		}

		go func() {

			results, err := store.GetClientWithFilters(creator, filters, ops)

			if err != nil {
				fmt.Println(err)
				errChan <- err
				return
			}

			if results != nil && len(results) > 0 {
				fmt.Printf("client %v. already exists\n", c)
				errChan <- nil
				return
			}
			_, err = store.CreateClient(c)

			if err != nil {
				errChan <- fmt.Errorf("failed to create client error='%w'", err)
				return
			} else {
				fmt.Printf("successfully created client %v\n", c)
				errChan <- nil
				return
			}

		}()

	}
	fmt.Println("out of the loop")

	for range clients {
		select {
		case e := <-errChan:
			{
				if e != nil {
					fmt.Println(err)
				}
			}
		case <-cc:
			fmt.Println("CANCELLED")
			os.Exit(1)

		}
	}
}

func init() {
	if os.Getenv("STAGE") == "local" {

		fmt.Println("init local")
		err := godotenv.Load(".env", "./control-tower/.env")
		if err != nil {
			log.Fatalf("Error loading env vars: %s", err)
		}
	}

	sess := awssess.MustGetSession()

	store = database.NewClientRepo(sess)

}
