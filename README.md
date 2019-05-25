# siasync

Siasync is a simple utility that syncs a local folder to [Sia](https://gitlab.com/NebulousLabs/Sia).

## Usage

First, you must create a Sia node and form contracts with hosts. Then, simply

`siasync [path-to-folder]`

siasync will upload every file in that directory to sia and continuously sync, until stopped.

## Building from Source
Siasync is written in Go, you have have a working Go installation before
attempting to build Siasync from source.

go get the Siasync dependencies:

`make dependencies`

build Siasync

`make all`

## License

The MIT License (MIT)
