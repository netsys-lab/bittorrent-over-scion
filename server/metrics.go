package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/netsys-lab/scion-path-discovery/packets"
)

type UploadConnMetrics struct {
	ConnId    string
	Remote    string
	Local     string
	SessionId string
	Metrics   packets.PathMetrics
	StartDate time.Time
	EndDate   time.Time
	Closed    bool
	Path      string
	Duration  time.Duration
	ExportTo  string
}

type jsonMetrics struct {
	ConnId    string
	Remote    string
	SessionId string
	Local     string
	StartDate time.Time
	EndDate   time.Time
	Closed    bool
	Path      string
	Duration  time.Duration
}

func (m *UploadConnMetrics) GetCsv() string {
	secs := int64((m.Duration * time.Second) - 3*time.Second)
	bw := (m.Metrics.WrittenBytes * 8 / 1024 / 1024) / secs
	// id;remote;sessionId;uploadBw;startDate;endDate;closed;path;duration;
	return fmt.Sprintf("%s;%s;%s;%d;%s;%s;%t;%s;%d;%s", m.ConnId, m.Remote, m.SessionId, bw, m.StartDate, m.EndDate, m.Closed, m.Path, m.Duration, m.Local)
}

func (m *UploadConnMetrics) GetCsvHeader() string {
	return "id;remote;sessionId;uploadBw;startDate;endDate;closed;path;duration;local;"
}

func (m *UploadConnMetrics) GetJSON() []byte {
	jmetrics := jsonMetrics{
		ConnId:    m.ConnId,
		Remote:    m.Remote,
		SessionId: m.SessionId,
		StartDate: m.StartDate,
		EndDate:   m.EndDate,
		Closed:    m.Closed,
		Path:      m.Path,
		Local:     m.Local,
		Duration:  m.Duration,
	}

	data, _ := json.Marshal(jmetrics)
	return data
}
