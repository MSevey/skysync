# [![Siasync Logo](https://i.imgur.com/bUJTPqU.png)](https://medium.com/@tbenz9/introducing-siasync-27452e90682f) Siasync v1.0

Siasync is a simple utility that syncs a local folder to [Sia](https://gitlab.com/NebulousLabs/Sia).

## Usage

First, you must create a Sia node and form contracts with hosts. Then, simply

`#> siasync [path-to-folder]`

siasync will upload every file in that directory to sia and continuously sync, until stopped.

Full usage:
```
#> siasync -h
usage: siasync <flags> <directory-to-sync>
  for example: ./siasync -password abcd123 /tmp/sync/to/sia
-address string
        Sia's API address (default "127.0.0.1:9980")
  -agent string
        Sia agent (default "Sia-Agent")
  -archive
        Files will not be removed from Sia, even if they are deleted locally
  -exclude string
        Comma separated list of file extensions to skip, all other files will be copied.
  -include string
        Comma separated list of file extensions to copy, all other files will be ignored.
  -password string
        Sia's API password
  -subfolder string
        Folder on Sia to sync files too (default "siasync")
```
More information can also be found in my [Introducing
Sia](https://medium.com/@tbenz9/introducing-siasync-27452e90682f) blog post.

## Building from Source
Siasync is written in Go, you have have a working Go installation before
attempting to build Siasync from source.

build Siasync dependencies:

`make dependencies`

build Siasync:

`make all`

## License

The MIT License (MIT)
