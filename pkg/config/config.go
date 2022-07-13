package config

import (
	"errors"
	"fmt"
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
	return "CollectorSize"
}

type Config struct {
	Account            string
	AccessID           string
	AccessKey          string
	BackupCollectorID  int32
	CollectorSize      CollectorSize
	Cleanup            bool
	CollectorGroup     string
	Version            int32
	Description        string
	EnableFailBack     bool
	EscalatingChainID  int32
	CollectorID        int32
	ResendInterval     int32
	SuppressAlertClear bool
	UseEa              bool
	Kubernetes         bool
	ProxyUrl           string
	ProxyUser          string
	ProxyPass          string
	IgnoreSSL          bool
	IDS                string
	Debug              bool
	DebugIndex         int
}

func (c *Config) Validate() error {
	if c.Account == "" {
		return fmt.Errorf("account must be set")
	}
	if c.AccessID == "" {
		return fmt.Errorf("access id must be set")
	}
	if c.AccessKey == "" {
		return fmt.Errorf("access key must be set")
	}
	if c.CollectorSize == Unknown {
		//fmt.Println("setting default collector size to small")
		c.CollectorSize = Small
	}
	if !c.Kubernetes && c.CollectorID == 0 && c.Description == "" {
		return fmt.Errorf(`\"collector_id\" or \"description\" must be set in non kubernetes environments`)
	}

	if c.Kubernetes {
		if err := c.setK8sCollectorID(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) setK8sCollectorID() error {
	var err error
	hostname, err := os.Hostname()
	//hostname := "abc-0"
	if err != nil {
		return err
	}
	var index int
	if c.Debug {
		index = c.DebugIndex
	} else {
		arr := strings.Split(hostname, "-")
		if len(arr) == 0 {
			return fmt.Errorf("hostname cannot parse: %s to get index", hostname)
		}
		index, err = strconv.Atoi(arr[len(arr)-1])
		if err != nil {
			return fmt.Errorf("collecor ID index parse failed with: %w", err)
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
	c.CollectorID = int32(cid)
	return nil
}
