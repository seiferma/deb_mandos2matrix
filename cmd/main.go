package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seiferma/mandos2mqtt/internal/logic"
	"github.com/seiferma/mandos2mqtt/internal/mandos"
	"github.com/seiferma/mandos2mqtt/internal/matrix"
)

const watchdogIntervalInSeconds = 60
const healthcheckTimeoutInSeconds = 10

func readConfigFilePaths() (initialConfigPath *string, runtimeConfigPath *string) {
	runtimeConfigPath = flag.String("runtime-config", "", "Path to the runtime configuration file (mandatory)")
	initialConfigPath = flag.String("initial-config", "", "Path to the initial configuration file")

	// Parse command line flags
	flag.Parse()

	// Validate that runtime config is provided (mandatory)
	if *runtimeConfigPath == "" {
		fmt.Fprintf(os.Stderr, "Error: runtime-config parameter is required\n")
		flag.Usage()
		os.Exit(1)
	}
	return
}

func createMatrixClient(initialConfigPath, runtimeConfigPath string) (*matrix.MatrixClient, error) {
	// create runtime config
	var runtimeConfig *matrix.RuntimeConfig
	var err error
	runtimeConfig, err = matrix.ParseRuntimeConfigFromFile(runtimeConfigPath)
	if err != nil {
		initialConfig, err := matrix.ParseInitialConfigFromFile(initialConfigPath)
		if err != nil {
			return nil, fmt.Errorf("could not read matrix client initial config: %w", err)
		}
		runtimeConfig, err = matrix.GenerateRuntimeConfig(*initialConfig)
		if err != nil {
			return nil, fmt.Errorf("could not create matrix client runtime config: %w", err)
		}
		err = runtimeConfig.SerializeToFile(runtimeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("could not write generated runtime config: %w", err)
		}
	}

	// create client
	matrixClient, err := matrix.NewMatrixClientByRuntimeConfig(runtimeConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create matrix client: %w", err)
	}

	// return result
	return matrixClient, nil
}

func waitForTerminationSignal() {
	receivedSignals := make(chan os.Signal, 1)
	signal.Notify(receivedSignals, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-receivedSignals
		log.Printf("Received signal %v", sig)
		done <- true
	}()
	<-done
}

func main() {
	// Define command line flags
	initialConfigPath, runtimeConfigPath := readConfigFilePaths()

	// create matrix client
	matrixClient, err := createMatrixClient(*initialConfigPath, *runtimeConfigPath)
	if err != nil {
		log.Fatalf("Could not create matrix client: %v", err)
	}
	defer matrixClient.Close()

	// create mandos client
	mandos, err := mandos.NewMandosCtl()
	if err != nil {
		log.Fatalf("Could not create mandos client: %v", err)
	}
	defer mandos.Close()

	// run matrix client
	matrixCtx, matrixCancel := context.WithCancel(context.Background())
	defer matrixCancel()
	go func() {
		err := matrixClient.Run(matrixCtx)
		if err != nil {
			log.Fatalf("matrix client failed: %v", err)
		}
	}()

	// send message function
	sendMessage := func(msg string) {
		err := matrixClient.SendMessage(msg)
		if err != nil {
			log.Fatalf("could not send message to matrix: %s", err)
		}
	}

	// create business logic
	logic := logic.CreateLogic(sendMessage, mandos)

	// wire business logic
	matrixClient.RegisterCallback(func(ctx context.Context, roomId, senderId, message string) {
		logic.HandleChatMessage(ctx, senderId, message)
	})
	mandos.SetOnClientDisabledHandler(func(clientPath string) {
		logic.HandleClientStatusChange(context.TODO(), clientPath, false)
	})
	mandos.SetOnClientEnabledHandler(func(clientPath string) {
		logic.HandleClientStatusChange(context.TODO(), clientPath, true)
	})
	mandos.SetOnClientRejectedHandler(func(clientPath, reason string) {
		logic.HandleClientRequestRejected(context.TODO(), clientPath, reason)
	})

	// start watchdog
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	startWatchdog(ctx, matrixClient, mandos)

	// wait for termination signal
	waitForTerminationSignal()
}

func startWatchdog(ctx context.Context, matrix *matrix.MatrixClient, mandos *mandos.MandosCtl) {
	ticker := time.NewTicker(watchdogIntervalInSeconds * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				err := watchdogTest(ctx, matrix, mandos)
				if err != nil {
					log.Fatalf("watchdog failed: %s", err)
				}
			}
		}
	}()
}

func watchdogTest(parentCtx context.Context, matrix *matrix.MatrixClient, mandos *mandos.MandosCtl) error {
	ctx, cancel := context.WithTimeout(parentCtx, healthcheckTimeoutInSeconds*time.Second)
	defer cancel()

	err := watchdogMandosTest(ctx, mandos)
	if err != nil {
		return fmt.Errorf("health check of mandos failed: %w", err)
	}

	err = watchdogMatrixTest(ctx, matrix)
	if err != nil {
		return fmt.Errorf("health check of matrix failed: %w", err)
	}

	return nil
}

func watchdogMandosTest(_ context.Context, mandos *mandos.MandosCtl) error {
	_, err := mandos.GetClients()
	if err != nil {
		return fmt.Errorf("health check of mandos failed: %w", err)
	}
	return nil
}

func watchdogMatrixTest(ctx context.Context, matrix *matrix.MatrixClient) error {
	_, err := matrix.GetRoomInfo(ctx)
	if err != nil {
		return fmt.Errorf("health check of matrix failed: %w", err)
	}
	return nil
}
