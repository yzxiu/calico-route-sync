
build:
	go build -o bin/calico-route-sync cmd/sync/main.go

docker-build:
	docker build -t