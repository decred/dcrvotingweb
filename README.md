dcrvotingweb
============

[![Build Status](https://github.com/decred/dcrvotingweb/workflows/Build%20and%20Test/badge.svg)](https://github.com/decred/dcrvotingweb/actions)
[![ISC License](https://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

## Overview
dcrvotingweb is a simple web app that connects to dcrd and displays
information about consensus rule voting.

## Installation

## Developing

It is recommended to use Go 1.15 (or newer) for development:

```no-highlight
$ go version
go version go1.15 linux/amd64
```

To build the code:

```no-highlight
go install
```

Start dcrd with the following options:

```no-highlight
dcrd --testnet -u USER -P PASSWORD --rpclisten=127.0.0.1:19109 --rpccert=$HOME/.dcrd/rpc.cert
```

Start dcrvotingweb:

```no-highlight
dcrvotingweb
```

## Docker

Build the docker container:

```no-highlight
docker build -t decred/dcrvotingweb .
```

Run the container:

```no-highlight
docker run -it -v ~/.dcrd:/root/.dcrd -v ~/.dcrvotingweb:/root/.dcrvotingweb -p <local port>:8000 decred/dcrvotingweb
```

This example assumes you have configured `.dcrd` and `.dcrvotingweb` directories in `~` on the host machine.

Your `dcrvotingweb.conf` file will need to specificy `listen=0.0.0.0` in order for the external port mapping to work correctly.

## Contact

If you have any further questions you can join the [Decred community](https://decred.org/community/) using your preferred chat platform.

## Issue Tracker

The [integrated github issue tracker](https://github.com/decred/dcrvotingweb/issues) is used for this project.

## License

dcrvotingweb is licensed under the [copyfree](http://copyfree.org) ISC License.
