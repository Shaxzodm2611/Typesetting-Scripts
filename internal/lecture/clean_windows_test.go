package lecture

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestInspectWindowsJunction(t *testing.T) {
	d := fixture(t)
	outside := t.TempDir()
	put(t, filepath.Join(outside, "sentinel"), []byte("keep"))
	junction := filepath.Join(d, "junction")
	output, err := exec.Command("cmd", "/c", "mklink", "/J", junction, outside).CombinedOutput()
	if err != nil {
		t.Fatalf("create ordinary directory junction: %v: %s", err, output)
	}
	defer os.Remove(junction)
	if f, err := Inspect(d); err == nil {
		f.Close()
		t.Fatal("accepted junction")
	}
	unchangedSource(t, d)
	b, _ := os.ReadFile(filepath.Join(outside, "sentinel"))
	if string(b) != "keep" {
		t.Fatal("changed external file")
	}
}
func TestFinalizeWindowsLockedFile(t *testing.T) {
	d := fixture(t)
	locked := filepath.Join(d, "locked.bin")
	put(t, locked, []byte("locked"))
	p, err := syscall.UTF16PtrFromString(locked)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ, syscall.FILE_SHARE_READ, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(h)
	f, err := Inspect(d)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = f.Apply(); err == nil {
		t.Fatal("expected locked-file failure")
	}
	b, _ := os.ReadFile(filepath.Join(d, "lecture.pdf"))
	if !bytes.Equal(b, testPDF) {
		t.Fatal("lost PDF")
	}
	if _, err := os.Stat(filepath.Join(d, ".notes.json")); err != nil {
		t.Fatal("lost marker")
	}
}
