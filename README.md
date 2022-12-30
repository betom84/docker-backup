# Docker Backup

CLI Tool to backup content of Docker container volumes to CIFS/SMB network shares (e.g. NAS).

## Install
```
go install github.com/betom84/docker-backup@latest
```

## Usage
```
usage: ./docker_backup [options]

  -container string
        Comma separated list of Docker container names
  -hold
        Hold container(s) during backup
  -host string
        TCP Docker host (port 2375)
  -target string
        CIFS volume address (user:pass@host/path)

example: ./docker_backup -host docker.local -target user:secret@mynas.local/backups -container container1,container2
```

## Prerequisites

- Docker daemon must allow TCP connections at port 2375 without authentication ([howto setup dockerd to allow tcp connections](https://docs.docker.com/engine/reference/commandline/dockerd/#daemon-socket-option))