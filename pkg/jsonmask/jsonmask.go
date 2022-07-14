package jsonmask

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

var defaultSensitiveData = []string{
	"name",
	"surName",
	"firstName",
	"lastName",
	"identification",
	"national",
	"card",
	"phone",
	"phoneNo",
	"number",
	"username",
	"password",
	"email",
	"address",
	"phoneNo",
}

type Mask interface {
	Json(b []byte) (*string, error)
}

type mask struct {
	sensitiveField []string
}

func Init(fields ...string) Mask {
	return &mask{
		sensitiveField: fields,
	}
}

func InitAppendToDefault(fields ...[]string) Mask {
	f := defaultSensitiveData
	if len(fields) > 0 {
		f = append(f, fields[0]...)
	}
	return &mask{
		sensitiveField: f,
	}
}

func (m mask) Json(b []byte) (*string, error) {
	var storage []any
	p := make(chan bool, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	p <- true
	err := m.walkThrough(b, &storage, p, &wg)
	if err != nil {
		return nil, err
	}
	wg.Wait()
	return masking(b, storage)
}

// walkThrough will recursive until no more array or map
func (m mask) walkThrough(b []byte, storage *[]any, p chan bool, wg *sync.WaitGroup) error {
	defer func() {
		<-p
		wg.Done()
	}()
	var gson any
	err := json.Unmarshal(b, &gson)
	if err != nil {
		return err
	}
	switch t := gson.(type) {
	case map[string]any:
		for k, v := range t {
			switch v := v.(type) {
			case string:
				m.sensitive(k, v, storage)
			case float64:
				m.sensitive(k, v, storage)
			case int32:
				m.sensitive(k, v, storage)
			case []any:
				for _, val := range v {
					err = m.next(val, storage, p, wg)
					if err != nil {
						return err
					}
				}
			case map[string]any:
				err = m.next(v, storage, p, wg)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (m mask) next(v any, storage *[]any, p chan bool, wg *sync.WaitGroup) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	wg.Add(1)
	p <- true
	go m.walkThrough(b, storage, p, wg)
	return nil
}

func (m mask) sensitive(k string, v any, storage *[]any) {
	ok := strings.Contains(strings.ToLower(fmt.Sprint(m.sensitiveField)), strings.ToLower(k))
	if ok {
		// append sensitive value to storage
		*storage = append(*storage, v)
	}
}

func masking(j []byte, d []any) (*string, error) {
	body := string(j)
	if len(d) == 0 {
		return &body, nil
	}
	for _, val := range d {
		body = strings.ReplaceAll(body, typeCasting(val.(any)), randomMask(typeCasting(val.(any))))
	}
	return &body, nil
}

func randomMask(c string) string {
	if len(c) == 0 {
		return c
	}
	r := []rune(c)
	cl := len(r)
	size := initMaskSize(cl)
	var count int
	raffle := make(map[int]int, size)
	for i := 0; i < cl; i++ {
		count += 1 // avoid random forever
		if len(raffle) == size || count == 10 {
			// break if mask enough
			break
		}
		v := randPos(cl)
		if _, ok := raffle[v]; ok {
			i -= 1
			continue
		}
		// case not mask yet
		if len(r)-1 >= v {
			r[v] = '*'
			raffle[v] = v
		}
	}
	return string(r)
}

func randPos(max int) int {
	source := rand.NewSource(time.Now().UnixNano())
	ra := rand.New(source)
	return ra.Intn(max)
}

func initMaskSize(l int) int {
	if l == 1 {
		return l
	}
	return l / 2
}

func typeCasting(d any) string {
	switch c := d.(type) {
	case string:
		return c
	case int64:
		return fmt.Sprint(c)
	case float64:
		return strconv.FormatFloat(c, 'f', -1, 64)
	default:
		return fmt.Sprint(d)
	}
}

func MaskJson(data any, keys ...string) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	m := Init(keys...) // optional
	t, err := m.Json(b)
	if err != nil {
		return "", err
	}
	return *t, nil
}
