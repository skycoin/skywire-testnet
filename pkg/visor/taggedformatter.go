package visor

import (
	"bytes"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/skycoin/skywire/internal/testhelpers"
)

// TaggedFormatter appends tag to log records and substitutes text
type TaggedFormatter struct {
	tag  []byte
	subs []struct{ old, new []byte }
	*logging.TextFormatter
}

// Format executes formatting of TaggedFormatter
func (tf *TaggedFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data, err := tf.TextFormatter.Format(entry)
	for _, sub := range tf.subs {
		data = bytes.ReplaceAll(data, sub.old, sub.new)
	}
	prepend := bytes.Repeat([]byte(" "), testhelpers.CallerDepth())
	return bytes.Join([][]byte{tf.tag, prepend, data}, []byte(" ")), err
}

// NewTaggedMasterLogger creates MasterLogger that prepends records with tag
func NewTaggedMasterLogger(tag string, ssubs []struct{ old, new string }) *logging.MasterLogger {
	s2bsub := func(s struct{ old, new string }) struct{ old, new []byte } {
		return struct{ old, new []byte }{[]byte(s.old), []byte(s.new)}
	}
	bsubs := make([]struct{ old, new []byte }, len(ssubs))
	for i := 0; i < len(ssubs); i++ {
		bsubs[i] = s2bsub(ssubs[i])
	}

	hooks := make(logrus.LevelHooks)
	return &logging.MasterLogger{
		Logger: &logrus.Logger{
			Out: os.Stdout,
			Formatter: &TaggedFormatter{
				[]byte(tag),
				bsubs,
				&logging.TextFormatter{
					AlwaysQuoteStrings: true,
					QuoteEmptyFields:   true,
					FullTimestamp:      true,
					ForceFormatting:    true,
					DisableColors:      false,
					ForceColors:        false,
					TimestampFormat:    time.StampMicro,
				},
			},
			Hooks:        hooks,
			Level:        logrus.DebugLevel,
			ReportCaller: true,
		},
	}
}
