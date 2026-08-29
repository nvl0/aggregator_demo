// Команда loadgen генерирует синтетическую нагрузку для агрегатора
// и замеряет время цикла агрегации при разных размерах пула воркеров.
// Инструмент только для разработки: docker/Dockerfile собирает пакет src/cmd
// и подпакет loadgen в образ не попадает.
package main

import (
	"aggregator/src/bimport"
	"aggregator/src/config"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/flowgen"
	"aggregator/src/tools/logger"
	"aggregator/src/tools/pgdb"
	"aggregator/src/uimport"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	// syntheticSessIDBase база sess_id синтетических сессий,
	// отделяет их от сессий из миграции
	syntheticSessIDBase = 100000
	// syntheticNasPrefix префикс синтетических nas_ip директорий
	syntheticNasPrefix = "10."
	// clientIPPerOctet количество клиентских ip в одном октете
	clientIPPerOctet = 254
	// maxSynthetic предел синтетических сессий,
	// ограничен подсетью internal 127.0.0.1/20 из subnet-disabled
	maxSynthetic = 15 * clientIPPerOctet
	// nasOctetSize размер одного октета ip-адреса
	nasOctetSize = 256
	// dbConnReserve резерв соединений с бд под параллельные стартовые запросы
	dbConnReserve = 2
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(log); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	var (
		n         = flag.Int("n", 0, "количество синтетических nas_ip директорий и online сессий")
		cleanup   = flag.Bool("cleanup", false, "удалить синтетические директории и строки бд")
		runCycleF = flag.Bool("run", false, "выполнить один цикл агрегации и замерить время")
		poolSize  = flag.Int("pool-size", 0, "размер пула воркеров, 0 берет значение из conf.yaml")
	)
	flag.Parse()

	flowDir := os.Getenv("FLOW_DIR")
	if flowDir == "" {
		return errors.New("не задана переменная окружения FLOW_DIR")
	}

	// AggregatorUsecase читает размер пула из конфига при создании,
	// env переопределяет значение conf.yaml
	if *poolSize > 0 {
		if err := os.Setenv("WORKER_POOL_SIZE", strconv.Itoa(*poolSize)); err != nil {
			return fmt.Errorf("не удалось задать WORKER_POOL_SIZE: %w", err)
		}
	}

	var (
		conf config.Config
		err  error
	)
	if conf, err = config.NewConfig(os.Getenv("CONF_PATH")); err != nil {
		return fmt.Errorf("не удалось загрузить конфиг: %w", err)
	}

	db := pgdb.SqlxDB(conf.PostgresURL(), conf.WorkerPoolSize()+dbConnReserve)
	defer db.Close()

	if *cleanup {
		if err = cleanupLoad(db, flowDir); err != nil {
			return fmt.Errorf("не удалось очистить синтетическую нагрузку: %w", err)
		}
		log.Info("синтетическая нагрузка удалена")
	}

	if *n > 0 {
		if *n > maxSynthetic {
			return fmt.Errorf("максимум %d сессий, подсеть internal исчерпана", maxSynthetic)
		}

		if err = seedLoad(db, flowDir, *n); err != nil {
			return fmt.Errorf("не удалось создать синтетическую нагрузку: %w", err)
		}
		log.Info("синтетическая нагрузка создана", "n", *n)
	}

	if *runCycleF {
		var elapsed time.Duration
		if elapsed, err = runCycle(conf); err != nil {
			return fmt.Errorf("не удалось выполнить цикл агрегации: %w", err)
		}
		log.Info("цикл агрегации завершен", "n", *n, "pool", conf.WorkerPoolSize(), "elapsed", elapsed.String())
	}

	return nil
}

// nasIPByIndex имя синтетической nas_ip директории
func nasIPByIndex(i int) string {
	return fmt.Sprintf("%s%d.%d.0", syntheticNasPrefix, i/nasOctetSize, i%nasOctetSize)
}

// clientIPByIndex ip клиента внутри подсети internal 127.0.0.1/20.
// Нумерация начинается со второго октета, чтобы не пересечься
// с 127.0.0.1 из миграции.
func clientIPByIndex(i int) string {
	return fmt.Sprintf("127.0.%d.%d", i/clientIPPerOctet+1, i%clientIPPerOctet+1)
}

// seedLoad создает flow директории с файлами и синтетические online сессии
func seedLoad(db *sqlx.DB, flowDir string, n int) (err error) {
	for i := range n {
		if _, _, err = flowgen.Generate(flowgen.Params{
			NasIP:    nasIPByIndex(i),
			ClientIP: clientIPByIndex(i),
			FlowDir:  flowDir,
		}); err != nil {
			return err
		}
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Preparex(`
		insert into online_session (ip, sess_id, nas_ip)
		values ($1, $2, $3)
		on conflict (ip) do update
		set sess_id = excluded.sess_id, nas_ip = excluded.nas_ip`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for i := range n {
		if _, err = stmt.Exec(clientIPByIndex(i), syntheticSessIDBase+i, nasIPByIndex(i)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// cleanupLoad удаляет синтетические flow директории, чанки и online сессии
func cleanupLoad(db *sqlx.DB, flowDir string) (err error) {
	dirList, err := os.ReadDir(flowDir)
	if err != nil {
		return err
	}

	for _, dir := range dirList {
		if dir.IsDir() && strings.HasPrefix(dir.Name(), syntheticNasPrefix) {
			if err = os.RemoveAll(fmt.Sprintf("%s/%s", flowDir, dir.Name())); err != nil {
				return err
			}
		}
	}

	if _, err = db.Exec(`delete from chunk where sess_id >= $1`, syntheticSessIDBase); err != nil {
		return err
	}

	_, err = db.Exec(`delete from online_session where sess_id >= $1`, syntheticSessIDBase)

	return err
}

// runCycle собирает зависимости как в src/cmd/main.go
// и выполняет один цикл агрегации с замером времени
func runCycle(conf config.Config) (elapsed time.Duration, err error) {
	log := logger.NewNoFileLogger("loadgen")

	pgDB := pgdb.SqlxDB(conf.PostgresURL(), conf.WorkerPoolSize()+dbConnReserve)
	defer pgDB.Close()

	if err = pgDB.Ping(); err != nil {
		return elapsed, err
	}

	ri := rimport.NewRepositoryImports(conf, transaction.NewSQLSessionManager(pgDB))
	bi := bimport.NewEmptyBridge()
	ui := uimport.NewUsecaseImports(log, ri, bi)

	bi.InitBridge(
		ui.Usecase.Flow,
		ui.Usecase.Session,
		ui.Usecase.Channel,
		ui.Usecase.Traffic,
		ui.Usecase.Aggregator,
	)

	start := time.Now()
	ui.Usecase.Aggregator.Start(context.Background())
	elapsed = time.Since(start)

	return elapsed, err
}
