package measure

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

// Measure  измерение затраченного времени
// необходимо использовать один экземпляр measure на одну управляющую функцию/метод
type Measure interface {
	Start(name string)
	Stop(name string) (elapsed time.Duration)
	Result() (total time.Duration)
}

type measure struct {
	writer   Writer
	data     map[string]measureData
	nameList []string
	s        sync.Mutex
}

type measureData struct {
	Name      string
	StartTime time.Time
	Elapsed   time.Duration
}

// NewMeasure конструктор
func NewMeasure(writer Writer) Measure {
	return &measure{
		writer:   writer,
		data:     make(map[string]measureData),
		nameList: make([]string, 0),
	}
}

func (m *measure) Start(name string) {
	m.s.Lock()
	m.data[name] = measureData{Name: name, StartTime: time.Now()}
	m.nameList = append(m.nameList, name)
	m.s.Unlock()
}

func (m *measure) Stop(name string) (elapsed time.Duration) {
	m.s.Lock()
	if mData, exists := m.data[name]; exists {
		elapsed = time.Since(mData.StartTime)
		mData.Elapsed = elapsed
		m.data[name] = mData
	} else {
		m.writer.Write(fmt.Sprintf("%s: %v", name, "некорректный start"))
	}
	m.s.Unlock()

	return elapsed
}

var measureEnable = os.Getenv("MEASURE") == "enable"

const maxResultCount = 5

func (m *measure) Result() (total time.Duration) {
	if !measureEnable {
		return
	}
	m.writer.Write("--------------------------------")
	m.writer.Write("Результаты замеров:")
	m.writer.Write("--------------------------------")

	// var heaviest measureData
	totalResults := make([]measureData, 0)

	for _, name := range m.nameList {
		if mData, exists := m.data[name]; exists && mData.Elapsed > 0 {
			total += mData.Elapsed

			m.writer.Write(fmt.Sprintf("%s....%v", name, mData.Elapsed))
			totalResults = append(totalResults, mData)

		} else {
			m.writer.Write(fmt.Sprintf("%s: %v", name, "таймер не завершен"))
		}
	}

	m.writer.Write("--------------------------------")
	m.writer.Write(fmt.Sprintf("Общее время....%v", total))
	m.writer.Write("--------------------------------")
	if len(totalResults) > 0 {
		sort.SliceStable(totalResults, func(i, j int) bool {
			return totalResults[i].Elapsed > totalResults[j].Elapsed
		})

		m.writer.Write("Топ долгих:")
		for index, row := range totalResults {
			m.writer.Write(fmt.Sprintf("%s....%v", row.Name, row.Elapsed))
			if index+1 >= maxResultCount {
				break
			}
		}

		m.writer.Write("--------------------------------")
	}

	m.reset()

	return
}

func (m *measure) reset() {
	m.data = make(map[string]measureData)
	m.nameList = make([]string, 0)
}
