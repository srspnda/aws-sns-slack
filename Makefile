USER = "srspnda"
NAME = "aws-sns-slack"
VERSION = "0.1.0"

GOXOS = "linux darwin windows"
GOXOUT = "build/{{.Dir}}_$(VERSION)_{{.OS}}_{{.Arch}}/$(NAME)"

all: build

build:
	@mkdir -p bin/
	go build -o bin/$(NAME)
	cp bin/$(NAME) $(GOPATH)/bin

xcompile:
	rm -rf build/
	@mkdir -p build
	gox -os=$(GOXOS) -output=$(GOXOUT)

package: xcompile
	$(eval FILES := $(shell ls build))
	@mkdir -p build/tgz
	for f in $(FILES); do \
		(cd $(shell pwd)/build && tar -czvf tgz/$$f.tar.gz $$f); \
		echo $$f; \
	done

release: package
	go get github.com/aktau/github-release
	github-release release \
		--user $(USER) \
		--repo $(NAME) \
		--tag v$(VERSION)
	$(eval FILES := $(shell ls build/tgz))
	for f in $(FILES); do \
		(cd $(shell pwd)/build/tgz && github-release upload \
			--user $(USER) \
			--repo $(NAME) \
			--tag v$(VERSION) \
			--name $$f \
			--file $$f); \
		echo $$f; \
	done

.PHONY: all build xcompile package release
