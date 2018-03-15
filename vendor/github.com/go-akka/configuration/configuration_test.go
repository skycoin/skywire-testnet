package configuration

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

func TestParseKeyOrder(t *testing.T) {

	wg := &sync.WaitGroup{}

	fn := func() {

		defer func() {
			wg.Done()
		}()

		for i := 0; i < 100000; i++ {
			conf := LoadConfig("tests/configs.conf")
			for g := 1; g < 3; g++ {
				for i := 1; i < 4; i++ {
					key := fmt.Sprintf("test.out.a.b.c.d.groups.g%d.o%d.order", g, i)
					order := conf.GetInt32(key, -1)

					if order != int32(i) {
						fmt.Println(conf)
						t.Fatalf("order not match,group %d, except: %d, real order: %d", g, i, order)
						return
					}
				}
			}
			conf = nil
			runtime.Gosched()
		}
	}

	wg.Add(2)

	go fn()
	go fn()

	wg.Wait()
}
