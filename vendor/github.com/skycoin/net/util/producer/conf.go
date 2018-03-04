package producer

import (
	"os"
	"path/filepath"
	"encoding/json"
	"io/ioutil"
	"fmt"
)

type Config struct {
	AWSAccessKeyId string `json:"aws_access_key_id"`
	AWSSecretKey   string `json:"aws_secret_key"`
	QueueURL       string `json:"queue_url"`
	Region         string `json:"region"`
}

// LoadJsonConfig is used to load config files as json format to config.
// config should be a pointer to structure, if not, panic
func LoadConfig(conf interface{}, filename string) (err error) {
	var decoder *json.Decoder
	file := OpenFile(conf, filename)
	defer file.Close()
	decoder = json.NewDecoder(file)
	if err = decoder.Decode(conf); err != nil {
		return
	}
	json.Marshal(&conf)
	return
}

func OpenFile(conf interface{}, filename string) *os.File {
	var file *os.File
	var err error

	file, err = os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			err = WriteConfig(conf, filename)
			if err != nil {
				panic(err)
			}
			panic(fmt.Sprintf("You should configure parameters in %s", filename))
		}
		msg := fmt.Sprintf("Can not load config at %s. Error: %v", filename, err)
		panic(msg)
	}

	return file
}

func WriteConfig(conf interface{}, path string) (err error) {
	d, err := json.Marshal(conf)
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, d, 0600)
	return
}
