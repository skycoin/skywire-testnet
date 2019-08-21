package visor

import (
	"bytes"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/logging"

	th "github.com/skycoin/skywire/internal/testhelpers"
)

// TaggedFormatter appends tag to log records and substitutes text
type TaggedFormatter struct {
	tag  []byte
	subs []bytesub
	*logging.TextFormatter
}
type strsub struct{ old, new string }
type bytesub struct{ old, new []byte }

// Format executes formatting of TaggedFormatter
func (tf *TaggedFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data, err := tf.TextFormatter.Format(entry)
	for _, sub := range tf.subs {
		data = bytes.ReplaceAll(data, sub.old, sub.new)
	}
	prepend := bytes.Repeat([]byte(" "), th.CallerDepth())
	return bytes.Join([][]byte{tf.tag, prepend, data}, []byte(" ")), err
}

// NewTaggedMasterLogger creates MasterLogger that prepends records with tag
func NewTaggedMasterLogger(tag string, subs []strsub) *logging.MasterLogger {

	bsubs := make([]bytesub, len(subs))
	for i := 0; i < len(subs); i++ {
		bsubs[i] = bytesub{[]byte(subs[i].old), []byte(subs[i].new)}
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
					TimestampFormat:    "05.000000",
				},
			},
			Hooks:        hooks,
			Level:        logrus.DebugLevel,
			ReportCaller: true,
		},
	}
}
