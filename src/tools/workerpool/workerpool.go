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
	sem     chan struct{}
	wg      sync.WaitGroup
	onPanic func(recovered any)
}

// Option опция пула
type Option func(p *Pool)

// WithOnPanic задает колбэк, вызываемый при панике внутри задачи.
// Пакет намеренно ничего не знает о логгере приложения, поэтому логированием
// восстановленного значения занимается вызывающая сторона.
// Если колбэк не задан, паника перехватывается и игнорируется
func WithOnPanic(onPanic func(recovered any)) Option {
	return func(p *Pool) {
		p.onPanic = onPanic
	}
}

// New создает пул на size одновременно выполняемых горутин.
// Если size меньше единицы, используется единица.
func New(size int, optList ...Option) *Pool {
	if size < minSize {
		size = minSize
	}

	p := &Pool{
		sem: make(chan struct{}, size),
	}

	for _, opt := range optList {
		opt(p)
	}

	return p
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

		// паника одной задачи не должна ронять процесс целиком и остальные задачи пула.
		// defer объявлен вторым, а выполняется первым, поэтому слот семафора
		// освобождается уже после перехвата
		defer func() {
			if recovered := recover(); recovered != nil && p.onPanic != nil {
				p.onPanic(recovered)
			}
		}()

		fn()
	}()

	return true
}

// Wait ожидает завершения всех запущенных через Go горутин
func (p *Pool) Wait() {
	p.wg.Wait()
}
