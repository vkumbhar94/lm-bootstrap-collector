package collector

import (
	"context"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/logicmonitor/lm-sdk-go/client"
	"github.com/logicmonitor/lm-sdk-go/client/lm"
	"github.com/logicmonitor/lm-sdk-go/models"
	"github.com/sirupsen/logrus"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/constants"
)

func Shutdown(logger logrus.FieldLogger, conf *config.Config, sdkGo *client.LMSdkGo) error {
	logger.Infof("Shutting Down")

	// DON'T DELETE EXISTING COLLECTOR IF COLLECTOR_ID SPECIFIED
	if _, err := os.Stat(constants.CollectorFound); errors.Is(err, os.ErrNotExist) && conf.Cleanup {
		collector, err := FindCollector(conf, sdkGo)
		if err != nil {
			return err
		}
		logger.Infof("Uninstalling collector")
		err = DeleteCollector(logger, sdkGo, collector)
		if err != nil {
			return err
		}
	}

	logger.Infof("Shutdown complete")
	return nil
}

func DeleteCollector(logger logrus.FieldLogger, sdkGo *client.LMSdkGo, collector *models.Collector) error {
	params := lm.NewDeleteCollectorByIDParams()
	params.SetID(collector.ID)
	_, err := sdkGo.LM.DeleteCollectorByID(params)
	if err != nil {
		if GetHTTPStatusCodeFromLMSDKError(err) == http.StatusNotFound {
			logger.Infof("Collector does not exists, seems already deleted")
			return nil
		}
		return err
	}
	return nil
}

// GetHTTPStatusCodeFromLMSDKError get code
func GetHTTPStatusCodeFromLMSDKError(err error) int {
	if err == nil {
		return -2
	}
	if errors.Is(err, context.DeadlineExceeded) {
		// 408 client timeout error
		return http.StatusRequestTimeout
	}
	errRegex := regexp.MustCompile(`(?P<api>\[.*\])\[(?P<code>\d+)\].*`)
	matches := errRegex.FindStringSubmatch(err.Error())
	if len(matches) < 3 { // nolint: gomnd
		return -1
	}

	code, err := strconv.Atoi(matches[2])
	if err != nil {
		return -1
	}

	return code
}
