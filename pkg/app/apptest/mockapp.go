package apptest

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/skycoin/skywire/pkg/app"
)

// GenerateMockApp generates a mock app binary within the specified path.
func GenerateMockApp(path string, m app.Meta) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.FileMode(0777))
	if err != nil {
		return err
	}
	defer func() { err = f.Close() }()

	jm, err := json.Marshal(m)
	if err != nil {
		return err
	}

	template := `#!/bin/bash
PK="${%s}"
if [[ $# -eq 1 && $1 = '%s' ]]; then echo '%s'; exit 0
elif [[ -n "${PK}" ]]; then echo "host: ${PK}"; while [ 1 ]; do test $? -gt 128 && exit 0; done
else exit 1
fi`
	_, err = fmt.Fprintf(f, template, app.EnvHostPK, app.SetupCmdName, string(jm))
	return err
}
