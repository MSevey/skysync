# SkySync v0.1.0

SkySync is a utility that will monitor a folder and synchronize its contents to
the [SkyNet](https://siasky.net/). As new files are created it will upload them
to SkyNet. SkySync also supports more advanced features like only syncing
certain file extensions or excluding certain file extensions. Best of all, it
works on Windows, MacOS, and Linux.

## Installation
SkySync Binaries can be found on
[Github](https://github.com/MSevey/skysync/releases).  Simply download the
relevant binary, rename it to Skysync (Skysync.exe for Windows) and execute it
in your operating systems' command line utility.

## Usage
SkySync uses the SkyNet portal hosted by Nebulous [here](https://siasky.net) to
upload files to SkyNet.

The most basic way to use SkySync is to have it monitor a local folder and
upload any files it sees to SkyNet.

```
#> skysync /tmp/foo
```

That will upload the contents of local folder `/tmp/foo` to SkyNet. As new files
and folders are created they will be uploaded to Skynet.

Files uploaded to SkyNet are uploaded as SkyFiles and a SkyLink will be
returned. These SkyLinks are then persisted along with the corresponding
filename.

A full list of SkySync commands can be found with `skysync -h`
```
#> skysync -h
usage: skysync <flags> <directory-to-sync>
  for example: ./skysync --dry-run=true /tmp/sync/to/skynet

  -debug
        Enable debug mode. Warning: generates a lot of output.
  -dry-run
        Show what would have been uploaded without changing files in Sia
  -exclude string
        Comma separated list of file extensions to skip, all other files will be copied.
  -include string
        Comma separated list of file extensions to copy, all other files will be ignored.
  -sync-only
        Sync, don't monitor directory for changes
```

## Building from Source
SkySync is written in Go, you have have a working Go installation before
attempting to build SKySync from source.

#### build Siasync
`make`

## License
The MIT License (MIT)


## Credit
This Repo was forked and modified from
[SiaSync](https://github.com/tbenz9/siasync)
