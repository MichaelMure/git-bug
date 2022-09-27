all: build

GIT_COMMIT:=$(shell git rev-list -1 HEAD)
GIT_LAST_TAG:=$(shell git describe --abbrev=0 --tags)
GIT_EXACT_TAG:=$(shell git name-rev --name-only --tags HEAD)
UNAME_S := $(shell uname -s)
XARGS:=xargs -r
ifeq ($(UNAME_S),Darwin)
    XARGS:=xargs
endif

COMMANDS_PATH:=github.com/MichaelMure/git-bug/commands
LDFLAGS:=-X ${COMMANDS_PATH}.GitCommit=${GIT_COMMIT} \
	-X ${COMMANDS_PATH}.GitLastTag=${GIT_LAST_TAG} \
	-X ${COMMANDS_PATH}.GitExactTag=${GIT_EXACT_TAG}

build:
	go generate
	go build -ldflags "$(LDFLAGS)" .

# produce a build debugger friendly
debug-build:
	go generate
	go build -ldflags "$(LDFLAGS)" -gcflags=all="-N -l" .

install:
	go generate
	go install -ldflags "$(LDFLAGS)" .

releases:
	go generate
	gox -ldflags "$(LDFLAGS)" -output "dist/{{.Dir}}_{{.OS}}_{{.Arch}}"

secure: secure-practices secure-vulnerabilities

secure-practices:
	go install github.com/praetorian-inc/gokart
	gokart scan

secure-vulnerabilities:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./... 

legal: build
	go install github.com/uw-labs/lichen@latest
	lichen --config=.lichen.yaml ./git-bug

test:
	go test -v -bench=. ./...

pack-webui:
	npm run --prefix webui build
	go run webui/pack_webui.go

# produce a build that will fetch the web UI from the filesystem instead of from the binary
debug-webui:
	go build -ldflags "$(LDFLAGS)" -tags=debugwebui

clean-local-bugs:
	git for-each-ref refs/bugs/ | cut -f 2 | $(XARGS) -n 1 git update-ref -d
	git for-each-ref refs/remotes/origin/bugs/ | cut -f 2 | $(XARGS) -n 1 git update-ref -d
	rm -f .git/git-bug/bug-cache

clean-remote-bugs:
	git ls-remote origin "refs/bugs/*" | cut -f 2 | $(XARGS) git push origin -d

clean-local-identities:
	git for-each-ref refs/identities/ | cut -f 2 | $(XARGS) -n 1 git update-ref -d
	git for-each-ref refs/remotes/origin/identities/ | cut -f 2 | $(XARGS) -n 1 git update-ref -d
	rm -f .git/git-bug/identity-cache

clean-remote-identities:
	git ls-remote origin "refs/identities/*" | cut -f 2 | $(XARGS) git push origin -d

.PHONY: build install releases test pack-webui debug-webui clean-local-bugs clean-remote-bugs
.PHONY: secure secure-vulnerabilities secure-practices
