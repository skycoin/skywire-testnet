package producer

import (
	"testing"
	"github.com/skycoin/skycoin/src/util/file"
	"path/filepath"
)

func init() {
	err := Init(filepath.Join(file.UserHome(), ".skywire", "discovery", "conf.json"))
	if err != nil {
		panic(err)
	}
}
func TestSend(t *testing.T) {
	err := Send(&MqBody{
		Key:          "TestKey",
		Seq:          2,
		FromApp:      "03c773268be29f8fe48144cc123dfd45b487dceadbb8e1d7817f84d612521bc68c",
		FromNode:     "03c773268be29f8fe48144ccb2c19045b487dceadbb8e1d7817f84d612521bc68c",
		ToNode:       "038b558e91a343c0f449b536ceab640d4055dbced85f6f5969a58ee56f2280588c",
		ToApp:        "03c773268be29f8zzvvcvdfcb2c19045b487dceadbb8e1d7817f84d612521bc68c",
		Uid:          1,
		FromHostPort: "03c773268be29fsdfdsfsdfds2c19045b487dceadbb8e1d7817f84d612521bc68c",
		ToHostPort:   "03c773268be29f8fe48144ccb2c19045b487dceadbb8e1d7817f84d6dzzxcxds8c",
		FromIp:       "03c773268be29f8fe48144ccb2c19045b487dceadbb8e1d7817f84dmmmdffasdas",
		ToIp:         "03c773268be29f8fe48144ccb2c19045b487dceadbb8e1d7817f84d61xzxczxsdf",
		Count:        100,
		IsEnd:        true,
	})
	//err := Send("03c773268be29f8fe48144ccb2c19045b487dceadbb8e1d7817f84d612521bc68c", "038b558e91a343c0f449b536ceab640d4055dbced85f6f5969a58ee56f2280588c", 100)
	if err != nil {
		panic(err)
		return
	}
}
