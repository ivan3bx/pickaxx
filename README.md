# Pickaxx

A standalone web console for managing a Minecraft server instance.

![Screenshot](https://user-images.githubusercontent.com/179345/101236138-365c2400-3694-11eb-8337-8f432a09ea6f.png "Screenshot #1")

The goals of this project are as follows:

* Minimal configuration, sane defaults (drop a server into the tool and go)
* Single binary, small memory footprint
* Basic authentication

## Current state

* **In development**. It can start an existing minecraft server and send/receive output, but not much else.
* It can not yet bootstrap new server instances (i.e. drag and drop).
* It does not handle any authentication yet, making it not yet able to run safely on a public server.

## Developer Setup

1. Install gowatch, packr2 and goreleaser
```bash
GO111MODULE=off go get -u github.com/silenceper/gowatch
GO111MODULE=off go get -u github.com/gobuffalo/packr/v2/packr2
wget -O /tmp/goreleaser_amd64.deb https://github.com/goreleaser/goreleaser/releases/download/v0.154.0/goreleaser_amd64.deb
sudo dpkg -i /tmp/goreleaser_amd64.deb
```
2. Until this tool can bootstrap new Minecraft instances on it's own, manually download `server.jar` from Minecraft's site (see below), and copy this into the path `testserver/server.jar`.
3. Run tests (or use `make test`).
4. Run `make` which will start the server. Load http://localhost:8080

## Dependencies

* This project uses [go-watch](https://github.com/silenceper/gowatch) to run/restart the server.
* Compatible Java runtime (tested on [OpenJDK 15.0.1](http://openjdk.java.net/projects/jdk/15/)).
* Minecraft server (tested on [1.16.4](https://www.minecraft.net/en-us/download/server)).
