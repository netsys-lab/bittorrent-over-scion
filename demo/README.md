# BitTorrent over SCION - Demo Torrent
To easily test BitTorrent over SCION as leecher, we provide a demo seeder that can be used to download a sample file containing SCION videos. To test the demo torrent, you need to have at least a SCION endhost (or a full SCION AS) running. The easiest way is to join [SCIONLab](https://www.scionlab.org/) and create two user ASes or to run a [local SCION topology](https://scion.docs.anapaya.net/en/latest/build/setup.html#setting-up-the-development-environment) (steps 7-10, to connect to a specific SCION Daemon, use the `SCION_DAEMON_ADDRESS` environment variable) with multiple ASes.


The demo seeder runs under the following SCION address: `19-ffaa:1:c3f,[141.44.25.148]:43000`. To download the file, please use the .torrent file in this folder and the newest version of BitTorrent over SCION. A sample command (executed from the bittorrent-over-scion folder) to fetch the file is: `SCION_CERT_KEY_FILE=key.pem SCION_CERT_FILE=cert.pem ./bittorrent-over-scion -inPath='demo/scion-videos.torrent' -outPath='demo/scion-videos.file' -peer="19-ffaa:1:c3f,[141.44.25.148]:43000" -seed=false -local="19-ffaa:1:111,[127.0.0.1]:43000"`.

Please follow the usage guideline in the main readme to create the TLS certificates and replace `19-ffaa:1:111,[127.0.0.1]:43000` with your local SCION address.
