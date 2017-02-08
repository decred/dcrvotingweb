# hardforkdemo

hardforkdemo is a simple web app that connects to dcrd and displays
information about the tesnet hardfork voting.

## Installation

## Developing

``` bash
git clone https://github.com/decred/hardforkdemo.git
cd hardforkdemo
glide install
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

