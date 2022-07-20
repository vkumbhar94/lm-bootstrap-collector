package config

import (
	"fmt"
	"strings"
)

type CoalesceFormat uint

const (
	UnknownFormat CoalesceFormat = iota
	Json
	Csv
	BitwiseOR
)

func (cf *CoalesceFormat) Set(v string) error {
	err := cf.UnmarshalText([]byte(v))
	if err != nil {
		return err
	}
	return nil
}

func (cf *CoalesceFormat) UnmarshalText(text []byte) error {
	s := string(text)
	switch strings.ToLower(s) {
	case "json":
		*cf = Json
	case "csv", ",":
		*cf = Csv
	case "bitOR", "|":
		*cf = BitwiseOR
	default:
		*cf = UnknownFormat
		return fmt.Errorf("unknown format: %s", s)
	}
	return nil
}

func (cf *CoalesceFormat) MarshalText() ([]byte, error) {
	switch *cf {
	case Json:
		return []byte("json"), nil
	case Csv:
		return []byte("csv"), nil
	case BitwiseOR:
		return []byte("bitOR"), nil
	case UnknownFormat:
		return []byte("unknown"), nil
	default:
		return []byte(""), fmt.Errorf("cannot marshal: %v", *cf)
	}
}

func (cf *CoalesceFormat) String() string {
	text, err := cf.MarshalText()
	if err != nil {
		return ""
	}
	return string(text)
}

type KeyValue struct {
	Key            string          `json:"key"`
	Discrete       bool            `json:"discrete"`
	Value          any             `json:"value"`
	Values         []any           `json:"values"`
	ValuesList     [][]any         `json:"valuesList"`
	CoalesceFormat *CoalesceFormat `json:"coalesceFormat"`
	ForceQuote     bool            `json:"forceQuote"`
	DontOverride   bool            `json:"dontOverride"`
}

type CollectorConf struct {
	DebugIndex *int        `json:"debugIndex"`
	AgentConf  []*KeyValue `json:"agent.conf"`
}

func (cc *CollectorConf) Validate() error {
	for _, v := range cc.AgentConf {
		if v.CoalesceFormat == nil {
			a := Csv
			v.CoalesceFormat = &a
		}
	}
	return nil
}
