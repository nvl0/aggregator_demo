package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var debugMode = os.Getenv("DEBUG") == "true"

type fileLogHook struct {
	file   *os.File
	level  string
	module string
}

func (f *fileLogHook) Levels() []logrus.Level {
	if f.level == "error" {
		return []logrus.Level{logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel}
	} else if f.level == "info" {
		if debugMode {
			return []logrus.Level{logrus.InfoLevel, logrus.WarnLevel, logrus.DebugLevel}
		}

		return []logrus.Level{logrus.InfoLevel, logrus.WarnLevel}
	} else {
		return logrus.AllLevels
	}
}

func (f *fileLogHook) SetInfoLevel() {
	f.level = "info"
}

func (f *fileLogHook) SetErrorLevel() {
	f.level = "error"
}

func (f *fileLogHook) Fire(e *logrus.Entry) error {
	pc := make([]uintptr, 3)
	cnt := runtime.Callers(8, pc)
	for i := 0; i < cnt; i++ {
		fu := runtime.FuncForPC(pc[i] - 1)
		name := fu.Name()
		if !strings.Contains(name, "github.com/sirupsen/logrus") {
			file, line := fu.FileLine(pc[i] - 1)
			fileDir := path.Base(path.Dir(file))
			if !strings.Contains(fileDir, "gin@") {
				e.Data["file"] = fmt.Sprintf("%s/%s", fileDir, path.Base(file))
				e.Data["line"] = line
			}

			break
		}
	}

	str, err := e.String()
	if err != nil {
		return err
	}

	str = fmt.Sprintf("[%s] %s", f.module, str)

	_, err = io.Writer(f.file).Write([]byte(str))

	return err
}

func textFormatter() *logrus.TextFormatter {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "02.01.2006 15:04:05"
	customFormatter.FullTimestamp = true
	customFormatter.ForceColors = true

	return customFormatter
}

func openFile(fileName string) *fileLogHook {
	hook := new(fileLogHook)
	logDir := os.Getenv("LOG_DIR")
	file, err := os.OpenFile(fmt.Sprintf("%s/%s", logDir, fileName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err == nil {
		hook.file = file
	} else {
		log.Fatalln("Нет возможности писать в файл-логи. Убедитесь, что существует папка c правом доступа", logDir)
	}

	return hook
}

// NewFileLogger файловый логгер
func NewFileLogger(module string) *logrus.Logger {
	newLogger := logrus.New()
	newLogger.Formatter = textFormatter()

	fileHook := openFile(fmt.Sprintf("%s.log", module))
	fileHook.SetInfoLevel()
	fileHook.module = module
	newLogger.Hooks.Add(fileHook)

	errorFileHook := openFile(fmt.Sprintf("%s_error.log", module))
	errorFileHook.SetErrorLevel()
	errorFileHook.module = module

	newLogger.Hooks.Add(errorFileHook)

	if debugMode {
		newLogger.Level = logrus.DebugLevel
	}

	return newLogger
}

// NewNoFileLogger файловый логгер
func NewNoFileLogger(module string) *logrus.Logger {
	newLogger := logrus.New()
	newLogger.Formatter = textFormatter()

	newLogger.Level = logrus.DebugLevel

	return newLogger
}

// NewSingleLogger файловый логгер без раздления лога ошибок
func NewSingleLogger(module string) *logrus.Logger {
	newLogger := logrus.New()
	newLogger.Formatter = textFormatter()
	fileName := fmt.Sprintf("%s.log", module)
	fileHook := openFile(fileName)
	fileHook.SetInfoLevel()
	fileHook.module = module
	newLogger.Hooks.Add(fileHook)

	errorFileHook := openFile(fileName)
	errorFileHook.SetErrorLevel()
	errorFileHook.module = module

	newLogger.Hooks.Add(errorFileHook)

	if debugMode {
		newLogger.Level = logrus.DebugLevel
	}

	return newLogger
}
