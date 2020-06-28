  
GO ?= go
TESTFOLDER := $(shell $(GO) list ./...)

.PHONY: test
test:
	echo "mode: atomic" > coverage.out
	for d in $(TESTFOLDER); do \
		$(GO) test -v -covermode=atomic -race -coverprofile=profile.out $$d; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done

release:
	mkdir releases;
	for GOOS in darwin linux windows; do \
			for GOARCH in 386 amd64; do \
					$(GO) build -v -o releases/medusa-$$GOOS-$$GOARCH; \
			done \
	done \