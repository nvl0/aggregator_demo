package logger

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type usecaseHook struct {
	parentHook logrus.Hook
	prefix     string
}

func NewUsecaseLogger(parentLogger *logrus.Logger, prefix string) *logrus.Logger {
	newLogger := logrus.New()
	newLogger.Formatter = parentLogger.Formatter
	newLogger.Level = parentLogger.Level
	newLogger.Hooks = make(logrus.LevelHooks)

	for level, hookList := range parentLogger.Hooks {
		newHookList := make([]logrus.Hook, len(hookList))
		copy(newHookList, hookList)
		newLogger.Hooks[level] = newHookList

		for index, hook := range newHookList {
			modifHook := newUsecaseHook(hook, prefix)
			newHookList[index] = modifHook
		}
	}

	return newLogger
}

func newUsecaseHook(h logrus.Hook, prefix string) logrus.Hook {
	return &usecaseHook{
		parentHook: h,
		prefix:     prefix,
	}
}

func (u *usecaseHook) Fire(e *logrus.Entry) error {
	prefix := fmt.Sprintf("[%s]", u.prefix)
	if !strings.HasPrefix(e.Message, prefix) {
		e.Message = fmt.Sprintf("%s %s", prefix, e.Message)
	}
	return u.parentHook.Fire(e)
}

func (u *usecaseHook) Levels() []logrus.Level {
	return u.parentHook.Levels()
}
