// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"text/template"

	_ "embed"
)

var (
	//go:embed main.go.tmpl
	mainGoTemplateData string

	mainGoTemplate = template.Must(template.New("main.go").Parse(mainGoTemplateData))
)

type mainGoTemplateArgs struct {
	OpenAPIPackage string
	OpenAPITitle   string
}

func main() {
	var (
		openapiPackage string
		openapiTitle   string
	)

	flag.StringVar(&openapiPackage, "openapi-package", "", "Package containing the openapi definitions.")
	flag.StringVar(&openapiTitle, "openapi-title", "", "Title for the generated openapi json definition.")
	flag.Parse()

	if openapiPackage == "" {
		slog.Error("must specify openapi-package")
		os.Exit(1)
	}
	if openapiTitle == "" {
		slog.Error("must specify openapi-title")
		os.Exit(1)
	}

	if err := run(openapiPackage, openapiTitle); err != nil {
		slog.Error("Error running models-schema", "error", err)
	}
}

func run(openapiPackage, openapiTitle string) error {
	tmpFile, err := os.CreateTemp("", "models-schema-*.go")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil && !errors.Is(err, fs.ErrNotExist) {
			slog.Error("Error cleaning up temporary file", "error", err)
		}
	}()

	if err := mainGoTemplate.Execute(tmpFile, mainGoTemplateArgs{
		OpenAPIPackage: openapiPackage,
		OpenAPITitle:   openapiTitle,
	}); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	cmd := exec.Command("go", "run", tmpFile.Name())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running command: %w", err)
	}
	return nil
}
