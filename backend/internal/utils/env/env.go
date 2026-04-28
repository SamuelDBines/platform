package env

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func String(key string, def ...string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	panic("missing env: " + key)
}

func Int(key string, def ...int) int {
	if v, ok := os.LookupEnv(key); ok {
		i, err := strconv.Atoi(v)
		if err != nil {
			panic("invalid int env " + key + ": " + v)
		}

		return i
	}

	if len(def) > 0 {
		return def[0]
	}

	panic("missing env: " + key)
}

func Bool(key string, def ...bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			return b
		}
	}
	if len(def) > 0 {
		return def[0]
	}
	panic("missing env: " + key)
}

func Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		if _, set := os.LookupEnv(key); set {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("setenv %q: %w", key, err)
		}
	}
	return sc.Err()
}
