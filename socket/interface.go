package socket
// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import "net"

type Socket interface {
	Listen(addr string) (*net.Listener, error)
	Dial(addr string, index int) (net.Conn, error)
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Accept() (net.Conn, error)
}

func NewSocket(network string) Socket {
	if network == "tcp" {
		return NewTCPSocket()
	}

	if network == "scion" {
		return NewSCIONSocket()
	}

	return nil
}
