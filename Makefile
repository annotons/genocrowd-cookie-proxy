TARGET := genocrowd-cookie-proxy

NAMESPACE	:=github.com/annotons/genocrowd-cookie-proxy
WORKSPACE	:=$(GOPATH)/src/$(NAMESPACE)
GO_SOURCES	:=$(wildcard *.go)
GO_PACKAGES	:=$(dir $(GO_SOURCES))
VERSION	:=$(shell git describe --tags --always)
GO_FLAGS	:=-ldflags="-X main.version=$(VERSION) -X main.builddate=$(shell date --iso-8601=seconds --utc)"
DEP_ARGS	:=-v

all: $(TARGET)
	echo $(WORKSPACE)

test: setup.lock
	@cd $(WORKSPACE)\
		&& go test $(addprefix $(NAMESPACE)/,$(GO_PACKAGES))

setup.lock: $(WORKSPACE) vendor
	@echo $(VERSION) > setup.lock

fmt: $(GO_SOURCES)
	gofmt -w $(GO_SOURCES)
	goimports -w $(GO_SOURCES)

check: vet lint

vet: $(GO_SOURCES)
	go vet $(NAMESPACE)/

lint: $(GO_SOURCES)
	golint $(NAMESPACE)/

dep: $(WORKSPACE)
	@cd $(WORKSPACE) && dep $(ARGS)

vendor: Gopkg.toml Gopkg.lock
	@cd $(WORKSPACE) && dep ensure $(DEP_ARGS)
	@touch $@


$(TARGET): $(SRC) setup.lock
	go build $(GO_FLAGS) -o $@

clean:
	$(RM) $(TARGET)

build:
	rm -rf dist/
	mkdir dist
	CGO_ENABLED=0 gox $(GO_FLAGS) -os="linux" -output "dist/{{.Dir}}_{{.OS}}_{{.Arch}}"

release: build
	ghr -u annotons -r genocrowd-cookie-proxy -replace $(VERSION) dist/

.PHONY: clean lint fmt vet test clean release
