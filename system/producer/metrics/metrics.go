package metrics

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	"github.com/NotSoFancyName/producer-consumer/system/producer/producer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Service struct {
	producerService producer.Service
	server          *http.Server
	config          *Config
	logger          *zap.Logger
}

type Config struct {
	Port string
}

func NewService(producerService producer.Service, cfg *Config, logger *zap.Logger) *Service {
	return &Service{
		producerService: producerService,
		config:          cfg,
		logger:          logger,
	}
}

func (s *Service) Run() {
	var customRegistry = prometheus.NewRegistry()
	totalTasksCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_tasks_received_counter",
	})
	customRegistry.MustRegister(totalTasksCounter)
	metricsHandler := func(w http.ResponseWriter, r *http.Request) {
		stats, err := s.producerService.GetStats()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		totalTasksCounter.Add(float64(stats.NewlyProducedTasks))

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
