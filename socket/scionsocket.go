package socket

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"sync"
	"unsafe"

	quic "github.com/lucas-clemente/quic-go"
	"github.com/netsec-ethz/scion-apps/pkg/appnet"
	"github.com/netsec-ethz/scion-apps/pkg/appnet/appquic"
	"github.com/scionproto/scion/go/lib/addr"
	"github.com/scionproto/scion/go/lib/snet"
)

var quicInit sync.Once
var scionInit sync.Once
var (
	// Don't verify the server's cert, as we are not using the TLS PKI.
	TLSCfg = &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h3"}}
)

type squicStreamWrapper struct {
	quic.Stream
	local, remote func() net.Addr
}

func (w squicStreamWrapper) LocalAddr() net.Addr {
	return w.local()
}
func (w squicStreamWrapper) RemoteAddr() net.Addr {
	return w.remote()
}

type SCIONSocket struct {
	conn       net.Conn
	listener   *quic.Listener
	localAddr  string
	remoteAddr string
	address    *net.TCPAddr
}

func NewSCIONSocket() *SCIONSocket {
	return &SCIONSocket{}
}

func (s *SCIONSocket) Listen(addr string) (*net.Listener, error) {
	if err := InitSQUICCerts(); err != nil {
		return nil, err
	}

	fmt.Printf("Listen SCION for %s", addr)
	address, err := snet.ParseUDPAddr(addr)
	if err != nil {
		return nil, err
	}

	udpAddr := net.UDPAddr{
		IP:   address.Host.IP,
		Port: address.Host.Port,
	}

	// conn, err := appnet.Listen(&udpAddr)
	conn, err := appnet.ListenPort(uint16(udpAddr.Port))
	if err != nil {
		return nil, err
	}

	qConn, err := quic.Listen(conn, TLSCfg, &quic.Config{KeepAlive: true})
	if err != nil {
		return nil, err
	}

	s.localAddr = addr
	s.listener = &qConn
	l := (*net.Listener)(unsafe.Pointer(s.listener))
	return l, err
}
func (s *SCIONSocket) Dial(addr string, index int) (net.Conn, error) {

	if err := InitSQUICCerts(); err != nil {
		return nil, err
	}

	address, err := snet.ParseUDPAddr(addr)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Dial SCION to addr %s\n", address)

	sess, err := appquic.DialAddr(address, appnet.MangleSCIONAddr(addr), TLSCfg, &quic.Config{
		KeepAlive: true,
	})

	if err != nil {
		return nil, err
	}

	conn, err := sess.OpenStreamSync(context.Background())
	if err != nil {
		return nil, err
	}

	qsconn := squicStreamWrapper{
		conn,
		sess.LocalAddr,
		sess.RemoteAddr,
	}

	s.conn = qsconn
	return s.conn, nil
}

func (s *SCIONSocket) Read(bytes []byte) (int, error) {
	return s.conn.Read(bytes)
}

func (s *SCIONSocket) Write(bytes []byte) (int, error) {
	return s.conn.Write(bytes)
}

func (s *SCIONSocket) Accept() (net.Conn, error) {
	qconn, err := (*s.listener).Accept(context.Background())
	if err != nil {
		return nil, err
	}

	stream, err := qconn.AcceptStream(context.Background())

	return squicStreamWrapper{
		stream,
		qconn.LocalAddr,
		qconn.RemoteAddr,
	}, nil
}

// GetDefaultDispatcher returns the default SCION dispatcher service
//func GetDefaultDispatcher() reliable.DispatcherService {
//	return reliable.NewDispatcherService("")
//}

// InitSQUICCerts reads certificate files from the os environment and Initializes the scion QUIC layer.
func InitSQUICCerts() error {
	var initErr error
	quicInit.Do(func() {
		// initErr = squic.Init(os.Getenv("SCION_CERT_KEY_FILE"), os.Getenv("SCION_CERT_FILE"))
		cert, err := tls.LoadX509KeyPair(os.Getenv("SCION_CERT_FILE"), os.Getenv("SCION_CERT_KEY_FILE"))
		initErr = err
		TLSCfg.Certificates = []tls.Certificate{cert}
	})
	return initErr
}

func InitScion(myAddr addr.IA) error {
	var initErr error
	scionInit.Do(func() {
		// initErr = snet.Init(myAddr, sciondPath, dispatcher)
	})
	return initErr
}
