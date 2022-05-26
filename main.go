package main

//Исходники задания для первого занятия у других групп https://github.com/t0pep0/GB_best_go

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type TargetFile struct {
	Path string
	Name string
}

type FileList map[string]TargetFile

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
func ListDirectory(ctx context.Context, dir string, sigChUsr chan os.Signal, depth int, wg sync.WaitGroup, files chan<- fileInfo) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		return
	case <-sigChUsr:
		fmt.Printf("Dir: %s  Depth:", dir)
		return
	default:
		//По SIGUSR1 вывести текущую директорию и текущую глубину поиска
		time.Sleep(time.Second * 10)
		res, err := ioutil.ReadDir(dir)
		if err != nil {
			log.Printf("[ERROR] %v\n", err)
			return
		}
		for _, entry := range res {
			path := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				depth++
				wg.Add(1)
				go ListDirectory(ctx, path, sigChUsr, depth, wg, files) //Дополнительно: вынести в горутину
			} else {
				files <- fileInfo{entry, path}
			}
		}
	}
}

func FindFiles(ctx context.Context, ext string, sigChUsr chan os.Signal) (FileList, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	depth := 1
	files := make(chan fileInfo, 100)
	var wg sync.WaitGroup
	wg.Add(1)
	go ListDirectory(ctx, wd, sigChUsr, depth, wg, files)
	wg.Wait()
	fl := make(FileList, len(files))
	for file := range files {
		if filepath.Ext(file.Name()) == ext {
			fl[file.Name()] = TargetFile{
				Name: file.Name(),
				Path: file.Path(),
			}
		}
	}
	return fl, nil
}

func main() {
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
		res, err := FindFiles(ctx, wantExt, sigChUsr)
		if err != nil {
			log.Printf("Error on search: %v\n", err)
			os.Exit(1)
		}
		for _, f := range res {
			fmt.Printf("\tName: %s\t\t Path: %s\n", f.Name, f.Path)
		}
		waitCh <- struct{}{}
	}()
	go func() {
		<-sigCh
		log.Println("Signal received, terminate...")
		cancel()
	}()
	//Дополнительно: Ожидание всех горутин перед завершением
	<-waitCh
	log.Println("Done")
}
