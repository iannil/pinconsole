// 1t 测试:migrations 包 embed 文件完整性。
package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestFiles_NotEmpty(t *testing.T) {
	entries, err := fs.ReadDir(Files, ".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) == 0 {
		t.Errorf("embedded migrations should not be empty")
	}
}

func TestFiles_HasUpAndDown(t *testing.T) {
	entries, err := fs.ReadDir(Files, ".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	upCount, downCount := 0, 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			upCount++
		}
		if strings.HasSuffix(name, ".down.sql") {
			downCount++
		}
	}
	if upCount == 0 {
		t.Errorf("no .up.sql files embedded")
	}
	if downCount == 0 {
		t.Errorf("no .down.sql files embedded")
	}
	// up/down 数量应一致(每个 migration 一对)
	if upCount != downCount {
		t.Errorf("up/down count mismatch: up=%d down=%d", upCount, downCount)
	}
}
