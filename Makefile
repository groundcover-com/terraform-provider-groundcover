default: fmt lint install generate

build:
	go build -v -o dist/ ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

# Assert every TestAcc test is selected by an acceptance-test matrix group. Needs no
# credentials — it only lists test names.
check-acceptance-coverage:
	./scripts/check-acceptance-coverage.sh .github/workflows/test.yml
	./scripts/check-acceptance-coverage.sh .github/workflows/release.yml

.PHONY: fmt lint test testacc build install generate check-acceptance-coverage
