package lecture

import (
	"bytes"
	"os"
	"syscall"
	"testing"
)

func TestExportLockedMarkdownRetainsVerifiedPDF(t *testing.T) {
	dir, vault, _ := exportFixture(t, "Lecture 3.md")
	plan, err := PlanExport(dir, ExportOptions{Vault: vault})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	path, err := syscall.UTF16PtrFromString(plan.Markdown)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(path, syscall.GENERIC_READ, syscall.FILE_SHARE_READ, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(h)
	if err := plan.Apply(); err == nil {
		t.Fatal("expected Markdown removal failure")
	}
	if _, err := os.Stat(plan.Markdown); err != nil {
		t.Fatal("lost locked Markdown", err)
	}
	if data, err := os.ReadFile(plan.Destination); err != nil || !bytes.Equal(data, testPDF) {
		t.Fatal("lost verified PDF", err)
	}
}
