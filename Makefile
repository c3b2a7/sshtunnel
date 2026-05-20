NAME=sshtunnel
BINDIR=bin
VERSION=$(shell git describe --tags --always || echo "unknown version")
BUILDTIME=$(shell date -u)
GOBUILD=CGO_ENABLED=0 go build -trimpath -ldflags ' \
		-X "main.Version=$(VERSION)" \
		-X "main.BuildTime=$(BUILDTIME)" \
		-w -s -buildid='

all: linux-amd64 linux-arm64 freebsd-amd64 freebsd-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-386

$(BINDIR):
	mkdir -p $(BINDIR)

linux-amd64: $(BINDIR)
	GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

linux-arm64: $(BINDIR)
	GOARCH=arm64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

freebsd-amd64: $(BINDIR)
	GOARCH=amd64 GOOS=freebsd $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

freebsd-arm64: $(BINDIR)
	GOARCH=arm64 GOOS=freebsd $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

darwin-amd64: $(BINDIR)
	GOARCH=amd64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

darwin-arm64: $(BINDIR)
	GOARCH=arm64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

windows-amd64: $(BINDIR)
	GOARCH=amd64 GOOS=windows $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.exe

windows-386: $(BINDIR)
	GOARCH=386 GOOS=windows $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.exe

releases: linux-amd64 linux-arm64 freebsd-amd64 freebsd-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-386
	chmod +x $(BINDIR)/$(NAME)-*
	mkdir -p $(BINDIR)/pkg

	for target in linux-amd64 linux-arm64 freebsd-amd64 freebsd-arm64 darwin-amd64 darwin-arm64; do \
		rm -rf $(BINDIR)/pkg/$$target; \
		mkdir -p $(BINDIR)/pkg/$$target; \
		cp $(BINDIR)/$(NAME)-$$target $(BINDIR)/pkg/$$target/$(NAME); \
		tar czf $(BINDIR)/$(NAME)-$$target.tar.gz -C $(BINDIR)/pkg/$$target $(NAME); \
	done
	for target in windows-amd64 windows-386; do \
		rm -rf $(BINDIR)/pkg/$$target; \
		mkdir -p $(BINDIR)/pkg/$$target; \
		cp $(BINDIR)/$(NAME)-$$target.exe $(BINDIR)/pkg/$$target/$(NAME).exe; \
		(cd $(BINDIR)/pkg/$$target && zip -q -r ../../$(NAME)-$$target.zip $(NAME).exe); \
	done
	for target in linux-amd64 linux-arm64 freebsd-amd64 freebsd-arm64 darwin-amd64 darwin-arm64; do \
		rm -f $(BINDIR)/$(NAME)-$$target; \
	done
	for target in windows-amd64 windows-386; do \
		rm -f $(BINDIR)/$(NAME)-$$target.exe; \
	done

	rm -rf $(BINDIR)/pkg

test:
	go test ./...

clean:
	rm -rf $(BINDIR)