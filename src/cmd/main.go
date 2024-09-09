package main

import (
	"aggregator/src/bimport"
	"aggregator/src/config"
	"aggregator/src/external"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/tools/ossignal"
	"aggregator/src/tools/pgdb"
	"aggregator/src/uimport"

	"os"
)

var (
	version = os.Getenv("VERSION")
	module  = "aggregator"
)

func main() {
	log := logger.NewFileLogger(module)
	log.Debugln("version", version)

	conf, err := config.NewConfig(os.Getenv("CONF_PATH"))
	if err != nil {
		log.Fatalln(err)
	}

	pgDB := pgdb.SqlxDB(conf.PostgresURL())
	if err := pgDB.Ping(); err != nil {
		log.Fatalln(err)
	}

	pgSessionManager := transaction.NewSQLSessionManager(pgDB)

	ri := rimport.NewRepositoryImports(pgSessionManager)

	bi := bimport.NewEmptyBridge()

	ui := uimport.NewUsecaseImports(log, ri, bi)

	bi.InitBridge(
		ui.Usecase.Flow,
		ui.Usecase.Session,
		ui.Usecase.Channel,
		ui.Usecase.Traffic,
	)

	flagTerm := make(chan struct{})
	go ossignal.WaitForTerm(flagTerm)

	external.NewCron(log, ui).Run(flagTerm)
}
