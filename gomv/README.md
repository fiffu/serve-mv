```sh
# Install deps
go get

# Build binary
go build -o serve-mv

# Move it to the same path where Game.exe resides
mv serve-mv /path/to/game

# Run with default options.
# sudo is required, because we're modifying /etc/hosts
sudo ./serve-mv

# Show usage hints
./serve-mv -h
```