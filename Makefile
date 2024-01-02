.PHONY: build clean deploy gomodgen serve
profile=personal
list=app schedule-handler ws-connection-handler ws-default-handler authorizer ws-ping-handler
build:
	export GO111MODULE=on 
	for i in $(list); do \
		env GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o "./bin/$$i/bootstrap" "./control-tower/cmd/$$i" \
		&& \
		zip -j "./bin/$$i/$$i.zip" "./bin/$$i/bootstrap"; \
	done 
	
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
	source "C:Users\Javier Perez\OneDrive\Desktop\eliEmail\environment.sh"
	go run ./cliApp/cmd --creator pratoelis@gmail.com --path "C:\Users\Javier Perez\OneDrive\Desktop\booksy_automation\customers.json"

docs:
	~/go/bin/swag init -d ./control-tower/internal/controller -g notification.go
	~/go/bin/swag init -d ./control-tower/internal/controller -g client.go