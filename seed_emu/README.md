# BitTorrent over SCION

## SEED-Emulator

You can run the BitTorrent over SCION application in the SEED-Emulator emulator.
All operating modes (leecher, seeder, HTTP API & frontend) should work.

In the following it is assumed that you've the SEED-Emulator set up already (installed dependencies, Python package in `PYTHONPATH`).
If not, clone the [SEED-Emulator Git repository](https://github.com/seed-labs/seed-emulator) and follow the steps in [this section of the emulator's README](https://github.com/seed-labs/seed-emulator?tab=readme-ov-file#examples) roughly.
Also, you have to have general knowledge on how to create topologies in the emulator with Python scripts, and you need an already installed Go toolchain.

In order to use BitTorrent over SCION in the emulator, you first need to compile the program to a binary that can be added to hosts in SEED, like so:
```
git clone https://github.com/netsys-lab/bittorrent-over-scion.git
cd bittorrent-over-scion
go build -o idint-vis-server main.go
```

Once that is done, you can import the binary on your host system to nodes in your SEED topology like so:
```Python
node.importFile('../bittorrent-over-scion') # or whereever you built the binary
node.appendStartCommand('chmod +x /bittorrent-over-scion')
```

Then, if you attach to a shell of the node that you have imported the binary to, you can just use the program as usual.

### Ready-to-use HTTP API & frontend service

There is also a service that you can use to install the HTTP server with its API and frontend in `BittorrentOverScionService.py`.
In order to use it, the binary has to be compiled to `../bittorrent-over-scion` as explained above.

An example topology with two nodes running in different SCION ASes that renders to Docker containers can be found in `multi-ui.py`, showing how to utilize the service.
Render & start with:

```
python3 multi-ui.py
cd multi_ui_output
docker compose up --build
```

The idea is that you open multiple frontends for every emulated host in the browser of the system running the emulator, that's why it port-forwards the HTTP ports respectively:

- http://localhost:8001/
- http://localhost:8002/