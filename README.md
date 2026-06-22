# Pointhole

An easy way to browse and transfer files from your ssh/terminal based access to a nice GUI. No FTP or port forwarding!

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
- FUSE filesystem