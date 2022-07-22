package collector

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/util"
)

func Apply(logger logrus.FieldLogger, cf *config.CollectorConf) error {
	// return ApplyConf(logger, "agent.conf-test", pkg.Properties, cf)
	return ApplyConf(logger, pkg.AgentConf, pkg.Properties, cf)
}

func ApplyConf(logger logrus.FieldLogger, confFile string, configFormat pkg.ConfigFormat, cf *config.CollectorConf) error {
	err := Backup(confFile, false)
	if err != nil {
		logger.Warnf("Failed to take backup with error: %s", err)
	}

	freshConfig := false
	if exists, err := util.FileExists(confFile); err == nil && !exists {
		freshConfig = true
	}

	size, modTime := int64(0), time.Now()
	if !freshConfig {
		fi, err := os.Stat(confFile)
		if err == nil {
			size, modTime = fi.Size(), fi.ModTime()
		}
	}

	var collectorIndex int
	if cf.DebugIndex != nil {
		collectorIndex = *cf.DebugIndex
	} else {
		collectorIndex, err = config.GetCollectorIndex()
		if err != nil {
			return fmt.Errorf("cannot retrieve collector index: %w", err)
		}
	}

	file, err := os.ReadFile(confFile)
	if err != nil {
		file = []byte{}
	}
	var updatedConf []byte
	switch configFormat {
	case pkg.Properties:
		updatedConf, err = ApplyPropertiesFile(logger, file, cf, collectorIndex)
		if err != nil {
			return fmt.Errorf("error while updating configuration: %w", err)
		}
	}

	if !freshConfig {
		fi2, err := os.Stat(confFile)
		if err != nil {
			return fmt.Errorf("failed to get file info while writing: %w", err)
		}
		if fi2.Size() != size || modTime != fi2.ModTime() {
			return fmt.Errorf("file changed after read")
		}
	}
	err = os.WriteFile(confFile, updatedConf, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error while writing updated configuration: %w", err)
	}

	return nil
}

func ApplyPropertiesFile(logger logrus.FieldLogger, confFile []byte, cf *config.CollectorConf, collectorIndex int) ([]byte, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(confFile)))
	//file, err := os.OpenFile(confFile, os.O_RDWR, fs.ModePerm)
	//if err != nil {
	//	return err
	//}
	//defer func(file *os.File) {
	//	_ = file.Close()
	//}(file)

	confKeys := make(map[string]*struct {
		visited bool
		index   int
	}, 0)

	for idx, kv := range cf.AgentConf {
		confKeys[kv.Key] = &struct {
			visited bool
			index   int
		}{visited: false, index: idx}
	}

	var lines []string
	// kvMap := map[string]string{}
	// scanner := bufio.NewScanner(file)
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
	if err := scanner.Err(); err != nil {
		return []byte{}, err
	}
	for k, v := range confKeys {
		if !v.visited {
			val := build(logger, "", cf.AgentConf[v.index], collectorIndex)
			lines = append(lines, k+"="+val)
		}
	}

	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

var ErrorNoBackup = errors.New("no file to take backup")

func Backup(file string, override bool) error {
	backupFile := file + ".bkp"
	if exists, err := util.FileExists(backupFile); err == nil && !exists {
		override = true
	}
	// if exists, err := util.FileExists(file +".bkp"); err == nil && !exists {
	if override {
		if exists, err := util.FileExists(file); err == nil && !exists {
			return fmt.Errorf("file does not exist to take backup: %w", ErrorNoBackup)
		}
		r, err := os.Open(file)
		if err != nil {
			return err
		}
		defer func(r *os.File) {
			_ = r.Close()
		}(r)
		w, err := os.Create(backupFile)
		if err != nil {
			return err
		}
		defer func(w *os.File) {
			_ = w.Close()
		}(w)
		_, err = w.ReadFrom(r)
		if err != nil {
			return err
		}
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
	case config.Csv, config.BitwiseOR:
		if reflect.TypeOf(values).Kind() == reflect.Map {
			return ""
		}
		var arr []string
		if val, ok := values.([]any); ok {
			for _, v := range val {
				arr = append(arr, fmt.Sprintf("%v", v))
			}
		}
		if dontOverride {
			prevArr := strings.Split(s, format.Separator())
			prevArrMap := make(map[string]bool)
			for _, v := range prevArr {
				if s != "" {
					prevArrMap[v] = false
				}
			}
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

		return strings.Join(arr, format.Separator())
	case config.Json:
		if dontOverride {
			var previous any
			err := json.Unmarshal([]byte(s), &previous)
			if err != nil {
				logger.Warnf("cannot retain old config value: %s", s)
			} else {
				pt := reflect.TypeOf(previous)
				if vt := reflect.TypeOf(values); vt == pt {
					switch pt.Kind() {
					case reflect.Map:
						if valuesMap, ok := values.(map[string]any); ok {
							if m, ok2 := previous.(map[string]any); ok2 {
								for k, v := range m {
									if _, ok := valuesMap[k]; !ok {
										valuesMap[k] = v
									}
								}
								values = valuesMap
							}
						}
					case reflect.Array, reflect.Slice:
						valuesArr := values.([]any)
						valuesMap := make(map[any]struct{}, 0)
						for _, v := range valuesArr {
							valuesMap[v] = struct{}{}
						}
						for _, v := range previous.([]any) {
							if _, ok := valuesMap[v]; !ok {
								valuesArr = append(valuesArr, v)
							}
						}
						values = valuesArr
					}
				} else {
					logger.Warnf("type mismatch hence cannot retain old config value: %s", s)
				}
			}
		}
		marshal, err := json.Marshal(values)
		if err != nil {
			return ""
		}
		return string(marshal)
	}
	return ""
}
