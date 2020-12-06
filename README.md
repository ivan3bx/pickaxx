# Pickaxx

A standalone web console for managing a Minecraft server instance.

![Screenshot](https://user-images.githubusercontent.com/179345/101236138-365c2400-3694-11eb-8337-8f432a09ea6f.png "Screenshot #1")

The goals of this project are as follows:

* Minimal configuration, sane defaults (drop a server into the tool and go)
* Single binary, small memory footprint
* Basic authentication

## Developer Setup

```bash
go get -u github.com/silenceper/gowatch
```

* See Makefile. This project uses [go-watch](https://github.com/silenceper/gowatch) to run/restart the server.

## Dependencies

* Compatible Java runtime (tested on [OpenJDK 15.0.1](http://openjdk.java.net/projects/jdk/15/)).
* Minecraft server (tested on [1.16.4](https://www.minecraft.net/en-us/download/server)).
