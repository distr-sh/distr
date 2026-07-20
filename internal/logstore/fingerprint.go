package logstore

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/google/uuid"
)

// fingerprintNamespace is the fixed UUIDv5 namespace for derived log record IDs.
var fingerprintNamespace = uuid.NewSHA1(uuid.NameSpaceURL, []byte("https://distr.sh/logstore"))

// DeploymentLogRecordID derives the deterministic record ID from the record's stream
// labels, timestamp, severity and body. Loki does not store an ID per entry, so it is
// recomputed at query time. Determinism makes the ID stable across overlapping
// paginated/live fetches, which the frontend relies on for de-duplication and pinning.
// Severity is part of the input because it is structured metadata (not a stream label):
// two entries identical except severity are both stored by Loki and must not collide.
func DeploymentLogRecordID(record DeploymentLogRecord) uuid.UUID {
	var buf bytes.Buffer
	buf.WriteString("deployment")
	buf.WriteByte(0)
	buf.Write(record.DeploymentID[:])
	buf.Write(record.DeploymentRevisionID[:])
	writeString(&buf, record.Resource)
	writeTimestamp(&buf, record.Timestamp)
	writeString(&buf, record.Severity)
	buf.WriteString(record.Body)
	return uuid.NewSHA1(fingerprintNamespace, buf.Bytes())
}

// DeploymentTargetLogRecordID derives the deterministic record ID for a deployment
// target log record. See DeploymentLogRecordID for the rationale.
func DeploymentTargetLogRecordID(record DeploymentTargetLogRecord) uuid.UUID {
	var buf bytes.Buffer
	buf.WriteString("deployment_target")
	buf.WriteByte(0)
	buf.Write(record.DeploymentTargetID[:])
	writeTimestamp(&buf, record.Timestamp)
	writeString(&buf, record.Severity)
	buf.WriteString(record.Body)
	return uuid.NewSHA1(fingerprintNamespace, buf.Bytes())
}

func writeString(buf *bytes.Buffer, s string) {
	buf.WriteString(s)
	buf.WriteByte(0)
}

func writeTimestamp(buf *bytes.Buffer, t time.Time) {
	_ = binary.Write(buf, binary.BigEndian, t.UnixNano())
}
