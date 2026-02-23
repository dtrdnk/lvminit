// Package lvm pkg/lvm/lvmutil.go
package lvm

import (
	"os/exec"
	"strings"
)

// VgExists checks if a volume group exists
func VgExists(name string) bool {
	cmd := exec.Command("vgs", "--noheadings", "-o", "vg_name")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	for _, l := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(l) == name {
			return true
		}
	}
	return false
}

// PvExists checks if a physical volume exists
func PvExists(dev string) bool {
	cmd := exec.Command("pvs", "--noheadings", "-o", "pv_name")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	for _, l := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(l) == dev {
			return true
		}
	}
	return false
}
