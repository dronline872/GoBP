package main

//Исходники задания для первого занятия у других групп https://github.com/t0pep0/GB_best_go

import (
	"context"
	"fmt"
	"main/internal/findfiles"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

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
		res, err := findfiles.FindFiles(ctx, wantExt, sigChUsr, logger)
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
