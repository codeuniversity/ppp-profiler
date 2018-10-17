install:
	curl https://glide.sh/get | sh

dep:
	glide install

test:
	go test ./... -timeout 10s

run:
	go run main/main.go
