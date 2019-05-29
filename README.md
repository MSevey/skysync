# [![Siasync Logo](https://i.imgur.com/bUJTPqU.png)](https://medium.com/@tbenz9/introducing-siasync-27452e90682f) Siasync v1.0

Siasync is a new utility that will monitor a folder and synchronize its contents
to the [Sia network](https://sia.tech/). As new files are created or removed it
will keep Sia in sync with the local source folder. Siasync also supports more
advanced features like only syncing certain file extensions, or excluding
certain file extensions, or archive mode which won't delete files from Sia even
if they are deleted locally. Best of all, it works on Windows, MacOS, and Linux.

## Installation
Siasync Binaries can be found on [Github](https://github.com/tbenz9/siasync/releases).  Simply download the
relevant binary, rename it to Siasync (Siasync.exe for Windows) and execute it in your operating systems' command line utility.

## Usage
*Siasync assumes that you already have a running siad daemon and are ready to
upload files. Siasync does some basic checking of the allowance and contracts to
ensure it can upload, but it does not resolve any issues it finds with your
wallet or contracts. If you're not sure how to run Sia and form storage
contracts check out [this blog
post](https://blog.sia.tech/a-guide-to-sia-ui-v1-4-0-7ec3dfcae35a).*

The most basic way to use Siasync is to have it stay 100% in sync with a local
folder.

```
#> siasync /tmp/foo
```

That will keep the local folder `/tmp/foo` synced to Sia. As new files and
folders are created they will be uploaded to Sia, as files and folders are
deleted they will be removed.

By default, files get uploaded into a `siasync` folder on Sia.  You can see the
files with `siac renter ls /siasync` when using Sia version 1.4.1 or later.

Now let's get *fancy*!

```
#> siasync -include jpg,jpeg,gif,png,raw -archive true -address 127.0.0.1:4280
-subfolder demo -password <your-api-password> /tmp/foo/
```

Let's break down this command one argument at a time:


`-include jpg,jpeg,gif,png,raw` - Only files with jpg, jpeg, gif, png, or raw
file extensions will be synced. All other files will be ignored. More
information on this flag and the `-exclude` flag can be found in this Siasync
case study.

`-archive true` - Never delete files from Sia, even if they are deleted locally.

`-address 127.0.0.1:4280` - Use the Sia daemon running at 127.0.0.1:4280 instead
of the default 127.0.0.1:9980.

`-subfolder demo` - Sync the files to the "demo" folder on Sia, instead of the
default "siasync" folder. Siasync will create the "demo" folder if it doesn't
exist. You can see the files with `siac renter ls /demo/`.

`-password <your-api-password>` - Use your API password instead of whatever API
password Siasync was able to find. Siasync checks the default Sia locations and
environment variables for API passwords.

`/tmp/foo/` - The local folder you want synced to Sia.

#### Quick demo starting Siasync, adding a file, then deleting it.
[![](https://i.imgur.com/YEnCuKV.gif)](https://medium.com/@tbenz9/introducing-siasync-27452e90682f)

A full list of Siasync commands can be found with `Siasync -h`
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
        Comma separated list of file extensions to skip, all other files will be
copied.
  -include string
        Comma separated list of file extensions to copy, all other files will be
ignored.
  -password string
        Sia's API password
  -subfolder string
        Folder on Sia to sync files too (default "siasync")
```

## Building from Source
Siasync is written in Go, you have have a working Go installation before
attempting to build Siasync from source.
#### build Siasync dependencies
`make dependencies`
#### build Siasync
`make`
## License
The MIT License (MIT)
