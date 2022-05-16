package util

import (
	"net"
	"time"

	"github.com/lucas-clemente/quic-go"
)

type WrappedStream struct {
	Stream quic.Stream
	Local  string
	Remote string
}

func (w *WrappedStream) Read(b []byte) (n int, err error) {
	return w.Stream.Read(b)
}

func (w *WrappedStream) Write(b []byte) (n int, err error) {
	return w.Stream.Write(b)
}

func (w *WrappedStream) Close() error {
	return w.Stream.Close()
}

func (w *WrappedStream) LocalAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("TCP", w.Local)
	return addr
}

func (w *WrappedStream) RemoteAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("TCP", w.Remote)
	return addr
}

func (w *WrappedStream) SetDeadline(t time.Time) error {
	return w.Stream.SetDeadline(t)
}

func (w *WrappedStream) SetReadDeadline(t time.Time) error {
	return w.Stream.SetReadDeadline(t)
}

func (w *WrappedStream) SetWriteDeadline(t time.Time) error {
	return w.Stream.SetWriteDeadline(t)
}
