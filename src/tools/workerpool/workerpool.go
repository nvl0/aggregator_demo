// Package workerpool ограниченный по размеру пул горутин.
package workerpool

import (
	"context"
	"sync"
)

// minSize минимально допустимый размер пула
const minSize = 1

// Pool пул горутин, количество одновременно выполняемых функций
// не превышает размер пула
type Pool struct {
	sem chan struct{}
	wg  sync.WaitGroup
}

// New создает пул на size одновременно выполняемых горутин.
// Если size меньше единицы, используется единица.
func New(size int) *Pool {
	if size < minSize {
		size = minSize
	}

	return &Pool{
		sem: make(chan struct{}, size),
	}
}

// Go захватывает слот пула и запускает fn в отдельной горутине.
// Блокируется, пока в пуле нет свободного слота.
// Возвращает false, если ctx отменился до захвата слота, при этом fn не запускается.
func (p *Pool) Go(ctx context.Context, fn func()) bool {
	// проверка до select, чтобы отмененный контекст
	// не проигрывал случайному выбору ветки в select
	if ctx.Err() != nil {
		return false
	}

	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return false
	}

	p.wg.Add(1)

	go func() {
		defer func() {
			<-p.sem
			p.wg.Done()
		}()

		fn()
	}()

	return true
}

// Wait ожидает завершения всех запущенных через Go горутин
func (p *Pool) Wait() {
	p.wg.Wait()
}
