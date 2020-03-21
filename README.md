# WebSocket Chat

This is a simple Go program and a JavaScript web application that allows the
user to "chat" with a command line program running on the host computer.

This program was developed to expose through the websocket protocol the chat
program developed for testing the [Othernet Dreamcatcher](https://othernet.is/products/dreamcatcher-3-0) 
capabilities.

## Message length limit

Currently, messages are limited to 47 characters. This is the number of
characters that we could reliably transmit in our trials. If you would like to
build a custom version to change the limit, you can do that by modifying the 
JavaScript part of the code.

## Getting the latest version

Look under [releases](https://github.com/foxbunny/wschat/releases).

## Running the server

To start the server run with:

```bash
./wschat PATH_TO_CHAT
```

`PATH_TO_CHAT` is full path to the Othernet chat program.

You can change the address and the port to which the server should bind by
specifying the `--addr` command line argument:

```bash
./wschat --addr 0.0.0.0:3000 PATH_TO_CHAT
```

## Developing

You will need both Go and NodeJS in order to develop this application. This 
program uses [statik](https://github.com/rakyll/statik) to bundle the static
assets inside the binary.

To build the front end, run the following command:

```bash
npm run build
```

This will build the static assets and create a `dist` directory containing
the compiled bundle. It will also update the `statik` package.

To build the server run the usual:

```bash
go build
```

You will end up with `wschat` executable file (or `wschat.exe` on Windows) in
the project directory.

## Cross-compiling for Dreamcatcher

To cross-compile, you will need to have [upx](https://upx.github.io) installed.

To cross-compile this program for the Dreamcatcher (build on a non-Dreamcatcher 
machine), run the following command:

```bash
GOARM=7 GOARCH=arm GOOS=linux go build -ldflags="-s -w"
```

On Windows machines, the above will not work. Use this instead:

```bash
set GOARM=7 
set GOARCH=arm 
set GOOS=linux 
go build -ldflags="-s -w"
```

Once the build is done, we compress the binary using UPX:

```bash
upx --brute wschat
```
