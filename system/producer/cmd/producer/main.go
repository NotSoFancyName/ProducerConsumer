package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/NotSoFancyName/producer-consumer/pkg/persistence"
	"github.com/NotSoFancyName/producer-consumer/pkg/proto"
	"github.com/NotSoFancyName/producer-consumer/pkg/utils"
	"github.com/NotSoFancyName/producer-consumer/system/producer/config"
	"github.com/NotSoFancyName/producer-consumer/system/producer/metrics"
	"github.com/NotSoFancyName/producer-consumer/system/producer/producer"
	"github.com/NotSoFancyName/producer-consumer/system/producer/server/v1"
)

var (
	version = "unknown"
)

var (
	pprofPortFlag   = flag.String("pprof-port", ":6061", "PPROF port for the service")
	metricsPortFlag = flag.String("metrics-port", ":8081", "Port for the metrics service")
	grpcPortFlag    = flag.String("grpc-port", ":50051", "Port for the gRPC server")
	configPathFlag  = flag.String("config", "", "Path to the configuration file")
	versionFlag     = flag.Bool("version", false, "Prints go versionFlag service was build with")
)

func main() {
	flag.Parse()

	if *versionFlag {
		log.Println(version)
	}

	cfg, err := config.Load(*configPathFlag)
	if err != nil {
		log.Fatal(err)
	}

	persistenceService, err := persistence.NewService(&cfg.PersistenceConfig)
	if err != nil {
		log.Fatal(err)
	}

	loggerConfig, err := utils.GetLoggerDefaultConfig(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	logger, err := loggerConfig.Build()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	logger.Info("Starting pprof server", zap.String("port", *pprofPortFlag))
	go func() {
		log.Fatal(http.ListenAndServe(*pprofPortFlag, nil))
	}()

	producerService, err := producer.NewService(persistenceService, &cfg.ProducerConfig, logger)
	if err != nil {
		log.Fatal(err)
	}
	producerService.Run()

	metricsService := metrics.NewService(producerService, &metrics.Config{Port: *metricsPortFlag}, logger)
	logger.Info("Starting metrics server", zap.String("port", *metricsPortFlag))
	metricsService.Run()

	lis, err := net.Listen("tcp", *grpcPortFlag)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcService, err := v1.NewGRPCTaskService(producerService, logger)
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterTaskProducerServer(grpcServer, grpcService)
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		logger.Info("Shutting down server gracefully...")

		grpcServer.GracefulStop()
		metricsService.Stop()
		logger.Info("Server stopped.")
	}()

	logger.Info("Starting gRPC server", zap.String("port", *grpcPortFlag))
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("Server returned an error", zap.Error(err))
	}
}
