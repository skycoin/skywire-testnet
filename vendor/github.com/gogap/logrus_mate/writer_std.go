package logrus_mate

import (
	"github.com/gogap/config"
	"io"
	"os"
)

func init() {
	RegisterWriter("stdout", NewStdoutWriter)
	RegisterWriter("stderr", NewStderrWriter)
}

func NewStdoutWriter(config.Configuration) (writer io.Writer, err error) {
	writer = os.Stdout
	return
}

func NewStderrWriter(config.Configuration) (writer io.Writer, err error) {
	writer = os.Stderr
	return
}
