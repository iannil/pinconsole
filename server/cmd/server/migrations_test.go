// 1k 测试：migrations 嵌入与版本解析 (P0-14)。
package main

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/iannil/marketing-monitor/migrations"
)

func TestParseMigrationVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"init", "000001_init.up.sql", 1, false},
		{"cobrowsing", "000002_cobrowsing.up.sql", 2, false},
		{"chat", "000003_chat.up.sql", 3, false},
		{"auth", "000004_auth.up.sql", 4, false},
		{"large version", "000099_some_change.up.sql", 99, false},
		{"missing prefix", "init.up.sql", 0, true},
		{"wrong format", "abc_def.up.sql", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMigrationVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMigrationVersion(%q) err = %v, wantErr = %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMigrationVersion(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestMigrationFiles_AllPresentInEmbed(t *testing.T) {
	// 验证 4 个 up + 4 个 down 都已嵌入
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	upCount := 0
	downCount := 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			upCount++
		} else if strings.HasSuffix(name, ".down.sql") {
			downCount++
		}
	}

	if upCount < 4 {
		t.Errorf("expected ≥4 up.sql files embedded, got %d", upCount)
	}
	if downCount < 4 {
		t.Errorf("expected ≥4 down.sql files embedded, got %d", downCount)
	}
}

func TestMigrationFiles_AllVersionsParseable(t *testing.T) {
	// 验证所有嵌入的 up.sql 文件名都能被 parseMigrationVersion 解析
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		if _, err := parseMigrationVersion(name); err != nil {
			t.Errorf("embedded migration %q not parseable: %v", name, err)
		}
	}
}
