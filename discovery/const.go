package discovery

// constants of version
const (
	Version = "0.1.0"
	Tag     = "dev"
)

// constants of log
const (
	logConf = `
discovery {

        level = "debug"

        formatter.name = "text"
        formatter.options  {
                            force-colors      = false
                            disable-colors    = false
                            disable-timestamp = false
                            full-timestamp    = false
                            timestamp-format  = "2006-01-02 15:04:05"
                            disable-sorting   = false
        }

        hooks {
                file {
                    filename = "discovery.log"
                    daily = true
                    rotate = true
                    level = 3
                    max-days = 7
                    max-size = 100000000
                }
        }
}
`
	logFilePath = "./discovery.log.conf"
)
