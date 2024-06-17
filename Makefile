default: test

ci: depsdev test

test: cert
	go test ./... -coverprofile=coverage.out -covermode=count

lint:
	golangci-lint run ./...

depsdev:
	go install github.com/Songmu/ghch/cmd/ghch@latest
	go install github.com/Songmu/gocredits/cmd/gocredits@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
	go get buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate/priv

testclient: depsdev
	mkdir -p testdata/routeguide
	mkdir -p testdata/hello
	cd testdata/ && protoc --go_out=routeguide --go_opt=paths=source_relative --go-grpc_out=routeguide --go-grpc_opt=paths=source_relative route_guide.proto
	cd testdata/ && protoc --go_out=hello --go_opt=paths=source_relative --go-grpc_out=hello --go-grpc_opt=paths=source_relative hello.proto
	cd testdata/bsr/protobuf && buf mod update && buf generate

cert:
	rm -f testdata/*.pem testdata/*.srl
	openssl req -x509 -newkey rsa:4096 -days 365 -nodes -sha256 -keyout testdata/cakey.pem -out testdata/cacert.pem -subj "/C=UK/ST=Test State/L=Test Location/O=Test Org/OU=Test Unit/CN=*.example.com/emailAddress=k1low@pepabo.com"
	openssl req -newkey rsa:4096 -nodes -keyout testdata/key.pem -out testdata/csr.pem -subj "/C=JP/ST=Test State/L=Test Location/O=Test Org/OU=Test Unit/CN=*.example.com/emailAddress=k1low@pepabo.com"
	openssl x509 -req -sha256 -in testdata/csr.pem -days 60 -CA testdata/cacert.pem -CAkey testdata/cakey.pem -CAcreateserial -out testdata/cert.pem -extfile testdata/openssl.cnf
	openssl verify -CAfile testdata/cacert.pem testdata/cert.pem

prerelease:
	git pull origin main --tag
	go mod tidy
	ghch -w -N ${VER}
	gocredits -w .
	git add CHANGELOG.md CREDITS go.mod go.sum
	git commit -m'Bump up version number'
	git tag ${VER}

prerelease_for_tagpr: depsdev
	go mod tidy
	gocredits -w .
	git add CHANGELOG.md CREDITS go.mod go.sum

release:
	git push origin main --tag

.PHONY: default test
