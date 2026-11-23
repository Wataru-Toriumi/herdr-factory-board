package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// 同じ層と、ここで先に並ぶ層への依存を許可する。src の直下は起動処理に使う。
var layers = []string{"domain", "infrastructure", "application", "tui", "main"}

func packageLayer(path string) string {
	if path == "src" {
		return "main"
	}
	rel, ok := strings.CutPrefix(path, "src/")
	layer, _, _ := strings.Cut(rel, "/")
	if !ok || layer == "main" || !slices.Contains(layers, layer) {
		return ""
	}
	return layer
}

func TestPackageDependencies(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-json=ImportPath,Imports,Standard,Module", "./...")
	cmd.Dir = ".."
	output, err := cmd.Output()
	if err != nil {
		if failure, ok := err.(*exec.ExitError); ok {
			t.Log(string(failure.Stderr))
		}
		t.Fatal(err)
	}

	type goPackage struct {
		ImportPath string
		Imports    []string
		Standard   bool
		Module     *struct {
			Main bool
			Path string
		}
	}
	standard := make(map[string]bool)
	var local []goPackage
	dec := json.NewDecoder(bytes.NewReader(output))
	for {
		var pkg goPackage
		if err := dec.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if pkg.Standard {
			standard[pkg.ImportPath] = true
		} else if pkg.Module != nil && pkg.Module.Main {
			local = append(local, pkg)
		}
	}
	for _, pkg := range local {
		module := pkg.Module.Path
		name := strings.TrimPrefix(pkg.ImportPath, module+"/")
		if name != "src" && !strings.HasPrefix(name, "src/") {
			continue
		}
		layer := packageLayer(name)
		if layer == "" {
			t.Errorf("%s: パッケージを src 配下のレイヤーに配置してください", name)
			continue
		}
		for _, dep := range pkg.Imports {
			target := strings.TrimPrefix(dep, module+"/")
			localImport := target != dep || dep == module
			if localImport {
				targetLayer := packageLayer(target)
				if targetLayer == "" || slices.Index(layers, targetLayer) > slices.Index(layers, layer) {
					t.Errorf("%s → %s: %s 層から参照できるのは同じ層と下位層です", name, target, layer)
				}
			}
			switch name {
			case "src/tui/taskboard":
				if localImport && packageLayer(target) != "domain" {
					t.Errorf("%s → %s: タスクボードの表示状態が参照できるモジュール内の層は domain だけです", name, target)
				}
			case "src/domain/schedule", "src/infrastructure/herdr":
				if !standard[dep] {
					t.Errorf("%s → %s: 標準ライブラリだけを利用できます", name, dep)
				}
			case "src/domain/board":
				if localImport && target != "src/domain/schedule" {
					t.Errorf("%s → %s: モジュール内で利用できるのは domain/schedule だけです", name, target)
				}
			}
			tuiLibrary := strings.HasPrefix(dep+"/", "github.com/rivo/tview/") || strings.HasPrefix(dep+"/", "github.com/gdamore/tcell/v2/")
			if name != "src/tui" && tuiLibrary {
				t.Errorf("%s → %s: TUI ライブラリは src/tui で利用してください", name, dep)
			}
		}
	}
}
