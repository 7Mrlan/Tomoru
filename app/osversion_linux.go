package main

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"
)

// osVersion is the system and its version, for a report of a problem: the distribution's name, as
// os-release gives it, "Ubuntu 24.04.3 LTS".
func osVersion() string {
	for _, path := range []string{"/etc/os-release", "/usr/lib/os-release"} {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(bytes.NewReader(b))
		for sc.Scan() {
			if v, ok := strings.CutPrefix(sc.Text(), "PRETTY_NAME="); ok {
				if u, err := strconv.Unquote(v); err == nil {
					v = u
				}
				return v
			}
		}
	}
	return "Linux"
}
