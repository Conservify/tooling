package tooling

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
)

type PropertiesMap struct {
	Properties map[string]string
}

func NewPropertiesMapFromFile(fn string) (pf *PropertiesMap, err error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return NewPropertiesMap(f)
}

func NewPropertiesMap(rd io.Reader) (pf *PropertiesMap, err error) {
	properties := make(map[string]string)

	cre := regexp.MustCompile("#.+")
	r := bufio.NewReader(rd)
	for {
		l, err := r.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		l = cre.ReplaceAllString(strings.TrimSpace(l), "")
		parts := strings.SplitN(l, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			// fmt.Printf("%s = %s\n", k, v)
			properties[k] = v
		}
	}
	pf = &PropertiesMap{Properties: properties}
	return
}

var ReplaceRe = regexp.MustCompile("{\\S+?}")

func (pf *PropertiesMap) Lookup(key string, ctx map[string]string) (string, error) {
	val, ok := ctx[key]
	if !ok {
		if val, ok = pf.Properties[key]; !ok {
			return "", fmt.Errorf("No key %s", key)
		}
	}
	return ReplaceRe.ReplaceAllStringFunc(val, func(group string) string {
		k := group[1 : len(group)-1]

		// Look for a platform specific version.
		if runtime.GOOS == "windows" {
			if winVal, ok := pf.Lookup(k+".windows", ctx); ok == nil {
				return winVal
			}
		}
		v, err := pf.Lookup(k, ctx)
		if err != nil {
			return fmt.Sprintf("{%s}", err)
		}
		return v
	}), nil
}

func (pf *PropertiesMap) ToSubtree(prefix string) *PropertiesMap {
	m := make(map[string]string)

	for k, v := range pf.Properties {
		if strings.Index(k, prefix) == 0 {
			m[k[len(prefix)+1:]] = v
		}
	}

	return &PropertiesMap{Properties: m}
}

func (pf *PropertiesMap) Merge(c *PropertiesMap) *PropertiesMap {
	m := make(map[string]string)

	for k, v := range pf.Properties {
		m[k] = v
	}

	for k, v := range c.Properties {
		m[k] = v
	}

	return &PropertiesMap{Properties: m}
}

func (pf *PropertiesMap) ToString(key string) *string {
	if s, ok := pf.Properties[key]; ok {
		return &s
	}
	return nil
}

func (pf *PropertiesMap) ToBool(key string) bool {
	if s := pf.ToString(key); s != nil {
		return *s == "true"
	}
	return false
}
