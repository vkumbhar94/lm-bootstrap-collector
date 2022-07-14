package constants

const (
	// AgentDirectory Agent Directory
	AgentDirectory = "agent/"
	// BinPath bin directory
	BinPath     = "bin/"
	ConfPath    = "conf/"
	InstallPath = "/usr/local/logicmonitor/"

	DefaultOs       = "Linux"
	TempPath        = "/tmp/"
	InstallStatPath = InstallPath + AgentDirectory + "/tmp/install.tmp"

	LockPath       = InstallPath + AgentDirectory + BinPath
	LogFile        = InstallPath + "/logs/wrapper.log"
	CollectorFound = InstallPath + "collector.found"
	FirstRun       = InstallPath + "first.run"
)

const (
	// MinNonRootInstallVer
	// TODO: this var and the logic that depends on it can be removed after non-root
	// installer makes it to MGD
	MinNonRootInstallVer = 28300
)
