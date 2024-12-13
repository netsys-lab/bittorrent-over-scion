from __future__ import annotations
from pathlib import Path

from seedemu.core import Node, Server, Service

BIN_PATH = Path(__file__).parent.parent / 'bittorrent-over-scion'
print(BIN_PATH)

class BittorrentOverScionServer(Server):
    """!
    @brief BitTorrent over SCION HTTP (API & frontend) server.
    """

    __port: int

    def __init__(self):
        """!
        @brief BittorrentOverScionServer constructor.
        """
        super().__init__()
        self.__port = 8000

    def setPort(self, port: int) -> BittorrentOverScionServer:
        """!
        @brief Set port the HTTP server listens on.

        @param port
        @returns self, for chaining API calls.
        """
        self.__port = port

        return self

    def install(self, node: Node):
        """!
        @brief Install the service.
        """
        node.importFile(BIN_PATH, '/bittorrent-over-scion')
        node.appendStartCommand('chmod +x /bittorrent-over-scion')
        node.appendStartCommand(
            '/bittorrent-over-scion -httpApi -httpApiAddr=0.0.0.0:{} -local=$(scion address)'.format(self.__port),
            fork=True
        )
        node.appendClassName("BittorrentOverScionService")

    def print(self, indent: int) -> str:
        out = ' ' * indent
        out += 'BitTorrent over SCION HTTP (API & frontend) server object.\n'
        return out


class BittorrentOverScionService(Service):
    """!
    @brief BitTorrent over SCION HTTP (API & frontend) server service class.
    """

    def __init__(self):
        """!
        @brief BittorrentOverScionService constructor.
        """
        super().__init__()
        self.addDependency('Base', False, False)
        self.addDependency('Scion', False, False)

    def _createServer(self) -> Server:
        return BittorrentOverScionServer()

    def getName(self) -> str:
        return 'BittorrentOverScionService'

    def print(self, indent: int) -> str:
        out = ' ' * indent
        out += 'BittorrentOverScionServiceLayer\n'
        return out