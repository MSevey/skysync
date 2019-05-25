dependencies:
	go get -u gitlab.com/NebulousLabs/Sia/node/api/client
	go get -u github.com/fsnotify/fsnotify
	go get -u gitlab.com/NebulousLabs/Sia/modules

all:
	go build -o siasync *.go

release:
	go build -o siasync *.go
