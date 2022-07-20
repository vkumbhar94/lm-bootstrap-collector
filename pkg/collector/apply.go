package collector

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"io/fs"
	"os"
	"reflect"
	"strings"
)

func Apply(logger logrus.FieldLogger, cf *config.CollectorConf) error {
	logger.Info("Applying...")
	fi, err := os.Stat("agent.conf")
	if err != nil {
		return err
	}
	size, modTime := fi.Size(), fi.ModTime()
	file, err := os.OpenFile("agent.conf", os.O_RDWR, fs.ModePerm)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	var confKeys = make(map[string]*struct {
		visited bool
		index   int
	}, 0)

	for idx, kv := range cf.AgentConf {
		confKeys[kv.Key] = &struct {
			visited bool
			index   int
		}{visited: false, index: idx}
	}

	var collectorIndex int
	if cf.DebugIndex != nil {
		collectorIndex = *cf.DebugIndex
	} else {
		collectorIndex, err = config.GetCollectorIndex()
		if err != nil {
			return err
		}
	}

	var lines []string
	//kvMap := map[string]string{}
	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		kv := strings.SplitN(scanner.Text(), "=", 2)
		if len(kv) < 2 {
			logger.Warnf("cannot parse config line: %s", scanner.Text())
			continue
		}
		if meta, ok := confKeys[kv[0]]; ok && !meta.visited {
			val := build(logger, kv[1], cf.AgentConf[meta.index], collectorIndex)
			meta.visited = true
			lines = append(lines, kv[0]+"="+val)
		} else {
			lines = append(lines, scanner.Text())
		}
	}
	for k, v := range confKeys {
		if !v.visited {
			val := build(logger, "", cf.AgentConf[v.index], collectorIndex)
			lines = append(lines, k+"="+val)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	fi2, err := os.Stat("agent.conf")
	if err != nil {
		return err
	}
	if fi2.Size() != size || modTime != fi2.ModTime() {
		return fmt.Errorf("file changed after read")
	}

	err = file.Truncate(0)
	if err != nil {
		return err

	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(strings.Join(lines, "\n") + "\n"))
	if err != nil {
		return err
	}

	return nil
}

func build(logger logrus.FieldLogger, s string, v *config.KeyValue, index int) string {
	val := ""
	if v.Discrete {
		if len(v.Values) > 0 && len(v.Values) > index {
			t := reflect.TypeOf(v.Values[index])
			if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
				val = coalesce(logger, s, v.Values[index], *v.CoalesceFormat, v.DontOverride)
			} else if t.Kind() == reflect.Map {
				val = coalesce(logger, s, v.Values[index], *v.CoalesceFormat, v.DontOverride)
			} else {
				val = fmt.Sprintf("%v", v.Values[index])
			}
		} else if len(v.ValuesList) > 0 && len(v.ValuesList) > index {
			val = coalesce(logger, s, v.ValuesList[index], *v.CoalesceFormat, v.DontOverride)
		}
	} else {
		if len(v.Values) > 0 {
			val = coalesce(logger, s, v.Values, *v.CoalesceFormat, v.DontOverride)
		} else {
			t := reflect.TypeOf(v.Value)
			if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
				val = coalesce(logger, s, v.Value, *v.CoalesceFormat, v.DontOverride)
			} else if t.Kind() == reflect.Map {
				val = coalesce(logger, s, v.Value, *v.CoalesceFormat, v.DontOverride)
			} else {
				val = fmt.Sprintf("%v", v.Value)
			}
		}
	}

	return val
}

func coalesce(logger logrus.FieldLogger, s string, values any, format config.CoalesceFormat, dontOverride bool) string {
	switch format {
	case config.Csv:
		if reflect.TypeOf(values).Kind() == reflect.Map {
			return ""
		}
		prevArr := strings.Split(s, ",")
		prevArrMap := make(map[string]bool)
		for _, v := range prevArr {
			if s != "" {
				prevArrMap[v] = false
			}
		}
		var arr []string
		if val, ok := values.([]any); ok {
			for _, v := range val {
				arr = append(arr, fmt.Sprintf("%v", v))
			}
		}
		if dontOverride {
			for _, v := range arr {
				if _, ok := prevArrMap[v]; ok {
					prevArrMap[v] = true
				}
			}
			for k, v := range prevArrMap {
				if !v {
					arr = append(arr, k)
				}
			}
		}

		return strings.Join(arr, ",")
	case config.Json:
		marshal, err := json.Marshal(values)
		if err != nil {
			return ""
		}
		return string(marshal)
	}
	return ""
}
