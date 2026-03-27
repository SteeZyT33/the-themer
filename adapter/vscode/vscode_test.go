package vscode_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/kylesnowschwartz/the-themer/adapter"
	_ "github.com/kylesnowschwartz/the-themer/adapter/vscode"
	"github.com/kylesnowschwartz/the-themer/palette"
)

func TestGenerate_OracleBleu(t *testing.T) {
	cfg, err := palette.Load("../../testdata/bleu.toml")
	if err != nil {
		t.Fatalf("Load bleu.toml: %v", err)
	}

	vscode := adapter.ByName([]string{"vscode"})
	if len(vscode) != 1 {
		t.Fatalf("expected 1 vscode adapter, got %d", len(vscode))
	}

	got, err := vscode[0].Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expected, err := os.ReadFile("../../testdata/expected/vscode/bleu.jsonc")
	if err != nil {
		t.Fatalf("reading expected fixture: %v", err)
	}

	if !bytes.Equal(got, expected) {
		t.Errorf("output differs from oracle\n--- got ---\n%s\n--- want ---\n%s", got, expected)
	}
}

func TestAdapterRegistration(t *testing.T) {
	all := adapter.All()

	found := false
	for _, a := range all {
		if a.Name() == "vscode" {
			found = true
			if a.DirName() != "vscode" {
				t.Errorf("DirName: got %q, want %q", a.DirName(), "vscode")
			}
			if a.FileName("bleu") != "bleu.jsonc" {
				t.Errorf("FileName: got %q, want %q", a.FileName("bleu"), "bleu.jsonc")
			}
		}
	}
	if !found {
		t.Fatal("vscode adapter not registered")
	}
}
