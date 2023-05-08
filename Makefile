.PHONY: bin

# Mincraft Stop Start Lambda function helpers
compile-mcss:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/mcss-linux-amd64/main cmd/mcss/main.go

zip-mcss: compile-mcss
	zip bin/mcss-linux-amd64/main.zip bin/mcss-linux-amd64/main

deploy-mcss: zip-mcss
	aws lambda update-function-code --function-name mc_operations --zip-file fileb://bin/mcss-linux-amd64/main.zip

# Minecraft BOT Steve helpers
steve:
	go run cmd/steve/main.go -t $(BOT_TOKEN)
