package constants

const (
	AGENT_DIRECTORY = "agent/"
	BIN_PATH        = "bin/"
	CONF_PATH       = "conf/"
	INSTALL_PATH    = "/usr/local/logicmonitor/"

	DEFAULT_OS        = "Linux"
	TEMP_PATH         = "/tmp/"
	INSTALL_STAT_PATH = INSTALL_PATH + AGENT_DIRECTORY + "/tmp/install.tmp"

	LOCK_PATH               = INSTALL_PATH + AGENT_DIRECTORY + BIN_PATH
	LOG_FILE                = INSTALL_PATH + "/logs/wrapper.log"
	COLLECTOR_FOUND         = INSTALL_PATH + "collector.found"
	FIRST_RUN               = INSTALL_PATH + "first.run"
	MIN_NONROOT_INSTALL_VER = 28300
)
