.PHONY: build clean deploy gomodgen serve
profile=personal
build: 
	export GO111MODULE=on
	env GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o ./bin/control-tower/bootstrap ./control-tower/cmd/app
	zip -j ./bin/control-tower/control-tower.zip ./bin/control-tower/bootstrap
	env GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o ./bin/schedule-handler/bootstrap ./control-tower/cmd/schedule-handler
	zip -j ./bin/schedule-handler/schedule-handler.zip ./bin/schedule-handler/bootstrap
	env GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o ./bin/ws-connection-handler/bootstrap ./control-tower/cmd/ws-connection-handler
	zip -j ./bin/ws-connection-handler/ws-connection-handler.zip ./bin/ws-connection-handler/bootstrap
	env GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o ./bin/ws-default-handler/bootstrap ./control-tower/cmd/ws-default-handler
	zip -j ./bin/ws-default-handler/ws-default-handler.zip ./bin/ws-default-handler/bootstrap
	env GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o ./bin/authorizer/bootstrap ./control-tower/cmd/authorizer
	zip -j ./bin/authorizer/authorizer.zip ./bin/authorizer/bootstrap
clean:
	rm -rf ./bin ./vendor go.sum

deploy: clean build
	
	APP_ID=control-tower sls deploy --verbose --aws-profile $(profile)

gomodgen:
	chmod u+x gomod.sh
	./gomod.sh

serve: 
	STAGE=local
	PORT=3000
	go run -tags=local ./control-tower/cmd/app/
upload:
	source "C:\Users\Javier Perez\OneDrive\Desktop\eliEmail\environment.sh"
	go run ./cliApp/cmd --creator pratoelis@gmail.com --path "C:\Users\Javier Perez\OneDrive\Desktop\booksy_automation\customers.json"

docs:
	~/go/bin/swag init -d ./control-tower/internal/controller -g notification.go
	~/go/bin/swag init -d ./control-tower/internal/controller -g client.go