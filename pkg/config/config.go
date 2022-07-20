package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type CollectorSize uint32

const (
	Unknown CollectorSize = iota
	Nano
	Small
	Medium
	Large
	ExtraLarge
	DoubleExtraLarge
)

func (of *CollectorSize) String() string {
	switch *of {
	case Nano:
		return "nano"
	case Small:
		return "small"
	case Medium:
		return "medium"
	case Large:
		return "large"
	case ExtraLarge:
		return "extra_large"
	case DoubleExtraLarge:
		return "double_extra_large"
	case Unknown:
		return "unknown"
	}
	return "unknown"
}

func (of *CollectorSize) Set(v string) error {
	switch v {
	case "nano":
		*of = Nano
	case "small":
		*of = Small
	case "medium":
		*of = Medium
	case "large":
		*of = Large
	case "extra_large":
		*of = ExtraLarge
	case "double_extra_large":
		*of = DoubleExtraLarge
	default:
		return errors.New(`must be one of "nano", "small", "medium", "large", "extra_large", or "double_extra_large". ( Default: small ) `)
	}
	return nil
}

// Type is only used in help text
func (of *CollectorSize) Type() string {
	return "Size"
}

// Creds config for lm client connection - persistent flags
type Creds struct {
	Account   string
	AccessID  string
	AccessKey string
	ProxyUrl  string
	ProxyUser string
	ProxyPass string
	IgnoreSSL bool

	// created internally from proxy parameters
	Proxy *url.URL
}

func (c *Creds) Validate() error {
	if c.Account == "" {
		return fmt.Errorf("account must be set")
	}
	if c.AccessID == "" {
		return fmt.Errorf("access id must be set")
	}
	if c.AccessKey == "" {
		return fmt.Errorf("access key must be set")
	}
	err := c.parseProxy()
	if err != nil {
		return err
	}
	return nil
}

func (c *Creds) parseProxy() error {
	if c.ProxyUrl == "" {
		return nil
	}
	var err error
	c.Proxy, err = url.Parse(c.ProxyUrl)
	if err != nil {
		return err
	}

	return nil
}

// Config Start ShutDown Config
type Config struct {
	BackupCollectorID  int32
	Size               CollectorSize
	Cleanup            bool
	Group              string
	Version            int32
	Description        string
	EnableFailBack     bool
	EscalatingChainID  int32
	ID                 int32
	ResendInterval     int32
	SuppressAlertClear bool
	UseEa              bool
	Kubernetes         bool

	IDS        string
	Debug      bool
	DebugIndex int
}

func (c *Config) Validate() error {
	if c.Size == Unknown {
		// fmt.Println("setting default collector size to small")
		c.Size = Small
	}
	if !c.Kubernetes && c.ID == 0 && c.Description == "" {
		return fmt.Errorf(`\"collector_id\" or \"description\" must be set in non kubernetes environments`)
	}

	if c.Kubernetes {
		if err := c.setK8sCollectorID(); err != nil {
			return err
		}
	}

	//     if not kwargs['use_ea']:
	//        if 'extra_large' in kwargs['collector_size'] or 'double_extra_large' in kwargs['collector_size']:
	//            err = 'Cannot proceed with installation because only Early Access collector versions support ' + kwargs[
	//                'collector_size'] + 'size. To proceed further with installation, set \"use_ea\" parameter to true or use appropriate collector size.\n'
	if !c.UseEa && (c.Size == ExtraLarge || c.Size == DoubleExtraLarge) {
		err := fmt.Errorf("cannot proceed with installation because only Early Access collector versions support " + c.Size.String() + "size. To proceed further with installation, set \"use_ea\" parameter to true or use appropriate collector size")
		return err
	}

	return nil
}

func (c *Config) setK8sCollectorID() error {
	var err error
	var index int
	if c.Debug {
		index = c.DebugIndex
	} else {
		index, err = GetCollectorIndex()
		if err != nil {
			return err
		}
	}

	ids := strings.Split(c.IDS, ",")
	if len(ids) < index+1 {
		return fmt.Errorf("unable to parse collector id from ids list: %s", c.IDS)
	}
	cid, err := strconv.ParseInt(ids[index], 10, 32)
	if err != nil {
		return fmt.Errorf("parsing collector failed with: %w", err)
	}
	c.ID = int32(cid)
	return nil
}

func GetCollectorIndex() (int, error) {
	var err error
	hostname, err := os.Hostname()
	if err != nil {
		return 0, err
	}
	arr := strings.Split(hostname, "-")
	if len(arr) == 0 {
		return 0, fmt.Errorf("hostname cannot parse: %s to get index", hostname)
	}
	index, err := strconv.Atoi(arr[len(arr)-1])
	if err != nil {
		return 0, fmt.Errorf("collecor ID index parse failed with: %w", err)
	}
	return index, nil
}
