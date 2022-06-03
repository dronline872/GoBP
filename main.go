package main

//Исходники задания для первого занятия у других групп https://github.com/t0pep0/GB_best_go

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type TargetFile struct {
	Path string
	Name string
}

type FileList []TargetFile

type FileInfo interface {
	os.FileInfo
	Path() string
}

type fileInfo struct {
	os.FileInfo
	path string
}

func (fi fileInfo) Path() string {
	return fi.path
}

//Ограничить глубину поиска заданым числом, по SIGUSR2 увеличить глубину поиска на +2
func ListDirectory(ctx context.Context, dir string, sigChUsr chan os.Signal, depth int, wg *sync.WaitGroup, files chan<- fileInfo, logger *zap.Logger) {
	defer wg.Done()
	res, err := ioutil.ReadDir(dir)
	if err != nil {
		logger.Error(fmt.Sprintf("Error on ReadDir: %v\n", err))
		return
	}

	for _, entry := range res {
		select {
		case <-ctx.Done():
			return
		case <-sigChUsr:
			logger.Info("Current directory:",
				zap.String("Dir", dir),
				zap.Int("Depth", depth),
			)
			return
		default:
			path := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				depth++
				wg.Add(1)
				go ListDirectory(ctx, path, sigChUsr, depth, wg, files, logger) //Дополнительно: вынести в горутину
			} else {
				files <- fileInfo{entry, path}
			}
		}
	}
}

func FindFiles(ctx context.Context, ext string, sigChUsr chan os.Signal, logger *zap.Logger) (FileList, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	depth := 1
	files := make(chan fileInfo, 100)
	var wg sync.WaitGroup
	wg.Add(1)
	go ListDirectory(ctx, wd, sigChUsr, depth, &wg, files, logger)

	go func() {
		wg.Wait()
		close(files)
	}()

	var ret FileList
	for file := range files {
		if filepath.Ext(file.Name()) == ext {
			ret = append(ret, TargetFile{Name: file.Name(), Path: file.Path()})
		}
	}

	return ret, nil
}

func main() {
	logger, _ := zap.NewProduction()
	const wantExt = ".go"
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sigChUsr := make(chan os.Signal, 1)
	signal.Notify(sigChUsr, syscall.SIGUSR1)

	waitCh := make(chan struct{})
	go func() {
		res, err := FindFiles(ctx, wantExt, sigChUsr, logger)
		if err != nil {
			logger.Error(fmt.Sprintf("Error on search: %v\n", err))
			os.Exit(1)
		}

		for _, f := range res {
			logger.Info("Result:",
				zap.String("Name", f.Name),
				zap.String("Path", f.Path),
			)
		}
		waitCh <- struct{}{}
	}()
	go func() {
		<-sigCh
		logger.Warn("Signal received, terminate...")
		cancel()
	}()
	//Дополнительно: Ожидание всех горутин перед завершением
	<-waitCh
	logger.Info("Done")
}
