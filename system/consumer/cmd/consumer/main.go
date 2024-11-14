package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/NotSoFancyName/producer-consumer/pkg/persistence"
	"github.com/NotSoFancyName/producer-consumer/pkg/utils"
	"github.com/NotSoFancyName/producer-consumer/system/consumer/client"
	"github.com/NotSoFancyName/producer-consumer/system/consumer/config"
	"github.com/NotSoFancyName/producer-consumer/system/consumer/metrics"
	"github.com/NotSoFancyName/producer-consumer/system/consumer/processor"
)

var (
	version = "unknown"
)

var (
	pprofPortFlag   = flag.String("pprof-port", ":6062", "PPROF port for the service")
	metricsPortFlag = flag.String("metrics-port", ":8082", "Port for the metrics service")
	grpcPortFlag    = flag.String("grpc-port", ":50051", "Port for the gRPC server to connect to")
	configPathFlag  = flag.String("config", "./configs/consumer.yml", "Path to the configuration file")
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

	loggerConfig, err := utils.GetLoggerDefaultConfig(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	logger, err := loggerConfig.Build()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	logger.Info("Starting pprof service", zap.String("port", *pprofPortFlag))
	go func() {
		log.Fatal(http.ListenAndServe(*pprofPortFlag, nil))
	}()

	logger.Info("Starting persistence service")
	persistenceService, err := persistence.NewService(&cfg.PersistenceConfig)
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("Starting consumer service")
	processorService, err := processor.NewTaskProcessorService(persistenceService, &cfg.ProcessorConfig, logger)
	if err != nil {
		log.Fatal(err)
	}
	processorService.Run()

	logger.Info("Starting metrics service", zap.String("port", *metricsPortFlag))
	metricsService := metrics.NewService(processorService, &metrics.Config{Port: *metricsPortFlag}, logger)
	metricsService.Run()

	logger.Info("Starting client service", zap.String("port", *grpcPortFlag))
	clientService, err := client.NewService(processorService, &client.Config{ProducerAddress: *grpcPortFlag,
		RateLimiterConfig: cfg.RateLimiterConfig}, logger)
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("Starting client service")
	clientService.Run()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	logger.Info("Shutting down server gracefully...")
	metricsService.Stop()
	clientService.Stop()
	logger.Info("Server stopped.")
}
