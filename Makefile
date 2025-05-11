.PHONY: proto build run docker-build docker-run test clean

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
		proto/event.proto proto/ticket.proto proto/notification.proto

build:
	go build -o bin/event-service ./event-service/cmd/server
	go build -o bin/ticket-service ./ticket-service/cmd/server
	go build -o bin/notification-service ./notification-service/cmd/server

run: build
	./bin/event-service & \
	./bin/ticket-service & \
	./bin/notification-service

docker-build:
	docker-compose build

docker-run:
	docker-compose up

test:
	go test ./... -v

clean:
	rm -rf bin/
	go clean

init-db:
	mysql -h localhost -u root -ppassword -e "CREATE DATABASE IF NOT EXISTS events;"
	mysql -h localhost -u root -ppassword -e "CREATE DATABASE IF NOT EXISTS notifications;"

stop:
	docker-compose down

logs:
	docker-compose logs -f