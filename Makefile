PLATFORMS := linux/amd64 windows/amd64 darwin/amd64

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

default: build

build:
	go build -o skysync *.go

release: $(PLATFORMS)

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build -o 'Skysync-$(os)-$(arch)' *.go

.PHONY:	release	$(PLATFORMS)
