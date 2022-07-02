default: test

ci: depsdev test

test:
	go test ./... -coverprofile=coverage.out -covermode=count

lint:
	golangci-lint run ./...

depsdev:
	go install github.com/Songmu/ghch/cmd/ghch@latest
	go install github.com/Songmu/gocredits/cmd/gocredits@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

testclient:
	mkdir -p testdata/routeguide
	cd testdata/ && protoc --go_out=routeguide --go_opt=paths=source_relative --go-grpc_out=routeguide --go-grpc_opt=paths=source_relative route_guide.proto

prerelease:
	git pull origin main --tag
	go mod tidy
	ghch -w -N ${VER}
	gocredits -w .
	git add CHANGELOG.md CREDITS go.mod go.sum
	git commit -m'Bump up version number'
	git tag ${VER}

release:
	git push origin main --tag

.PHONY: default test
