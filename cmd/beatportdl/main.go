package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"unspok3n/beatportdl/config"
	"unspok3n/beatportdl/internal/beatport"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
)

const (
	configFilename = "beatportdl-config.yml"
	cacheFilename  = "beatportdl-credentials.json"
	errorFilename  = "beatportdl-err.log"
)

var (
	outputDirectory string
)

type application struct {
	config      *config.AppConfig
	logFile     *os.File
	logWriter   io.Writer
	ctx         context.Context
	wg          sync.WaitGroup
	downloadSem chan struct{}
	globalSem   chan struct{}
	pbp         *mpb.Progress

	urls             []string
	activeFiles      map[string]struct{}
	activeFilesMutex sync.RWMutex

	bp *beatport.Beatport
	bs *beatport.Beatport
}

func main() {
	config, cachePath, err := Setup()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	app := &application{
		ctx:         ctx,
		config:      config,
		logWriter:   os.Stdout,
		globalSem:   make(chan struct{}, config.MaxGlobalWorkers),
		downloadSem: make(chan struct{}, config.MaxDownloadWorkers),
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		<-sigCh

		if len(app.urls) > 0 {
			app.LogInfo("Shutdown signal received. Waiting for download workers to finish")
			cancel()

			<-sigCh
		}

		os.Exit(0)
	}()

	if config.WriteErrorLog {
		logFilePath, _, err := FindErrorLogFile()
		if err != nil {
			fmt.Println(err.Error())
			Pause()
		}
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		app.logFile = f
		defer f.Close()
	}

	auth := beatport.NewAuth(config.Username, config.Password, cachePath)
	app.bp = beatport.New(beatport.StoreBeatport, config.Proxy, auth)
	app.bs = beatport.New(beatport.StoreBeatsource, config.Proxy, auth)

	if err := auth.LoadCache(); err != nil {
		if err := auth.Init(app.bp); err != nil {
			app.FatalError("beatport", err)
		}
	}

	app.pbp = mpb.New(mpb.WithAutoRefresh(), mpb.WithOutput(color.Output))

	rootCmd := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			// panic("TODO")
		},
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}

	// quitFlag := flag.Bool("q", false, "Quit the main loop after finishing")
	//
	// flag.Parse()
	// inputArgs := flag.Args()
	//
	// for _, arg := range inputArgs {
	// 	if strings.HasSuffix(arg, ".txt") {
	// 		app.parseTextFile(arg)
	// 	} else {
	// 		app.urls = append(app.urls, arg)
	// 	}
	// }
	//
	// for {
	// 	if len(app.urls) == 0 {
	// 		app.mainPrompt()
	// 	}
	//
	// 	app.logWriter = app.pbp
	// 	app.activeFiles = make(map[string]struct{}, len(app.urls))
	//
	// 	for _, url := range app.urls {
	// 		app.globalWorker(func() {
	// 			app.handleUrl(url)
	// 		})
	// 	}
	//
	// 	app.wg.Wait()
	// 	app.pbp.Shutdown()
	//
	// 	if *quitFlag || ctx.Err() != nil {
	// 		break
	// 	}
	//
	// 	app.urls = []string{}
	// }
}
