package config

type App struct {
	Daemon bool     `yaml:"daemon"`
	Path   string   `yaml:"path"`
	Args   []string `yaml:"default_args"`
	Desc   string   `yaml:"desc"`
	Help   string   `yaml:"help"`
}

type Settings struct {
	VerboseInput  bool `yaml:"verboseInput"`
	VerifyParsing bool `yaml:"verifyParsingByPrinting"`
	RunHeadless   bool `yaml:"runHeadless"`
}

type Config struct {
	Apps     map[string]App `yaml:"apps"`
	Settings Settings       `yaml:"settings"`
}
