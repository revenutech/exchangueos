package sanctions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrInvalidSnapshot is returned when a snapshot cannot be written or read back as a complete, valid list.
var ErrInvalidSnapshot = errors.New("sanctions: invalid snapshot")

// Snapshot is one list as cmd/sanctions-refresher hands it to the screening pods: the entries a ListProvider
// accepted (parsed, validated, all of the snapshot's source, at least one), and when their download started.
//
// It is the hand-off between the refresher CronJob and the processes that call Match. A CronJob cannot fill another
// pod's memory, so the refresher writes one file per list (SnapshotPath) on a volume the screening pods mount
// read-only; they ReadSnapshot it and Load the entries into their own Cache. The file holds public list data only,
// never a screening query.
type Snapshot struct {
	Source Source `json:"source"`
	// FetchedAt is when the download that produced the entries started, in UTC. The data is at least this fresh.
	FetchedAt time.Time `json:"fetched_at"`
	Entries   []Entry   `json:"entries"`
}

// SnapshotPath is the file of one list's snapshot inside dir: «<dir>/<source in lower case>.json».
func SnapshotPath(dir string, s Source) string {
	return filepath.Join(dir, strings.ToLower(string(s))+".json")
}

func (s Snapshot) validate() error {
	switch {
	case !validSource(s.Source):
		return fmt.Errorf("%w: unknown source %q", ErrInvalidSnapshot, s.Source)
	case s.FetchedAt.IsZero():
		return fmt.Errorf("%w: %s snapshot without fetched_at", ErrInvalidSnapshot, s.Source)
	case len(s.Entries) == 0:
		return fmt.Errorf("%w: %s snapshot without entries", ErrInvalidSnapshot, s.Source)
	}
	for _, e := range s.Entries {
		if e.Source != s.Source {
			return fmt.Errorf("%w: %s snapshot holds an entry of %q", ErrInvalidSnapshot, s.Source, e.Source)
		}
		if err := e.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidSnapshot, err)
		}
	}
	return nil
}

// WriteSnapshot validates snap and replaces its file in dir atomically: the content goes to a temporary file in the
// same directory, is synced, and is renamed over SnapshotPath. A reader sees the previous snapshot or the new one,
// never half of either. An invalid snapshot is refused with ErrInvalidSnapshot and leaves the directory as it was.
//
// The file is created 0600: every exchangeos pod runs as the same non-root user (65532), so the screening pods can
// read it without widening the mode.
func WriteSnapshot(dir string, snap Snapshot) (err error) {
	if err := snap.validate(); err != nil {
		return err
	}
	snap.FetchedAt = snap.FetchedAt.UTC()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("sanctions: snapshot dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+strings.ToLower(string(snap.Source))+"-*.tmp")
	if err != nil {
		return fmt.Errorf("sanctions: snapshot temp file: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()
	if err := json.NewEncoder(tmp).Encode(snap); err != nil {
		return fmt.Errorf("sanctions: encode %s snapshot: %w", snap.Source, err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sanctions: sync %s snapshot: %w", snap.Source, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("sanctions: close %s snapshot: %w", snap.Source, err)
	}
	if err := os.Rename(tmp.Name(), SnapshotPath(dir, snap.Source)); err != nil {
		return fmt.Errorf("sanctions: publish %s snapshot: %w", snap.Source, err)
	}
	return nil
}

// ReadSnapshot reads the snapshot of source from dir and checks it as WriteSnapshot does. A missing file is reported
// with an error that wraps fs.ErrNotExist; anything else that is not a complete, valid snapshot of source wraps
// ErrInvalidSnapshot.
func ReadSnapshot(dir string, source Source) (Snapshot, error) {
	f, err := os.Open(SnapshotPath(dir, source))
	if err != nil {
		return Snapshot{}, fmt.Errorf("sanctions: open %s snapshot: %w", source, err)
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(f)
	var snap Snapshot
	if err := dec.Decode(&snap); err != nil {
		return Snapshot{}, fmt.Errorf("%w: decode %s snapshot: %w", ErrInvalidSnapshot, source, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Snapshot{}, fmt.Errorf("%w: %s snapshot has data after the document", ErrInvalidSnapshot, source)
	}
	if snap.Source != source {
		return Snapshot{}, fmt.Errorf("%w: file of %s holds a snapshot of %q", ErrInvalidSnapshot, source, snap.Source)
	}
	if err := snap.validate(); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}
