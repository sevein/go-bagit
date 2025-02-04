# go-bagit
Early Go implementation of the bagit specification

## Binary Install
latest binary:
* osx https://github.com/nyudlts/go-bagit/releases/download/v0.1.1-alpha/go-bagit-v0.1.1-alpha-osx.tgz
* linux https://github.com/nyudlts/go-bagit/releases/download/v0.1.1-alpha/go-bagit-v0.1.1-alpha-linux.tgz 

`tar xvzf go-bagit-[VERSION]-[OS].tgz`</br> 
`sudo mv go-bagit /path/to/bins`

## Building From Source
`go build -o go-bagit main/main.go`

## Supported Algorithms
* md5
* sha1
* sha256
* sha512

## Usage:
`go-bagit [command] [flags]`

## Available Commands:
    help        Help about any command
    validate    validate a bag
    create      create a bag
    modify      modify a bag

## Validate Command

### Usage
`go-bagit validate [flags]`

### Flags
    --bag string        location of the bag to be validated, Mandatory
    -h, --help          help for validate
    --fast              test whether the bag directory has the expected payload specified in the checksum manifests without performing checksum validation to detect corruption
    --completeness-only only test whether the bag directory has the number of files and total size specified in Payload-Oxum without performing checksum validation to detect corruption

## Create Command

### Usage
`go-bagit create [flags]`

### Flags

    --algorithm string   the algorithm used for checksums (default "md5")
    -h, --help           help for create
    --input-dir string   the directory to be bagged
    --processes int      Use multiple processes to calculate checksums faster (default 1)

## Modify Command

### Usage
`go-bagit modify [flags]`

### Flags
    --add-to-bag    add a file to tag manifest
    --bag string    bag to be validated
    --file string   location of a file
    -h, --help      help for modify
    
