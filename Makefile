.PHONY: build clean deploy gomodgen serve

build: 
	export GO111MODULE=on
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ./bin/scheduleEmail ./scheduleEmail/cmd
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ./bin/scheduleCheck ./scheduleCheck/cmd
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ./bin/queueHandler ./clientQueue/cmd/main.go
	go build -o ./bin/cliApp ./cliApp/cmd
clean:
	rm -rf ./bin ./vendor go.sum

deploy: clean build
	sls deploy --verbose

gomodgen:
	chmod u+x gomod.sh
	./gomod.sh

serve: 
	source "C:\Users\Javier Perez\OneDrive\Desktop\eliEmail\environment.sh"
	STAGE=local
	go run ./scheduleEmail/cmd
upload:
	source "C:\Users\Javier Perez\OneDrive\Desktop\eliEmail\environment.sh"
	go run ./cliApp/cmd --creator pratoelis@gmail.com --path "C:\Users\Javier Perez\OneDrive\Desktop\booksy_automation\customers.json"