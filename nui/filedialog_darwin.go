package nui

/*
#include <stdlib.h>
#include "window.h"
*/
import "C"
import (
	"strings"
	"unsafe"
)

func parentWindowID(parent Window) C.int {
	if p, ok := parent.(*nativeWindow); ok && p != nil {
		return C.int(p.hwnd)
	}
	return C.int(-1)
}

// extensionsCSV flattens every filter's patterns into a single comma-separated
// list of bare extensions (NSOpenPanel/NSSavePanel have no Windows/GTK-style
// grouped, switchable filter, just one allowed-extensions list). "*" or "*.*"
// means "any file" and is dropped, since that's the default when no filter is
// set at all.
func extensionsCSV(filters []FileDialogFilter) *C.char {
	seen := make(map[string]bool)
	var exts []string
	for _, f := range filters {
		for _, p := range f.Patterns {
			ext := strings.TrimPrefix(strings.TrimPrefix(p, "*"), ".")
			if ext == "" || ext == "*" {
				continue
			}
			if !seen[ext] {
				seen[ext] = true
				exts = append(exts, ext)
			}
		}
	}
	if len(exts) == 0 {
		return nil
	}
	return C.CString(strings.Join(exts, ","))
}

func cStringOrNil(s string) *C.char {
	if s == "" {
		return nil
	}
	return C.CString(s)
}

func freeIfNotNil(p *C.char) {
	if p != nil {
		C.free(unsafe.Pointer(p))
	}
}

func openFileDialog(parent Window, opts OpenFileDialogOptions) ([]string, error) {
	title := cStringOrNil(opts.Title)
	dir := cStringOrNil(opts.DefaultDirectory)
	exts := extensionsCSV(opts.Filters)
	defer freeIfNotNil(title)
	defer freeIfNotNil(dir)
	defer freeIfNotNil(exts)

	allowMultiple := C.int(0)
	if opts.AllowMultiple {
		allowMultiple = 1
	}

	res := C.ShowOpenFileDialog(parentWindowID(parent), title, dir, exts, allowMultiple)
	if res == nil {
		return nil, nil
	}
	defer C.free(unsafe.Pointer(res))

	return strings.Split(C.GoString(res), "\n"), nil
}

func saveFileDialog(parent Window, opts SaveFileDialogOptions) (string, error) {
	title := cStringOrNil(opts.Title)
	dir := cStringOrNil(opts.DefaultDirectory)
	name := cStringOrNil(opts.DefaultFileName)
	exts := extensionsCSV(opts.Filters)
	defer freeIfNotNil(title)
	defer freeIfNotNil(dir)
	defer freeIfNotNil(name)
	defer freeIfNotNil(exts)

	res := C.ShowSaveFileDialog(parentWindowID(parent), title, dir, name, exts)
	if res == nil {
		return "", nil
	}
	defer C.free(unsafe.Pointer(res))

	return C.GoString(res), nil
}

func selectDirectoryDialog(parent Window, opts SelectDirectoryDialogOptions) (string, error) {
	title := cStringOrNil(opts.Title)
	dir := cStringOrNil(opts.DefaultDirectory)
	defer freeIfNotNil(title)
	defer freeIfNotNil(dir)

	res := C.ShowSelectDirectoryDialog(parentWindowID(parent), title, dir)
	if res == nil {
		return "", nil
	}
	defer C.free(unsafe.Pointer(res))

	return C.GoString(res), nil
}
