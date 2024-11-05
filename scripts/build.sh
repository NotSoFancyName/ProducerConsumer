#!/bin/bash

GO_VERSION=$(go version | awk '{print $3}')

go build -ldflags "-X 'main.version=$GO_VERSION'" -o ./build/producer ./system/producer/cmd/producer/main.go
go build -ldflags "-X 'main.version=$GO_VERSION'" -o ./build/consumer ./system/consumer/cmd/consumer/main.go
