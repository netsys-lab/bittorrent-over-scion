module github.com/veggiedefender/torrent-client

go 1.13

require (
	github.com/HdrHistogram/hdrhistogram-go v1.1.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/gopacket v1.1.19 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/inconshreveable/log15 v0.0.0-20201112154412-8562bdadbbac // indirect
	github.com/jackpal/bencode-go v1.0.0
	github.com/lucas-clemente/quic-go v0.19.2
	github.com/marten-seemann/qtls-go1-15 v0.1.4 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/netsec-ethz/scion-apps v0.3.0
	github.com/netsys-lab/scion-path-discovery v0.0.0-20210920082250-82e0785b5f6c
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pelletier/go-toml v1.9.3 // indirect
	github.com/prometheus/common v0.29.0 // indirect
	github.com/scionproto/scion v0.6.0
	github.com/sirupsen/logrus v1.6.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/uber/jaeger-client-go v2.29.1+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	go.uber.org/atomic v1.8.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a // indirect
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1 // indirect
	google.golang.org/genproto v0.0.0-20210614182748-5b3b54cad159 // indirect
	google.golang.org/grpc/examples v0.0.0-20210615210310-549c53a90c2a // indirect
	zombiezen.com/go/capnproto2 v2.18.2+incompatible // indirect
)

// replace github.com/netsec-ethz/scion-apps => /home/marten/go/src/github.com/git.deinstapel.de/scion-apps

replace github.com/netsys-lab/scion-path-discovery => /home/marten/go/src/github.com/martenwallewein/scion-path-discovery
