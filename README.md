# hardforkdemo

[![Build Status](https://travis-ci.org/decred/hardforkdemo.png?branch=master)](https://travis-ci.org/decred/hardforkdemo)
[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

hardforkdemo is a simple web app that connects to dcrd and displays
information about the tesnet hardfork voting.

## Installation

## Developing

``` bash
go get -v github.com/golang/dep/cmd/dep
go get -v github.com/decred/hardforkdemo
cd $GOPATH/github.com/decred/hardforkdemo
dep ensure
go install
```

Start dcrd with the following options.  

```bash
dcrd --testnet -u USER -P PASSWORD --rpclisten=127.0.0.1:19109 --rpccert=$HOME/.dcrd/rpc.cert
```

Start hardforkdemo

```bash
hardforkdemo
```

## Docker

Build the docker container:

```bash
docker build -t decred/hardforkdemo .
```

Run the container:

```bash
docker run -it -v ~/.dcrd:/root/.dcrd -v ~/.hardforkdemo:/root/.hardforkdemo -p <local port>:8000 hardforkdemo
```

This example assumes you have configured `.dcrd` and `.hardforkdemo` directories in `~` on the host machine.

Your `hardforkdemo.conf` file will need to specificy `listen=0.0.0.0` in order for the external port mapping to work correctly.

## Contact

If you have any further questions you can find us at:

- irc.freenode.net (channel #decred)
- [webchat](https://webchat.freenode.net/?channels=decred)
- forum.decred.org
- decred.slack.com

## Issue Tracker

The
[integrated github issue tracker](https://github.com/decred/hardforkdemo/issues)
is used for this project.

## License

hardforkdemo is licensed under the [copyfree](http://copyfree.org) ISC License.
