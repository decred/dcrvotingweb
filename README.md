# hardforkdemo

hardforkdemo is a simple web app that connects to dcrd and displays
information about the tesnet hardfork voting.

## Installation

## Developing


1. Install Go version 1.11 or higher, and Git.
Make sure each of these are in the PATH.

2. Clone this repository.
    
    2.1. If cloning under GOPATH
    ``` bash
    go get -v github.com/decred/hardforkdemo
    cd $GOPATH/src/github.com/decred/hardforkdemo
    ```

    2.2 If not, just clone as a git repository.
    ```bash
    git clone https://github.com/decred/hardforkdemo
    ```

3. Install hardforkdemo

    3.1. If building in a folder under GOPATH, it is necessary to explicitly build with modules enabled:

    ``` bash
    cd $GOPATH/src/github.com/decred/hardforkdemo
    export GO111MODULE=on
    go install
    ```

    3.2. If building outside of GOPATH, modules are automatically enabled, and go install is sufficient.
    ``` bash
    go install
    ```

4. Start dcrd with the following options.  

```bash
dcrd --testnet -u USER -P PASSWORD --rpclisten=127.0.0.1:19109 --rpccert=$HOME/.dcrd/rpc.cert
```

Start hardforkdemo with the following options.  

```bash
hardforkdemo --testnet -u USER -P PASSWORD --rpccert ~/.dcrd/rpc.cert -c 127.0.0.1:19109
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
