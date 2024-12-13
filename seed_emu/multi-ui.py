#!/usr/bin/env python3

import pathlib

from seedemu.compiler import Docker
from seedemu.core import Emulator, Binding, Filter
from seedemu.layers import ScionBase, ScionRouting, ScionIsd, Scion
from seedemu.layers.Scion import LinkType as ScLinkType

from BittorrentOverScionService import BittorrentOverScionService

BIN_PATH = pathlib.Path(__file__).parent.parent / 'bittorrent-over-scion'
print(BIN_PATH)

# Initialize
emu = Emulator()
base = ScionBase()
routing = ScionRouting()
scion_isd = ScionIsd()
scion = Scion()
bittorrent_over_scion = BittorrentOverScionService()

# SCION ISDs
base.createIsolationDomain(1)

# Internet Exchange
base.createInternetExchange(100)

# AS-151
as151 = base.createAutonomousSystem(151)
scion_isd.addIsdAs(1, 151, is_core=True)
as151.createNetwork('net0')
as151.createControlService('cs1').joinNetwork('net0')
as151.createRouter('br0').joinNetwork('net0').joinNetwork('ix100')
host0 = as151.createHost('host0').joinNetwork('net0', address='10.151.0.30')
emu.addBinding(Binding('bittorrent151', filter=Filter(nodeName='host0', asn=151)))
## install BitTorrent-over-SCION UI to host
bittorrent_over_scion.install('bittorrent151').setPort(8001)
## add port forwarding so we can access the UI of the node from the host system
host0.addPortForwarding(8001, 8001)

# AS-152
as152 = base.createAutonomousSystem(152)
scion_isd.addIsdAs(1, 152, is_core=True)
as152.createNetwork('net0')
as152.createControlService('cs1').joinNetwork('net0')
as152.createRouter('br0').joinNetwork('net0').joinNetwork('ix100')
host1 = as152.createHost('host0').joinNetwork('net0', address='10.152.0.30')
emu.addBinding(Binding('bittorrent152', filter=Filter(nodeName='host0', asn=152)))
## install BitTorrent-over-SCION UI to host
bittorrent_over_scion.install('bittorrent152').setPort(8002)
## add port forwarding so we can access the UI of the node from the host system
host1.addPortForwarding(8002, 8002)

# Inter-AS routing
scion.addIxLink(100, (1, 151), (1, 152), ScLinkType.Core)

# Rendering
emu.addLayer(base)
emu.addLayer(routing)
emu.addLayer(scion_isd)
emu.addLayer(scion)
emu.addLayer(bittorrent_over_scion)

emu.render()

# Compilation
emu.compile(Docker(), './multi_ui_output')
