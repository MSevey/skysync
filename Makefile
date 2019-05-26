PLATFORMS := linux/amd64 windows/amd64 darwin/amd64

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

default: build

build:
	go build -o siasync *.go

release: $(PLATFORMS)

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build -o 'Siasync-$(os)-$(arch)' *.go

dependencies:
	go get -u gitlab.com/NebulousLabs/Sia/node/api/client
	go get -u github.com/fsnotify/fsnotify
	go get -u gitlab.com/NebulousLabs/Sia/modules
	
.PHONY:	release	$(PLATFORMS)
