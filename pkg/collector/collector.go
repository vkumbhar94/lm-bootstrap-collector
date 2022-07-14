package collector

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/logicmonitor/lm-sdk-go/client"
	"github.com/logicmonitor/lm-sdk-go/client/lm"
	"github.com/logicmonitor/lm-sdk-go/models"
	"github.com/sirupsen/logrus"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/cerrors"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/constants"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/util"
)

func Start(logger logrus.FieldLogger, creds *config.Creds, conf *config.Config, client *client.LMSdkGo) error {
	collector, err := FindCollector(conf, client)
	if err != nil {
		logger.Warn("collector not found")
		if conf.Kubernetes {
			return fmt.Errorf("running in kubernetes but collector not found: %w", err)
		}
		// TODO: create collector from config
		collector, err = NewCollector(logger, conf, client)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Collector Found")
		// we want to make a note on the fs that we found an existing collector
		// so that we don't remove it during a future cleanup, but we should
		// only make this note if this is the first time the container is run
		// (otherwise, every subsequent should detect the existing collector
		// that we're going to create below. Not the behavior we want)
		if _, err := os.Stat(constants.FirstRun); errors.Is(err, os.ErrNotExist) {
			_ = util.Touch(constants.CollectorFound)
		}
	}

	// let subsequent runs know that this isn't the first container run
	_ = util.Touch(constants.FirstRun)
	if _, err := os.Stat(constants.InstallPath + constants.AgentDirectory); !errors.Is(err, os.ErrNotExist) {
		logger.Info(`Collector already installed.`)
		_ = util.Cleanup(logger)
		return nil
	}
	return Install(logger, creds, conf, client, collector)
}

func Install(logger logrus.FieldLogger, creds *config.Creds, conf *config.Config, sdkGo *client.LMSdkGo, collector *models.Collector) error {
	currentVersion := collector.Build
	filename, err := DownloadInstaller(logger, conf, sdkGo, collector)
	if filename == "" && errors.Is(err, cerrors.VersionError) {
		collector.Build, conf.Version = "0", 0
		logger.Warn("retry to get latest available collector version")
		filename, err = DownloadInstaller(nil, conf, sdkGo, collector)
		if err != nil {
			return err
		}
	}
	logger.Infof("Installing collector: %s", filename)
	installArgs := []string{"-y"}
	//  force update the collector object to ensure all details are up-to-date
	//  e.g. build version
	if collector.Build != "0" {
		params := lm.NewGetCollectorByIDParams()
		params.ID = collector.ID
		c, err := sdkGo.LM.GetCollectorByID(params)
		if err == nil {
			logger.Infof("Collector version: %s", c.Payload.Build)
			currentVersion = c.Payload.Build
		}
	}
	if conf.Version >= constants.MinNonRootInstallVer || conf.UseEa || conf.Version == 0 {
		installArgs = append(installArgs, "-u", "root")
	}
	if creds.ProxyUrl != "" {
		installArgs = append(installArgs, "-p", creds.Proxy.Host)
		if creds.ProxyUser != "" {
			installArgs = append(installArgs, "-U", creds.ProxyUser)
			if creds.ProxyPass != "" {
				installArgs = append(installArgs, "-P", creds.ProxyPass)
			}
		}
	}
	err, stdout, stderr := util.Shellout(filename, installArgs...)
	if err != nil || stderr != "" {
		msg := stderr
		if msg == "" {
			msg = stdout
			if strings.Contains(msg, "LogicMonitor Collector has been installed successfully") &&
				strings.Contains(msg, "unknown option: u") {
			} else {
				return err
			}
		}
	}
	if creds.IgnoreSSL {
		_, _, _ = util.Shellout("sed", "-i",
			"s/EnforceLogicMonitorSSL=true/EnforceLogicMonitorSSL=false/g",
			"/usr/local/logicmonitor/agent/conf/agent.conf")
	}
	logger.Info("Cleaning up downloaded installer")
	// log message if version is outdated
	if f, err := os.Open(constants.InstallStatPath); err == nil {
		defer func(f *os.File) {
			_ = f.Close()
		}(f)
		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "complexInfo=") {
				val := strings.Split(line, "=")[1]
				complexInfo := map[string]any{}
				_ = json.Unmarshal([]byte(val), &complexInfo)
				if installedCollector, ok := complexInfo["installedCollector"]; ok {
					if cm, ok := installedCollector.(map[string]any); ok {
						if version, ok := cm["version"]; ok {
							installedVersion := version.(string)
							upgradeMsg := fmt.Sprintf("Requested installedCollector version %s is outdated so upgraded to %s  version ",
								currentVersion, installedVersion)
							logger.Info(upgradeMsg)
						}
					}
				}

			}
		}
	}
	return nil
}

func DownloadInstaller(logger logrus.FieldLogger, conf *config.Config, sdkGo *client.LMSdkGo, collector *models.Collector) (string, error) {
	logger.Infof("Downloading collector %d", collector.ID)

	params := lm.NewGetCollectorInstallerParamsWithTimeout(10 * time.Minute)

	csize := conf.Size.String()
	logger.Infof("size: %s", csize)
	params.SetCollectorSize(&csize)

	params.SetCollectorID(collector.ID)
	params.SetUseEA(&conf.UseEa)

	osAndArch := constants.DefaultOs + "64"
	if strconv.IntSize == 32 {
		osAndArch = constants.DefaultOs + "32"
	}
	params.SetOsAndArch(osAndArch)

	if conf.Version != 0 {
		params.SetCollectorVersion(&conf.Version)
	} else if collector.Build != "0" && !conf.UseEa {
		v, _ := strconv.ParseInt(collector.Build, 10, 32)
		v2 := int32(v)
		params.SetCollectorVersion(&v2)
	}
	filename := fmt.Sprintf("%slogicmonitorsetupx64_%d.bin", constants.TempPath, conf.ID)

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0o755)
	if err != nil {
		return "", err
	}
	_, err = sdkGo.LM.GetCollectorInstaller(params, f)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "only those versions") {
			logger.Errorf("%s. Most likely the collector_version %d is invalid/out-dated. See https://www.logicmonitor.com/support/settings/collectors/collector-versions/ for more information on collector versioning\n", msg, params.CollectorVersion)
			return "", cerrors.VersionError
		}
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()
	logger.Infof("Downloaded installer at %s", filename)
	// detect cases where we download an invalid installer
	stat, err := os.Stat(filename)
	if err == nil {
		logger.Infof("Installer size: %s", ToSI(stat.Size()))
	}

	return filename, nil
}

func ToSI(size int64) string {
	suffixes := []string{"B", "KB", "MB", "GB", "TB"}
	base := math.Log(float64(size)) / math.Log(1024)
	getSize := Round(math.Pow(1024, base-math.Floor(base)), .5, 2)
	getSuffix := suffixes[int(math.Floor(base))]
	return strconv.FormatFloat(getSize, 'f', -1, 64) + " " + getSuffix
}

func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func NewCollector(logger logrus.FieldLogger, conf *config.Config, sdkGo *client.LMSdkGo) (*models.Collector, error) {
	collectorGroupID, err := FindCollectorGroupID(logger, conf.Group, sdkGo)
	if err != nil {
		return nil, fmt.Errorf("collector group [%s] doesn't exist: %w", conf.Group, err)
	}
	collector := &models.Collector{
		CollectorGroupID:              collectorGroupID,
		BackupAgentID:                 conf.BackupCollectorID,
		EnableFailBack:                conf.EnableFailBack,
		EscalatingChainID:             conf.EscalatingChainID,
		ID:                            conf.ID,
		ResendIval:                    conf.ResendInterval,
		SuppressAlertClear:            conf.SuppressAlertClear,
		NeedAutoCreateCollectorDevice: false,
	}

	if conf.Description != "" {
		collector.Description = conf.Description
	} else if hostname, err := os.Hostname(); err == nil {
		collector.Description = hostname
	}
	params := lm.NewAddCollectorParams()
	params.SetBody(collector)
	resp, err := sdkGo.LM.AddCollector(params)
	if err != nil {
		if GetHTTPStatusCodeFromLMSDKError(err) == 600 {
			// Status 600: The record already exists
			return collector, nil
		}
		if es, ok := err.(*lm.AddCollectorDefault); ok && es.Payload.ErrorCode == 600 {
			// Status 600: The record already exists
			return collector, nil
		}
		return nil, err
	}
	collector = resp.Payload
	return collector, nil
}

func FindCollectorGroupID(logger logrus.FieldLogger, collectorGroup string, sdkGo *client.LMSdkGo) (int32, error) {
	logger.Infof("Finding collector group " + collectorGroup)
	// if the root group is set, no need to search
	if collectorGroup == "/" {
		return 1, nil
	}
	// trim leading / if it exists
	collectorGroup = strings.TrimSuffix(collectorGroup, "/")
	params := lm.NewGetCollectorGroupListParams()
	size := int32(-1)
	params.SetSize(&size)
	resp, err := sdkGo.LM.GetCollectorGroupList(params)
	if err != nil {
		return -1, err
	}
	for _, c := range resp.Payload.Items {
		if *c.Name == collectorGroup {
			return c.ID, nil
		}
	}
	return -1, cerrors.CollectorGroupNotFoundError
}

func FindCollector(conf *config.Config, sdkGo *client.LMSdkGo) (*models.Collector, error) {
	if conf.ID != 0 {
		params := lm.NewGetCollectorByIDParams()
		params.ID = conf.ID
		resp, err := sdkGo.LM.GetCollectorByID(params)
		if err != nil {
			return nil, err
		}
		return resp.Payload, nil
	} else {
		list, err := sdkGo.LM.GetCollectorList(nil)
		if err != nil {
			return nil, err
		}
		for _, c := range list.Payload.Items {
			if c.Description == conf.Description {
				params := lm.NewGetCollectorByIDParams()
				params.SetID(c.ID)
				resp, err := sdkGo.LM.GetCollectorByID(params)
				if err != nil {
					return nil, err
				}
				return resp.Payload, nil
				// return c, nil

			}
		}
	}
	return nil, errors.New("collector not found")
}
