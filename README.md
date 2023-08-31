<!--
SPDX-FileCopyrightText: 2019 NetSys Lab

SPDX-License-Identifier: GPL-3.0-only
-->

# BitTorrent over SCION

BitTorrent client written in Go. Uses SCION's pathawareness with the [pathdiscovery](https://github.com/netsys-lab/scion-path-discovery) library.

## Install

```sh
go get github.com/netsys-lab/bittorrent-over-scion
```

## Usage
To use this Bittorrent client (at the moment, SCION usage is mandatory. We will support also TCP in the future), you need to have at least a SCION endhost (or a full SCION AS) running. The easiest way is to join [SCIONLab](https://www.scionlab.org/) and create two user ASes or to run a [local SCION topology](https://scion.docs.anapaya.net/en/latest/build/setup.html#setting-up-the-development-environment) (steps 7-10, to connect to a specific SCION Daemon, use the `SCION_DAEMON_ADDRESS` environment variable) with multiple ASes.

Finally, a valid .torrent file is required to start BitTorrent as seeder. To generate a torrent from a local file, you can use anacrolix `torrent-create` tool: `go run github.com/anacrolix/torrent/cmd/torrent-create samplefile >> samplefile.torrent`.

### Run a seeder
The following command runs BitTorrent as a seeder:
```sh
./bittorrent-over-scion -inPath='sample.torrent' -seed=true -file='sample.file' -local="19-ffaa:1:000,[127.0.0.1]:46000"
```

At least the following command line flags are required:
- `inPath`: Source .torrent file
- `file`: Source file from which the .torrent file was created
- `seed`: Start as seeder
- `local`: The full local SCION address, of format `ISD-AS,[IP]:Port`,

### Run a leecher
The following command runs BitTorrent as a leecher:
```
./bittorrent-over-scion -inPath='sample.torrent' -outPath='sample.file' -peer="19-ffaa:1:000,[127.0.0.1]:46000" -seed=false -local="19-ffaa:1:111,[127.0.0.1]:43000" 
```

At least the following command line flags are required:
- `inPath`: Source .torrent file
- `outPath`: Destination to which BitTorrent writes the downloaded file
- `seed`: Start as leecher (seed=false)
- `local`: The full local SCION address, of format `ISD-AS,[IP]:Port`,
- `peer`: The full remote SCION address, of format `ISD-AS,[IP]:Port`,

### Run HTTP API & frontend
The following command starts up an API that permits running & managing multiple parallel leechers & seeders on `http://localhost:8000/api`.
It also serves a frontend at `http://0.0.0.0:8000/frontend` that can be used optionally at your convenience.
```
./bittorrent-over-scion -httpApi -local="19-ffaa:1:111,[127.0.0.1]"
```

At least the following command line flags are required:
- `httpApi`: Start HTTP API instead of leecher or seeder
- `local`: The full local SCION address, of format `ISD-AS,[IP]` (without port)

Please note that port forwarding of port `8000` might be necessary if you want to use the program from within a SCIONLab VM or similar.

### Help Info
Run `bittorrent-over-scion -h` to get a full overview of all command line flags and their explanations.

### Demo Torrent
We provide a running seeder and a sample torrent file in the [demo](https://github.com/netsys-lab/bittorrent-over-scion/tree/master/demo) folder. Please visit the readme for further information.

## Roadmap
- [ ] Support SCION HTTP tracker
- [x] Support Dht based peer discovery
- [ ] Support magnet links
- [ ] Support multi-file torrents
- [ ] Support multiple torrents by one running instance
- [ ] Support TCP and SCION connections depending on peer information
- [x] Add a GUI on top of the command line client

## License
This project is licensed under the GPLv3 license. However, for accurate information regarding license and copyrights, please check individual files.

## Security
This project is at the moment under ongoing development. API's or expected behavior may change in further versions. In case you observe any security issues, please contact [me](https://github.com/martenwallewein) via mail.
