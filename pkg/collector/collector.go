package collector

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/logicmonitor/lm-sdk-go/client"
	"github.com/logicmonitor/lm-sdk-go/client/lm"
	"github.com/logicmonitor/lm-sdk-go/models"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/cerrors"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/constants"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/util"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

func Start(conf *config.Config, client *client.LMSdkGo) error {
	collector, err := FindCollector(conf, client)
	if err != nil {
		fmt.Println("collector not found")
		if conf.Kubernetes {
			return fmt.Errorf("running in kubernetes but collector not found: %w", err)
		}
		// TODO: create collector from config
		collector, err = NewCollector(conf, client)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("Collector Found")
		if _, err := os.Stat(constants.FIRST_RUN); errors.Is(err, os.ErrNotExist) {
			_ = util.Touch(constants.COLLECTOR_FOUND)
		}
	}
	_ = util.Touch(constants.FIRST_RUN)
	if _, err := os.Stat(constants.INSTALL_PATH + constants.AGENT_DIRECTORY); !errors.Is(err, os.ErrNotExist) {
		fmt.Println(`Collector already installed.`)
		Cleanup(conf, client)
		return nil
	}
	return Install(conf, client, collector)
}

func Install(conf *config.Config, sdkGo *client.LMSdkGo, collector *models.Collector) error {
	currentVersion := collector.Build
	filename, err := DownloadInstaller(conf, sdkGo, collector)
	fmt.Println("bin file 1: ", filename)
	fmt.Println("err 1: ", err)
	if filename == "" && errors.Is(err, cerrors.VersionError) {
		collector.Build, conf.Version = "0", 0
		fmt.Println("retry to get latest available collector version")
		filename, err = DownloadInstaller(conf, sdkGo, collector)
		fmt.Println("bin file 2: ", filename)
		fmt.Println("err 2: ", err)
		if err != nil {
			return err
		}
	}
	installArgs := []string{"-y"}
	// # force update the collector object to ensure all details are up to date
	//    # e.g. build version
	if collector.Build != "0" {
		params := lm.NewGetCollectorByIDParams()
		params.ID = collector.ID
		c, err := sdkGo.LM.GetCollectorByID(params)
		if err == nil {
			fmt.Println("Collector version ", c.Payload.Build)
			currentVersion = c.Payload.Build
		}
	}
	if conf.Version >= constants.MIN_NONROOT_INSTALL_VER || conf.UseEa || conf.Version == 0 {
		installArgs = append(installArgs, "-u", "root")
	}
	if conf.ProxyUrl != "" {
		installArgs = append(installArgs, "-p", conf.Proxy.Host)
		if conf.ProxyUser != "" {
			installArgs = append(installArgs, "-U", conf.ProxyUser)
			if conf.ProxyPass != "" {
				installArgs = append(installArgs, "-P", conf.ProxyPass)
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
	if conf.IgnoreSSL {
		_, _, _ = util.Shellout("sed", "-i",
			"s/EnforceLogicMonitorSSL=true/EnforceLogicMonitorSSL=false/g",
			"/usr/local/logicmonitor/agent/conf/agent.conf")
	}
	fmt.Println("Cleaning up downloaded installer")
	if f, err := os.Open(constants.INSTALL_STAT_PATH); err == nil {
		defer func(f *os.File) {
			_ = f.Close()
		}(f)
		scanner := bufio.NewScanner(f)
		// optionally, resize scanner's capacity for lines over 64K, see next example
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
							fmt.Println(upgradeMsg)
						}
					}
				}

			}
		}
	}
	return nil
}

func DownloadInstaller(conf *config.Config, sdkGo *client.LMSdkGo, collector *models.Collector) (string, error) {
	fmt.Println("Downloading collector ", collector.ID)

	params := lm.NewGetCollectorInstallerParamsWithTimeout(10 * time.Minute)
	params.SetCollectorID(collector.ID)
	osAndArch := constants.DEFAULT_OS + "64"
	if strconv.IntSize == 32 {
		osAndArch = constants.DEFAULT_OS + "32"
	}
	params.SetOsAndArch(osAndArch)
	csize := conf.CollectorSize.String()
	fmt.Println("size: ", csize)
	params.SetCollectorSize(&csize)
	if conf.Version != 0 {
		params.SetCollectorVersion(&conf.Version)
	} else if collector.Build != "0" && !conf.UseEa {
		v, _ := strconv.ParseInt(collector.Build, 10, 32)
		v2 := int32(v)
		params.SetCollectorVersion(&v2)
	}
	filename := fmt.Sprintf("%slogicmonitorsetupx64_%d.bin", constants.TEMP_PATH, conf.CollectorID)

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return "", err
	}
	_, err = sdkGo.LM.GetCollectorInstaller(params, f)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "only those versions") {
			fmt.Printf("%s. Most likely the collector_version %d is invalid/out-dated. See https://www.logicmonitor.com/support/settings/collectors/collector-versions/ for more information on collector versioning\n", msg, params.CollectorVersion)
			return "", cerrors.VersionError
		}
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()
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

func Cleanup(*config.Config, *client.LMSdkGo) {

}

func NewCollector(conf *config.Config, sdkGo *client.LMSdkGo) (*models.Collector, error) {
	collector := &models.Collector{
		BackupAgentID:      conf.BackupCollectorID,
		EnableFailBack:     conf.EnableFailBack,
		EscalatingChainID:  conf.EscalatingChainID,
		ID:                 conf.CollectorID,
		ResendIval:         conf.ResendInterval,
		SuppressAlertClear: conf.SuppressAlertClear,
	}
	id, err := FindCollectorGroupID(conf.CollectorGroup, sdkGo)
	if err != nil {
		return nil, err
	}
	collector.CollectorGroupID = id
	if conf.Description != "" {
		collector.Description = conf.Description
	} else if hostname, err := os.Hostname(); err == nil {
		collector.Description = hostname

	}
	params := lm.NewAddCollectorParams()
	params.SetBody(collector)
	resp, err := sdkGo.LM.AddCollector(params)
	if err != nil {
		if es, ok := err.(*lm.AddCollectorDefault); ok && es.Payload.ErrorCode == 600 {
			// Status 600: The record already exists
			return collector, nil
		}
		return nil, err
	}
	collector = resp.Payload
	return collector, nil
}

func FindCollectorGroupID(collectorGroup string, sdkGo *client.LMSdkGo) (int32, error) {
	fmt.Println("Finding collector group " + collectorGroup)
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
	if conf.CollectorID != 0 {
		params := lm.NewGetCollectorByIDParams()
		params.ID = conf.CollectorID
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
				//return c, nil

			}
		}
	}
	return nil, errors.New("collector not found")
}
