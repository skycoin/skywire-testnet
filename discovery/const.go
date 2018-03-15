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
                expander {

                }

                file {
                    filename = "discovery.log"
                    daily = true
                    rotate = true
                }

       
        }
}
`
	logFilePath = "./discovery.log.conf"
)
