package agent

import (
	"github.com/hashicorp/logutils"
	"log"
)

// logStream is used to stream logs to a client over IPC
type logStream struct {
	client streamClient
	filter *logutils.LevelFilter
	logCh  chan string
	logger *log.Logger
	seq    uint64
}

func newLogStream(client streamClient, filter *logutils.LevelFilter,
	seq uint64, logger *log.Logger) *logStream {
	ls := &logStream{
		client: client,
		filter: filter,
		logCh:  make(chan string, 512),
		logger: logger,
		seq:    seq,
	}
	go ls.stream()
	return ls
}

func (ls *logStream) HandleLog(l string) {
	// Check the log level
	if !ls.filter.Check([]byte(l)) {
		return
	}

	// Do a non-blocking send
	select {
	case ls.logCh <- l:
	default:
		ls.logger.Printf("[WARN] Dropping logs to %v", ls.client)
	}
}

func (ls *logStream) Stop() {
	close(ls.logCh)
}

func (ls *logStream) stream() {
	header := responseHeader{Seq: ls.seq, Error: ""}
	rec := logRecord{Log: ""}

	for line := range ls.logCh {
		rec.Log = line
		if err := ls.client.Send(&header, &rec); err != nil {
			ls.logger.Printf("[ERR] Failed to stream log to %v: %v",
				ls.client, err)
			return
		}
	}
}
