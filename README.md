# BitTorrent over SCION

BitTorrent client written in Go. Uses SCION's pathawarenss with the [pathdiscovery](https://github.com/netsys-lab/scion-path-discovery) libary.

## Install

```sh
go get github.com/martenwallewein/torrent-client
```

## Usage
To use this Bittorrent client (at the moment, SCION usage is mandatory. We will support also TCP in the future), you need to have at least a SCION endhost (or a full SCION AS) running. The easiest way is to join [SCIONLab](https://www.scionlab.org/) and create a user-AS. 

Furthermore, you need valid TLS certificates (used by [quic-go](https://github.com/lucas-clemente/quic-go)). To create these, use:
`openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes`


### Run a seeder
```sh
SCION_CERT_KEY_FILE=key.pem SCION_CERT_FILE=cert.pem ./torrent-client -inPath='5G_1.torrent' -outPath='5G.file' -peer="19-ffaa:1:111,[127.0.0.1]:43000" -seed=true -file=5G.file -local="19-ffaa:1:000,[127.0.0.1]:46000"
```

### Run a leecher
```
SCION_CERT_KEY_FILE=key.pem SCION_CERT_FILE=cert.pem ./torrent-client -inPath='5G_1.torrent' -outPath='ubuntu.file' -peer="19-ffaa:1:c3f,[127.0.0.1]:43000" -seed=false -file='5G.file' -local="19-ffaa:1:111,[127.0.0.1]:43000"
```

## Limitations
* Only supports `.torrent` files (no magnet links)
* Only supports HTTP trackers
* Does not support multi-file torrents

## Roadmap
- [ ] Support SCION HTTP tracker
- [ ] Support Dht based peer discovery
- [ ] Support magnet links
- [ ] Support multi-file torrents
