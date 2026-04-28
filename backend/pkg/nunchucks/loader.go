package nunchucks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LoaderResponse struct {
	Err string `json:"err"` // empty == no error
	Res string `json:"res"`
}

type Loader interface {
	TypeName() string
	Source(name string) LoaderResponse
	Read(name string) LoaderResponse
}

func ExtractComments(s string) string {
	return s
}

func PErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

type fileSystemLoader struct {
	base string
}

func FileSystemLoader(root string) Loader {
	if strings.TrimSpace(root) == "" {
		root = "views"
	}

	base, err := filepath.Abs(root)
	if err != nil {
		base = filepath.Clean(root)
	}
	base = filepath.Clean(base)

	return &fileSystemLoader{base: base}
}

func (l *fileSystemLoader) TypeName() string { return "file" }

func (l *fileSystemLoader) Source(name string) LoaderResponse {
	res := filepath.Clean(filepath.Join(l.base, name))

	rel, err := filepath.Rel(l.base, res)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		msg := fmt.Sprintf("No file found: %s", res)
		PErr(msg)
		return LoaderResponse{Err: msg, Res: res}
	}
	if _, err := os.Stat(res); err == nil {
		return LoaderResponse{Err: "", Res: res}
	}
	msg := fmt.Sprintf("No file found: %s", res)
	PErr(msg)
	return LoaderResponse{Err: msg, Res: res}
}

func (l *fileSystemLoader) Read(name string) LoaderResponse {
	file := l.Source(name)
	if file.Err != "" {
		return file
	}
	b, err := os.ReadFile(file.Res)
	if err != nil {
		msg := err.Error()
		PErr(msg)
		return LoaderResponse{Err: msg, Res: file.Res}
	}
	return LoaderResponse{Err: "", Res: ExtractComments(string(b))}
}
