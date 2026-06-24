# Pointhole

An easy way to browse and transfer files from your ssh/terminal based access to a nice GUI. No FTP or port forwarding!

## Installation

### Client

On Windows:

```bat
powershell -NoProfile -ExecutionPolicy Bypass -Command "Invoke-WebRequest https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-client.bat -OutFile install-client.bat; Start-Process cmd.exe -ArgumentList '/c install-client.bat' -Wait; Remove-Item install-client.bat;"
```

On Linux/MacOS:

```bash
curl -fsSL https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-client.sh | bash
```

### Server

On Windows:

```bat
powershell -NoProfile -ExecutionPolicy Bypass -Command "Invoke-WebRequest https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-server.bat -OutFile install-server.bat; Start-Process cmd.exe -ArgumentList '/c install-server.bat' -Wait; Remove-Item install-server.bat";
```

On Linux/MacOS:

```bash
curl -fsSL https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-server.sh | bash
```

The installers will provide you with instructions on how to run the programs.

### Direct URLs

Client: [Linux](https://cdn.lu2000luk.com/pointhole/client/client) or [Windows](https://cdn.lu2000luk.com/pointhole/client/client.exe)

Server: [Linux](https://cdn.lu2000luk.com/pointhole/server/server) or [Windows](https://cdn.lu2000luk.com/pointhole/server/server.exe)


## Demo

We provide a demo instance for you to try out the program (demo instance resets every 30 mins). Code to connect to it: `DEMO67676767`

## Stack

Built with Go.

- client/ Desktop app with cimgui-go
- server/ CLI/file server

Server software at https://git.lu2000luk.com/lu2000luk/end2end (all packets are encrypted)

## How does it do it

Websockets.

## Don't move large files with this!

Read and writes are mainly for editing small files such as configs. Large files are supported but the speed is capped at less than 1mbps and errors might occour.

## Planned

- QoL (Upload to pastebin, integrated terminal, explorer tabs/favourites)