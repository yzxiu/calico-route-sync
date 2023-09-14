
build:
	rm -rf bin/*
	go build -ldflags="-w -s" -o bin/calico-route-sync-linux cmd/sync/main.go

docker-build:
	docker build -t