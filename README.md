# serve-mv

Serve RPG Maker MV games on a HTTP server.

This is mainly for running games in a native JavaScript interpreter (i.e. web browser),
which is useful when the node binary bundled with Game.exe is not suitable for your
operating system.

## Usage

```sh
# Build the binary
go get
go build -o gomv

# Option 1: Copy the binary to the game directory where Game.exe is, then execute
cp gomv /path/to/game/dir
cd /path/to/game/dir
./gomv

# Option 2: Don't move the binary, just pass --dir
cd /path/to/game/dir
/git/serve-mv/gomv --dir .
```
