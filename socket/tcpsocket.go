package socket
// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import (
	"net"
	"unsafe"
)

type TCPSocket struct {
	conn       *net.Conn
	listener   *net.TCPListener
	localAddr  string
	remoteAddr string
	address    *net.TCPAddr
}

func NewTCPSocket() *TCPSocket {
	return &TCPSocket{}
}

func (s *TCPSocket) Listen(addr string) (*net.Listener, error) {
	localAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.localAddr = addr
	s.address = localAddr
	listener, err := net.ListenTCP("tcp", localAddr)
	if err != nil {
		return nil, err
	}
	s.listener = listener
	l := (*net.Listener)(unsafe.Pointer(s.listener))
	return l, err
}
func (s *TCPSocket) Dial(addr string, index int) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.conn = &conn
	return *s.conn, nil
}

func (s *TCPSocket) Read(bytes []byte) (int, error) {
	return (*s.conn).Read(bytes)
}

func (s *TCPSocket) Write(bytes []byte) (int, error) {
	return (*s.conn).Write(bytes)
}

func (s *TCPSocket) Accept() (net.Conn, error) {
	return s.listener.Accept()
}
