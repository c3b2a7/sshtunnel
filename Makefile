NAME=sshtunnel
BINDIR=bin
VERSION=$(shell git describe --tags --always || echo "unknown version")
BUILDTIME=$(shell date -u)
GOBUILD=CGO_ENABLED=0 go build -trimpath -ldflags ' \
		-X "main.Version=$(VERSION)" \
		-X "main.BuildTime=$(BUILDTIME)" \
		-w -s -buildid='

all: linux-amd64 linux-arm64 freebsd-amd64 freebsd-arm64 darwin-amd64 darwin-arm64 win64 win32

linux-amd64:
	GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

linux-arm64:
	GOARCH=arm64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

freebsd-amd64:
	GOARCH=amd64 GOOS=freebsd $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

freebsd-arm64:
	GOARCH=arm64 GOOS=freebsd $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

darwin-amd64:
	GOARCH=amd64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

darwin-arm64:
	GOARCH=arm64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

win64:
	GOARCH=amd64 GOOS=windows $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.exe

win32:
	GOARCH=386 GOOS=windows $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.exe

releases: linux-amd64 linux-arm64 freebsd-amd64 freebsd-arm64 darwin-amd64 darwin-arm64 win64 win32
	chmod +x $(BINDIR)/$(NAME)-*
	tar czf $(BINDIR)/$(NAME)-linux-amd64.tar.gz -C $(BINDIR) $(NAME)-linux-amd64
	tar czf $(BINDIR)/$(NAME)-linux-arm64.tar.gz -C $(BINDIR) $(NAME)-linux-arm64
	tar czf $(BINDIR)/$(NAME)-freebsd-amd64.tar.gz -C $(BINDIR) $(NAME)-freebsd-amd64
	tar czf $(BINDIR)/$(NAME)-freebsd-arm64.tar.gz -C $(BINDIR) $(NAME)-freebsd-arm64
	tar czf $(BINDIR)/$(NAME)-darwin-amd64.tar.gz -C $(BINDIR) $(NAME)-darwin-amd64
	tar czf $(BINDIR)/$(NAME)-darwin-arm64.tar.gz -C $(BINDIR) $(NAME)-darwin-arm64
	rm $(BINDIR)/*-amd64
	rm $(BINDIR)/*-arm64
	zip -m -j $(BINDIR)/$(NAME)-win32.zip $(BINDIR)/$(NAME)-win32.exe
	zip -m -j $(BINDIR)/$(NAME)-win64.zip $(BINDIR)/$(NAME)-win64.exe

test:
	go test ./...

clean:
	rm -rf $(BINDIR)