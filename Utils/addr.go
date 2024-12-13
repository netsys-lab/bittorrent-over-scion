package util

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/phayes/freeport"
	"github.com/scionproto/scion/go/lib/addr"
	"github.com/scionproto/scion/go/lib/sciond"
	"github.com/scionproto/scion/go/lib/snet"
	"github.com/scionproto/scion/go/lib/sock/reliable"
)

// findAnyHostInLocalAS returns the IP address of some (infrastructure) host in the local AS.
func findAnyHostInLocalAS(ctx context.Context, sciondConn sciond.Connector) (*net.UDPAddr, error) {
	addr, err := sciond.TopoQuerier{Connector: sciondConn}.UnderlayAnycast(ctx, addr.SvcCS)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func findSciond(ctx context.Context) (sciond.Connector, error) {
	address, ok := os.LookupEnv("SCION_DAEMON_ADDRESS")
	if !ok {
		address = sciond.DefaultAPIAddress
	}
	sciondConn, err := sciond.NewService(address).Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to SCIOND at %s (override with SCION_DAEMON_ADDRESS): %w", address, err)
	}
	return sciondConn, nil
}

func findDispatcher() (reliable.Dispatcher, error) {
	path, err := findDispatcherSocket()
	if err != nil {
		return nil, err
	}
	dispatcher := reliable.NewDispatcher(path)
	return dispatcher, nil
}

func findDispatcherSocket() (string, error) {
	path, ok := os.LookupEnv("SCION_DISPATCHER_SOCKET")
	if !ok {
		path = reliable.DefaultDispPath
	}

	if err := statSocket(path); err != nil {
		return "", fmt.Errorf("error looking for SCION dispatcher socket at %s (override with SCION_DISPATCHER_SOCKET): %w", path, err)
	}
	return path, nil
}

func statSocket(path string) error {
	fileinfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !isSocket(fileinfo.Mode()) {
		return fmt.Errorf("%s is not a socket (mode: %s)", path, fileinfo.Mode())
	}
	return nil
}

func isSocket(mode os.FileMode) bool {
	return mode&os.ModeSocket != 0
}

func GetLocalHost() (*net.UDPAddr, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	sciondConn, err := findSciond(ctx)
	if err != nil {
		return nil, err
	}
	hostInLocalAS, err := findAnyHostInLocalAS(ctx, sciondConn)
	if err != nil {
		return nil, err
	}
	return hostInLocalAS, nil
}

func GetDefaultLocalAddr() (*snet.UDPAddr, error) {
	netAddr, err := GetLocalHost()
	if err != nil {
		return nil, err
	}
	netAddr.Port, _ = freeport.GetFreePort()
	sciondConn, err := findSciond(context.Background())
	if err != nil {
		return nil, err
	}
	localIA, err := sciondConn.LocalIA(context.Background())
	if err != nil {
		return nil, err
	}
	sAddr := &snet.UDPAddr{
		IA:   localIA,
		Host: netAddr,
	}
	return sAddr, nil
}

func GetDefaultLocalAddrWithoutPort() (*snet.UDPAddr, error) {
	netAddr, err := GetLocalHost()
	if err != nil {
		return nil, err
	}
	sciondConn, err := findSciond(context.Background())
	if err != nil {
		return nil, err
	}
	localIA, err := sciondConn.LocalIA(context.Background())
	if err != nil {
		return nil, err
	}
	sAddr := &snet.UDPAddr{
		IA:   localIA,
		Host: netAddr,
	}
	return sAddr, nil
}
