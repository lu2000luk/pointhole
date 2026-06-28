# Pointhole

An easy way to browse and transfer files from your ssh/terminal based access to a nice GUI. No FTP or port forwarding!

## Demo

We provide a demo instance for you to try out the program (demo instance resets every 30 mins). Code to connect to it: `DEMO67676767`

## Installation

### Client

On Windows:

```bat
powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-client.ps1 | iex"
```

On Linux/MacOS:

```bash
curl -fsSL https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-client.sh | bash
```

### Server

On Windows:

```bat
powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-server.ps1 | iex"
```

On Linux/MacOS:

```bash
curl -fsSL https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-server.sh | bash
```

The installers will provide you with instructions on how to run the programs.

### Direct URLs

Client: [Linux](https://cdn.lu2000luk.com/pointhole/client/client) or [Windows](https://cdn.lu2000luk.com/pointhole/client/client.exe)

Server: [Linux](https://cdn.lu2000luk.com/pointhole/server/server) or [Windows](https://cdn.lu2000luk.com/pointhole/server/server.exe)

## Support

Linux and Windows are tested to work. MacOS support is unknown, the server should have no issues since the filesystem is identical to Linux. You might encounter bugs using the client on MacOS.

## Stack

Built with Go.

- client/ Desktop app with cimgui-go
- server/ CLI/file server

Server software at https://git.lu2000luk.com/lu2000luk/end2end (all packets are encrypted)

Demo server runs on https://git.lu2000luk.com/lu2000luk/demobox

## How does it do it

Websockets.

## Don't move large files with this!

Read and writes are mainly for editing small files such as configs. Large files are supported but the speed is capped at less than 1mbps and errors might occour.

## Planned

- PTY support for the SSH feature (makes stuff like the node repl work and makes bash/sh better)
- QoL (explorer tabs/favourites)
