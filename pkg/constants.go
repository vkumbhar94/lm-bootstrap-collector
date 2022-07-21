package pkg

import "os"

const (
	InstallDir = "/usr/local/logicmonitor"
	AgentDir   = InstallDir + string(os.PathSeparator) + "agent"
	AgentConf  = AgentDir + string(os.PathSeparator) + "conf" + string(os.PathSeparator) + "agent.conf"
)

type ConfigFormat uint

const (
	Unknown = iota
	Properties
	Json
	Yaml
	Csv
)
