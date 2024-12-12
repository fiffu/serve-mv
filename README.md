# serve-mv

Serve RPG Maker MV games on a HTTP server.

This is mainly for running games in a native JavaScript interpreter (i.e. web browser),
which is useful when the node binary bundled with Game.exe is not suitable for your
operating system.

This repository offers two implementations, one in Python 3 and another in Golang.


## Design and caveats

**Why are we tampering with /etc/hosts?**

When running the game in your browser, the saving mechanism writes to the browser cache
(LocalStorage) instead of the disk (which is the behaviour of Game.exe).

LocalStorage is unique according to the DNS record (or more precisely, the origin) used
to access the game. Hence, if you're just running all your games using a HTTP server
without any differentiation in the DNS, all your saves will be sharing save slots
across every game.

To avoid this problem, serve-mv will generate a DNS record based on the game title, then
temporarily apply it to `/etc/hosts`. Hence, `sudo` is needed in order to update this file.
A backup of the original `/etc/hosts` will be retained while the HTTP server is running.

I'll add an option later to disable this temporary DNS, which will also avoid `sudo`.


## Usage

```sh
# Build the binary
go get
go build -o gomv

# Option 1: Copy the binary to the game directory where Game.exe is, then execute
cp gomv /path/to/game/dir
cd /path/to/game/dir
sudo ./gomv

# Option 2: Don't move the binary, just pass --dir
cd /path/to/game/dir
sudo /git/serve-mv/gomv --dir .
```
