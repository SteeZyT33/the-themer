package wt_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/kylesnowschwartz/the-themer/adapter"
	_ "github.com/kylesnowschwartz/the-themer/adapter/wt"
	"github.com/kylesnowschwartz/the-themer/palette"
)

func TestGenerate_OracleBleu(t *testing.T) {
	cfg, err := palette.Load("../../testdata/bleu.toml")
	if err != nil {
		t.Fatalf("Load bleu.toml: %v", err)
	}

	wt := adapter.ByName([]string{"windows-terminal"})
	if len(wt) != 1 {
		t.Fatalf("expected 1 windows-terminal adapter, got %d", len(wt))
	}

	got, err := wt[0].Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expected, err := os.ReadFile("../../testdata/expected/windows-terminal/bleu.json")
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
		if a.Name() == "windows-terminal" {
			found = true
			if a.DirName() != "windows-terminal" {
				t.Errorf("DirName: got %q, want %q", a.DirName(), "windows-terminal")
			}
			if a.FileName("bleu") != "bleu.json" {
				t.Errorf("FileName: got %q, want %q", a.FileName("bleu"), "bleu.json")
			}
		}
	}
	if !found {
		t.Fatal("windows-terminal adapter not registered")
	}
}
