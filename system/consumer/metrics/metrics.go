package metrics

import (
	"context"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/NotSoFancyName/producer-consumer/system/consumer/processor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Service struct {
	processorService processor.TaskProcessorService
	server           *http.Server
	config           *Config
	logger           *zap.Logger
}

type Config struct {
	Port string
}

func NewService(processorService processor.TaskProcessorService, cfg *Config, logger *zap.Logger) *Service {
	return &Service{
		processorService: processorService,
		config:           cfg,
		logger:           logger,
	}
}

func (s *Service) Run() {
	var customRegistry = prometheus.NewRegistry()
	totalTasksProcessingCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_tasks_processing_counter",
	})
	totalTasksDoneCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_tasks_done_counter",
	})

	totalTypeValueSumCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "total_type_value_sum_counter",
	}, []string{"type"})

	totalTypeProcessedCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "total_type_processed_counter",
	}, []string{"type"})

	customRegistry.MustRegister(totalTasksProcessingCounter)
	customRegistry.MustRegister(totalTasksDoneCounter)
	customRegistry.MustRegister(totalTypeValueSumCounter)
	customRegistry.MustRegister(totalTypeProcessedCounter)
	metricsHandler := func(w http.ResponseWriter, r *http.Request) {
		stats := s.processorService.GetStats()

		totalTasksProcessingCounter.Add(float64(stats.NewTasksProcessed))
		totalTasksDoneCounter.Add(float64(stats.NewTasksDone))

		for k, v := range stats.NewTasksTypeValueSum {
			totalTypeValueSumCounter.WithLabelValues(strconv.Itoa(k)).Add(float64(v))
		}

		for k, v := range stats.NewTasksPerType {
			totalTypeProcessedCounter.WithLabelValues(strconv.Itoa(k)).Add(float64(v))
		}

		promhttp.HandlerFor(customRegistry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", metricsHandler)
	s.server = &http.Server{
		Addr:    s.config.Port,
		Handler: mux,
	}

	go func() {
		err := s.server.ListenAndServe()
		if err != nil {
			s.logger.Error("Metrics server returned an error", zap.Error(err))
		}
	}()
}

func (s *Service) Stop() {
	if err := s.server.Shutdown(context.Background()); err != nil {
		s.logger.Error("Error shutting down metrics server", zap.Error(err))
	}
}
