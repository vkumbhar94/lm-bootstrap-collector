package logicmonitor

import (
	"bytes"
	"errors"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/logicmonitor/lm-sdk-go/client"
	"github.com/logicmonitor/lm-sdk-go/client/lm"
	"github.com/sirupsen/logrus"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"io"
	"net/http"
	"net/url"
)

func NewLMClient(argusConfig *config.Config) (*client.LMSdkGo, error) {
	conf := client.NewConfig()
	conf.SetAccessID(&argusConfig.AccessID)
	conf.SetAccessKey(&argusConfig.AccessKey)
	domain := argusConfig.Account + ".logicmonitor.com"
	conf.SetAccountDomain(&domain)
	// conf.UserAgent = constants.UserAgentBase + constants.Version
	if argusConfig.ProxyUrl == "" {
		if argusConfig.IgnoreSSL {
			return newLMClientWithoutSSL(conf)
		}

		c := NewClient(conf)
		return c, nil
	}

	return newLMClientWithProxy(conf, argusConfig)
}

func NewClient(c *client.Config) *client.LMSdkGo {
	transport := httptransport.New(c.TransportCfg.Host, c.TransportCfg.BasePath, c.TransportCfg.Schemes)
	authInfo := client.LMv1Auth(*c.AccessID, *c.AccessKey)

	cli := new(client.LMSdkGo)
	transport.Consumers["application/binary"] = LMBinaryFileConsumer()
	cli.Transport = transport

	cli.LM = lm.New(transport, strfmt.Default, authInfo)

	return cli
}

func LMBinaryFileConsumer() runtime.Consumer {
	return runtime.ConsumerFunc(func(reader io.Reader, data interface{}) error {
		if reader == nil {
			return errors.New("LMBinaryFileConsumer requires a reader") // early exit
		}
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(reader)
		if err != nil {
			return err
		}
		b := buf.Bytes()
		w, ok := data.(io.Writer)
		if !ok {
			return errors.New("provided output object is not of type io.writer")
			// the assertion failed.
		}
		_, err = w.Write(b)
		if err != nil {
			return err
		}
		c, ok := w.(interface{ Close() })
		if ok {
			c.Close()
		}
		return nil
	})
}

func newLMClientWithProxy(config *client.Config, argusConfig *config.Config) (*client.LMSdkGo, error) {
	proxyURL, err := url.Parse(argusConfig.ProxyUrl)
	if err != nil {
		return nil, err
	}
	if argusConfig.ProxyUser != "" {
		if argusConfig.ProxyPass != "" {
			proxyURL.User = url.UserPassword(argusConfig.ProxyUser, argusConfig.ProxyPass)
		} else {
			proxyURL.User = url.User(argusConfig.ProxyUser)
		}
	}
	logrus.Infof("Using http/s proxy: %s with username: %s", argusConfig.ProxyUrl, argusConfig.ProxyUser)
	httpClient := http.Client{
		Transport: &http.Transport{ // nolint: exhaustivestruct
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	transport := httptransport.NewWithClient(config.TransportCfg.Host, config.TransportCfg.BasePath, config.TransportCfg.Schemes, &httpClient)
	authInfo := client.LMv1Auth(*config.AccessID, *config.AccessKey)
	clientObj := new(client.LMSdkGo)
	clientObj.Transport = transport
	clientObj.LM = lm.New(transport, strfmt.Default, authInfo)

	return clientObj, nil
}

func newLMClientWithoutSSL(config *client.Config) (*client.LMSdkGo, error) {
	opts := httptransport.TLSClientOptions{InsecureSkipVerify: true}
	httpClient, err := httptransport.TLSClient(opts)
	if err != nil {
		return nil, err
	}
	transport := httptransport.NewWithClient(config.TransportCfg.Host, config.TransportCfg.BasePath, config.TransportCfg.Schemes, httpClient)
	authInfo := client.LMv1Auth(*config.AccessID, *config.AccessKey)
	cli := new(client.LMSdkGo)
	cli.Transport = transport
	cli.LM = lm.New(transport, strfmt.Default, authInfo)

	return cli, nil
}
