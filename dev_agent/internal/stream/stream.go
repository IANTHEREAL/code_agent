package stream

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"dev_agent/internal/logx"
)

// JSONStreamer emits NDJSON events to stdout when enabled.
type JSONStreamer struct {
	enabled  bool
	threadID string
	seq      uint64
	writer   io.Writer
	mu       sync.Mutex
}

func NewJSONStreamer(enabled bool) *JSONStreamer {
	stream := &JSONStreamer{
		enabled:  enabled,
		threadID: newThreadID(),
		writer:   os.Stdout,
	}
	return stream
}

func (s *JSONStreamer) Enabled() bool {
	return s != nil && s.enabled
}

func (s *JSONStreamer) ThreadID() string {
	if s == nil {
		return ""
	}
	return s.threadID
}

func (s *JSONStreamer) Emit(eventType string, payload map[string]any) {
	if s == nil || !s.enabled || eventType == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	obj := map[string]any{
		"type":      eventType,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"sequence":  s.seq,
		"thread_id": s.threadID,
	}
	for k, v := range payload {
		if k == "" || v == nil {
			continue
		}
		obj[k] = v
	}
	data, err := json.Marshal(obj)
	if err != nil {
		logx.Errorf("failed to marshal stream event %q: %v", eventType, err)
		return
	}
	if _, err := fmt.Fprintln(s.writer, string(data)); err != nil {
		logx.Errorf("failed to write stream event %q: %v", eventType, err)
	}
}

func newThreadID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("thread-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(buf[0:4]),
		hex.EncodeToString(buf[4:6]),
		hex.EncodeToString(buf[6:8]),
		hex.EncodeToString(buf[8:10]),
		hex.EncodeToString(buf[10:16]),
	)
}
