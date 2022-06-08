package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/go-playground/assert"
	"go.uber.org/zap"
)

func TestPath(t *testing.T) {
	file := fileInfo{
		path: "/test/test",
	}

	assert.Equal(t, "/test/test", file.Path())
}

func TestListDirectory(t *testing.T) {
	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, 30*time.Second)
	dir := "./dir"
	sigChUsr := make(chan os.Signal, 1)
	signal.Notify(sigChUsr, syscall.SIGUSR1)
	depth := 1
	var wg sync.WaitGroup
	wg.Add(1)
	files := make(chan fileInfo)
	logger, _ := zap.NewProduction()
	ext := ".txt"
	go ListDirectory(ctx, dir, sigChUsr, depth, &wg, files, logger)
	go func() {
		wg.Wait()
		close(files)
	}()

	var resultFileList FileList
	resultFileList = append(resultFileList, TargetFile{
		Path: "dir/dir2/dir3/file.txt",
		Name: "file.txt",
	})

	var ret FileList
	for file := range files {
		if filepath.Ext(file.Name()) == ext {
			ret = append(ret, TargetFile{Name: file.Name(), Path: file.Path()})
		}
	}

	assert.Equal(t, resultFileList, ret)
}
