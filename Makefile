
build:
	rm -rf bin/*
	go build -o bin/calico-route-sync-linux cmd/sync/main.go

docker-build:
	docker build -t