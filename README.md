# hardforkdemo

[![Build Status](https://travis-ci.org/decred/hardforkdemo.png?branch=master)](https://travis-ci.org/decred/hardforkdemo)
[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

hardforkdemo is a simple web app that connects to dcrd and displays
information about the tesnet hardfork voting.

## Installation

## Developing

It is recommended to use Go 1.12 for development:

```no-highlight
$ go version
go version go1.12 linux/amd64
```

To build the code:

```no-highlight
go install
```

Start dcrd with the following options:

```no-highlight
dcrd --testnet -u USER -P PASSWORD --rpclisten=127.0.0.1:19109 --rpccert=$HOME/.dcrd/rpc.cert
```

Start hardforkdemo:

```no-highlight
hardforkdemo
```

## Docker

Build the docker container:

```no-highlight
docker build -t decred/hardforkdemo .
```

Run the container:

```no-highlight
docker run -it -v ~/.dcrd:/root/.dcrd -v ~/.hardforkdemo:/root/.hardforkdemo -p <local port>:8000 hardforkdemo
```

This example assumes you have configured `.dcrd` and `.hardforkdemo` directories in `~` on the host machine.

Your `hardforkdemo.conf` file will need to specificy `listen=0.0.0.0` in order for the external port mapping to work correctly.

## Contact

If you have any further questions you can join the [Decred community](https://decred.org/community/) using your preferred chat platform.

## Issue Tracker

The [integrated github issue tracker](https://github.com/decred/hardforkdemo/issues) is used for this project.

## License

hardforkdemo is licensed under the [copyfree](http://copyfree.org) ISC License.
