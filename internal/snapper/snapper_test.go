package snapper

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseCSV(t *testing.T) {
	data := []byte("number,type,description\n0,single,current\n1,pre,\"zypp, install\"\n")

	recs, err := parseCSV(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}

	if got := recs[1]["description"]; got != "zypp, install" {
		t.Errorf("description = %q", got)
	}
}

// fakeSnapper installs a shell script as the snapper binary for the test.
func fakeSnapper(t *testing.T, script string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "snapper")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}

	old := Bin
	Bin = path
	t.Cleanup(func() { Bin = old })
}

func TestListSkipsCurrent(t *testing.T) {
	fakeSnapper(t, `cat <<'EOF'
number,type,pre-number,date,user,cleanup,description
0,single,,,root,,current
1,pre,,2026-01-01,root,number,before
2,post,1,2026-01-01,root,number,after
EOF
`)

	snaps, err := List(context.Background(), "root")
	if err != nil {
		t.Fatal(err)
	}

	if len(snaps) != 2 || snaps[0].Number != 1 || snaps[1].PreNumber != "1" {
		t.Fatalf("unexpected snapshots: %+v", snaps)
	}
}

func TestErrorUsesStderr(t *testing.T) {
	fakeSnapper(t, `echo "Unknown config." >&2; exit 1`)

	_, err := Configs(context.Background())
	if err == nil || err.Error() != "snapper list-configs: Unknown config." {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteRejectsCurrent(t *testing.T) {
	if err := Delete(context.Background(), "root", 0); err == nil {
		t.Fatal("expected error for snapshot 0")
	}
}
