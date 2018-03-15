package logrus_mate

import (
	"errors"
	"sort"
	"sync"

	"github.com/gogap/config"
	"github.com/sirupsen/logrus"
)

var (
	hooksLocker  = sync.Mutex{}
	newHookFuncs = make(map[string]NewHookFunc)
)

type NewHookFunc func(config.Configuration) (hook logrus.Hook, err error)

func RegisterHook(name string, newHookFunc NewHookFunc) {
	hooksLocker.Lock()
	hooksLocker.Unlock()

	if name == "" {
		panic("logurs mate: Register hook name is empty")
	}

	if newHookFunc == nil {
		panic("logurs mate: Register hook is nil")
	}

	if _, exist := newHookFuncs[name]; exist {
		panic("logurs mate: Register called twice for hook " + name)
	}

	newHookFuncs[name] = newHookFunc
}

func Hooks() []string {
	hooksLocker.Lock()
	defer hooksLocker.Unlock()
	var list []string
	for name := range newHookFuncs {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func NewHook(name string, config config.Configuration) (hook logrus.Hook, err error) {
	hooksLocker.Lock()
	defer hooksLocker.Unlock()

	if newHookFunc, exist := newHookFuncs[name]; !exist {
		err = errors.New("logurs mate: hook not registerd: " + name)
		return
	} else {
		hook, err = newHookFunc(config)
	}

	return
}
