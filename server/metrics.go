package server

import (
	"fmt"
	"time"

	"github.com/netsys-lab/scion-path-discovery/packets"
)

type UploadConnMetrics struct {
	ConnId    string
	Remote    string
	SessionId string
	Metrics   packets.PathMetrics
	StartDate time.Time
	EndDate   time.Time
	Closed    bool
	Path      string
	Duration  time.Duration
}

func (m *UploadConnMetrics) GetCsv() string {
	secs := int64((m.Duration * time.Second) - 3*time.Second)
	bw := (m.Metrics.WrittenBytes * 8 / 1024 / 1024) / secs
	fmt.Printf("bw %d in secs %d\n", m.Metrics.WrittenBytes, secs)
	// id;remote;sessionId;uploadBw;startDate;endDate;closed;path;duration;
	return fmt.Sprintf("%s;%s;%s;%d;%s;%s;%t;%s;%d", m.ConnId, m.Remote, m.SessionId, bw, m.StartDate, m.EndDate, m.Closed, m.Path, m.Duration)
}

func (m *UploadConnMetrics) GetCsvHeader() string {
	return "id;remote;sessionId;uploadBw;startDate;endDate;closed;path;duration;"
}
