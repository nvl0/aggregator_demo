package external_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"aggregator/src/external"
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/uimport"

	"go.uber.org/mock/gomock"
)

// waitTimeout предельное ожидание возврата Run/Start в тестах
const waitTimeout = 5 * time.Second

// dispatchGrace время, которое воркер держит единственный слот пула после
// отмены контекста: за него цикл обязан увидеть отмену и прекратить рассылку
const dispatchGrace = 200 * time.Millisecond

var testLogger = logger.NewDiscard()

func TestRunStopsOnContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ri := rimport.NewTestRepositoryImports(ctrl)
	ui := uimport.NewUsecaseImports(testLogger, ri.RepositoryImports(), nil)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	// тикер заведен на global.StartDur, за время теста он не срабатывает:
	// Run обязан вернуться по отмене контекста, а не по тику
	go func() {
		defer close(done)
		external.NewCron(testLogger, ui).Run(ctx)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(waitTimeout):
		t.Fatalf("Run не вернулся за %v после отмены контекста", waitTimeout)
	}
}

func TestCancelDuringCycleStopsDispatch(t *testing.T) {
	// один слот пула: второй nas_ip уходит в ожидание слота и упирается в отмену
	t.Setenv("WORKER_POOL_SIZE", "1")

	const (
		nasIP1 = "127.0.0.0"
		nasIP2 = "127.0.0.1"
	)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ri := rimport.NewTestRepositoryImports(ctrl)
	ts := transaction.NewMockSession(ctrl)

	channelList := []channel.Channel{
		{ID: channel.Internal, Enabled: true},
		{ID: channel.External, Enabled: false},
	}
	sessionList := []session.OnlineSession{
		{SessID: 1, NasIP: nasIP1, IP: "127.0.0.2"},
		{SessID: 2, NasIP: nasIP2, IP: "127.0.0.3"},
	}

	// три транзакции: мапка каналов, мапка сессий и чекпоинт единственного
	// nas_ip, дошедшего до воркера
	ri.SessionManager.EXPECT().CreateSession().Return(ts).Times(3)
	ts.EXPECT().Start(gomock.Any()).Return(nil).Times(3)
	ts.EXPECT().Rollback().Return(nil).Times(3)
	ri.MockRepository.Channel.EXPECT().LoadChannelList(gomock.Any(), ts).Return(channelList, nil)
	ri.MockRepository.Session.EXPECT().LoadOnlineSessionList(gomock.Any(), ts).Return(sessionList, nil)
	ri.MockRepository.Flow.EXPECT().ReadFlowDirNames().Return([]string{nasIP1, nasIP2}, nil)

	ctx, cancel := context.WithCancel(context.Background())

	canceled := make(chan struct{})
	release := make(chan struct{})

	// воркер первого nas_ip занимает единственный слот пула и отменяет контекст.
	// для nasIP2 загрузка чекпоинта не ожидается: рассылка обязана прекратиться
	ri.MockRepository.FlowBatch.EXPECT().
		LoadCommittedFileNames(gomock.Any(), ts, nasIP1).
		DoAndReturn(func(_ context.Context, _ transaction.Session, _ string) (map[string]bool, error) {
			cancel()
			close(canceled)
			<-release

			return nil, errors.New("обработка nas_ip остановлена тестом")
		})

	ui := uimport.NewUsecaseImports(testLogger, ri.RepositoryImports(), nil)
	c := external.NewCron(testLogger, ui)

	done := make(chan struct{})
	// тикер cron заведен на global.StartDur, дождаться тика в тесте нельзя,
	// поэтому цикл запускается напрямую тем же ctx, который Run отдает в Start
	go func() {
		defer close(done)
		c.Usecase.Aggregator.Start(ctx)
	}()

	<-canceled
	// слот пула все еще занят, поэтому рассылка nasIP2 упирается именно в ctx.Done
	time.Sleep(dispatchGrace)
	close(release)

	select {
	case <-done:
	case <-time.After(waitTimeout):
		t.Fatalf("Start не вернулся за %v после отмены контекста", waitTimeout)
	}
}
