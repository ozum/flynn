package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"

	matrix "github.com/jvatic/asset-matrix-go"
)

func main() {
	installerSrcDir := os.Getenv("INSTALLER_SRC_DIR")
	if installerSrcDir == "" {
		installerSrcDir = "./lib/installer"
	}
	prevManifest := parsePrevManifest()
	m := matrix.New(&matrix.Config{
		Paths: []*matrix.AssetRoot{
			&matrix.AssetRoot{
				GitRepo:   "git://github.com/jvatic/marbles-js.git",
				GitBranch: "master",
				GitRef:    "50fe2ed6d530d9b695b98a775dcc28ec7c9478bc",
				Path:      "src",
			},
			&matrix.AssetRoot{
				GitRepo:   "git://github.com/flynn/flynn-dashboard-web-icons.git",
				GitBranch: "master",
				GitRef:    "a03e56dbe3dd6f073a7ab212ae0912a355a85ab0",
				Path:      "assets",
			},
			&matrix.AssetRoot{
				Path: filepath.Join(installerSrcDir, "images"),
			},
			&matrix.AssetRoot{
				Path: "./lib/javascripts",
			},
			&matrix.AssetRoot{
				Path: "./lib/stylesheets",
			},
			&matrix.AssetRoot{
				Path: "./lib/images",
			},
			&matrix.AssetRoot{
				Path: "./vendor/javascripts",
			},
			&matrix.AssetRoot{
				Path: "./vendor/stylesheets",
			},
			&matrix.AssetRoot{
				Path: "./vendor/fonts",
			},
		},
		Outputs: []string{
			"dashboard.js",
			"dashboard-*.js",
			"dashboard.scss",
			"moment.js",
			"es6promise.js",
			"react.js",
			"react.dev.js",
			"*.png",
			"*.eot",
			"*.svg",
			"*.ttf",
			"*.woff",
		},
		OutputDir:      "./build/assets",
		AssetURLPrefix: "/assets/",
	})
	if err := m.Build(); err != nil {
		log.Fatal(err)
	}
	if err := compileTemplate(m.Manifest); err != nil {
		log.Fatal(err)
	}
	cleanupAssets(prevManifest, m.Manifest)
}

func parsePrevManifest() *matrix.Manifest {
	prevManifest := &matrix.Manifest{
		Assets: make(map[string]string, 0),
	}
	file, err := os.Open("./build/assets/manifest.json")
	if err != nil {
		return prevManifest
	}
	dec := json.NewDecoder(file)
	dec.Decode(&prevManifest)
	return prevManifest
}

func compileTemplate(manifest *matrix.Manifest) error {
	type TemplateData struct {
		Development bool
	}
	tmplHTML, err := readTemplate()
	if err != nil {
		return err
	}
	tmpl, err := template.New("dashboard.html").Funcs(template.FuncMap{
		"assetPath": func(p string) string {
			return "/assets/" + manifest.Assets[p]
		},
	}).Parse(tmplHTML)
	if err != nil {
		return err
	}
	file, err := os.Create("./build/dashboard.html")
	defer file.Close()
	if err != nil {
		return err
	}
	return tmpl.Execute(file, &TemplateData{
		Development: os.Getenv("ENVIRONMENT") == "development",
	})
}

func readTemplate() (string, error) {
	var buf bytes.Buffer
	file, err := os.Open("./lib/dashboard.html.tmpl")
	defer file.Close()
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(&buf, file); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func cleanupAssets(prevManifest *matrix.Manifest, manifest *matrix.Manifest) {
	for logicalPath, path := range prevManifest.Assets {
		if manifest.Assets[logicalPath] == path {
			continue
		}
		p := filepath.Join("./build/assets", path)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		os.Remove(p)
	}
}
