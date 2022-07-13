package collector

import (
	"errors"
	"fmt"
	"github.com/logicmonitor/lm-sdk-go/client"
	"github.com/logicmonitor/lm-sdk-go/client/lm"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/constants"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/util"
	"math"
	"os"
	"strconv"
	"time"
)

func Start(conf *config.Config, client *client.LMSdkGo) error {
	err := FindCollector(conf, client)
	if err != nil {
		fmt.Println("collector not found")
		if conf.Kubernetes {
			return fmt.Errorf("running in kubernetes but collector not found: %w", err)
		}
		// TODO: create collector from config
		//NewCollector(conf, client)
	} else {
		fmt.Println("Collector Found")
		if _, err := os.Stat(constants.FIRST_RUN); errors.Is(err, os.ErrNotExist) {
			util.Touch(constants.COLLECTOR_FOUND)
		}
	}
	util.Touch(constants.FIRST_RUN)
	if _, err := os.Stat(constants.INSTALL_PATH + constants.AGENT_DIRECTORY); !errors.Is(err, os.ErrNotExist) {
		fmt.Println(`Collector already installed.`)
		Cleanup(conf, client)
		return nil
	}
	return Install(conf, client)
}

func Install(conf *config.Config, sdkGo *client.LMSdkGo) error {
	filename, err := DownloadInstaller(conf, sdkGo)
	if err != nil {
		return err
	}
	cmd := []string{"-y"}
	if conf.Version >= constants.MIN_NONROOT_INSTALL_VER || conf.UseEa || conf.Version == 0 {
		cmd = append(cmd, "-u", "root")
	}
	err, stdout, stderr := util.Shellout(filename, cmd...)
	if err != nil {
		return err
	}
	fmt.Println(stdout)
	fmt.Println(stderr)
	return nil
}

func DownloadInstaller(conf *config.Config, sdkGo *client.LMSdkGo) (string, error) {
	fmt.Println("Downloading installer")
	params := lm.NewGetCollectorInstallerParamsWithTimeout(10 * time.Minute)
	params.SetCollectorID(conf.CollectorID)
	params.SetOsAndArch("Linux64")
	csize := conf.CollectorSize.String()
	fmt.Println("size: ", csize)
	params.SetCollectorSize(&csize)
	params.SetCollectorVersion(&conf.Version)
	filename := fmt.Sprintf("%slogicmonitorsetupx64_%d.bin", constants.TEMP_PATH, conf.CollectorID)

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return "", err
	}
	_, err = sdkGo.LM.GetCollectorInstaller(params, f)
	if err != nil {
		return "", err
	}
	defer func() {
		err = f.Close()
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

func NewCollector(conf *config.Config, sdkGo *client.LMSdkGo) {

}

func FindCollector(conf *config.Config, sdkGo *client.LMSdkGo) error {
	if conf.CollectorID != 0 {
		params := lm.NewGetCollectorByIDParams()
		params.ID = conf.CollectorID
		_, err := sdkGo.LM.GetCollectorByID(params)
		if err != nil {
			return err
		}
		return nil
	} else {
		list, err := sdkGo.LM.GetCollectorList(nil)
		if err != nil {
			return err
		}
		for _, c := range list.Payload.Items {
			if c.Description == conf.Description {
				return nil
			}
		}
	}
	return nil
}
