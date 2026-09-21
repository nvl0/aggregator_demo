package main

import (
	"aggregator/src/config"
	"aggregator/src/external"
	"aggregator/src/external/health"
	"aggregator/src/external/httpsrv"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/tools/metrics"
	"aggregator/src/tools/pgdb"
	"aggregator/src/tools/tracing"
	"aggregator/src/uimport"

	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var version = os.Getenv("VERSION")

// dbConnReserve резерв соединений с бд под параллельные стартовые запросы
// и пробу готовности /readyz, которая ходит в бд во время цикла агрегации
const dbConnReserve = 3

// tracerShutdownTimeout предельное время на выгрузку накопленных спанов при остановке
const tracerShutdownTimeout = 5 * time.Second

func main() {
	log := logger.New()
	log.Debug("version", "version", version)

	conf, err := config.NewConfig(os.Getenv("CONF_PATH"))
	if err != nil {
		log.Error("не удалось загрузить конфиг", "error", err)
		os.Exit(1)
	}

	pgDB := pgdb.SqlxDB(conf.PostgresURL(), conf.WorkerPoolSize()+dbConnReserve)
	if err = pgDB.Ping(); err != nil {
		log.Error("не удалось установить соединение с бд", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	m := metrics.New(log, version, pgDB.DB)

	tr, tracerShutdown := tracing.New(ctx, log, conf.TracingEndpoint())
	defer func() {
		// собственный контекст: ctx уже отменен на этом этапе graceful shutdown
		sc, cancel := context.WithTimeout(context.Background(), tracerShutdownTimeout)
		defer cancel()

		if shutdownErr := tracerShutdown(sc); shutdownErr != nil {
			log.Error("не удалось корректно остановить трейсер, ошибка", "error", shutdownErr)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", m.Handler())
	mux.HandleFunc("/healthz", health.Live)
	mux.HandleFunc("/readyz", health.Ready(pgDB))

	wg.Add(1)

	go func() {
		defer wg.Done()

		if srvErr := httpsrv.Run(ctx, log, conf.MetricsAddr(), mux); srvErr != nil {
			log.Error("служебный http сервер остановлен с ошибкой", "error", srvErr)
		}
	}()

	pgSessionManager := transaction.NewSQLSessionManager(pgDB)

	ri := rimport.NewRepositoryImports(conf, pgSessionManager)

	ui := uimport.NewUsecaseImports(log, ri, m, tr)

	external.NewCron(log, ui).Run(ctx)

	// Run мог выйти не по сигналу, поэтому гасим сервер явно
	stop()
	// даем httpsrv.Shutdown доработать
	wg.Wait()
}
