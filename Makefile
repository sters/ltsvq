
GITHUB_TOKEN ?= fake

.PHONY: test
test:
	@go test -race --tags json1 ./...

.PHONY: cover
cover:
	@go test -cover -race --tags json1 ./...

.PHONY: build
build:
	CGO_ENABLED=1 go build ./...

.PHONY: release-with-docker
release-with-docker:
	docker build -t releaser:latest .
	docker run --rm --privileged \
		-v $(PWD):/go/src/github.com/sters/ltsvq \
		-w /go/src/github.com/sters/ltsvq \
		-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
		releaser:latest goreleaser release --rm-dist
