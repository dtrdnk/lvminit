package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"lvminit/pkg/lvm"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	Mode         string     `yaml:"mode"` // create (default) or destroy
	VolumeGroups []VGConfig `yaml:"volumeGroups"`
}

type VGConfig struct {
	Name    string   `yaml:"name"`
	Devices []string `yaml:"devices"`
}

// JSON logging helpers

func logJSON(level string, msg string, kv ...interface{}) {
	m := map[string]interface{}{
		"level": level,
		"msg":   msg,
		"time":  time.Now().Format(time.RFC3339),
	}
	for i := 0; i+1 < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			continue
		}
		m[k] = kv[i+1]
	}
	b, _ := json.Marshal(m)
	fmt.Fprintln(os.Stderr, string(b))
}

func logInfo(msg string, kv ...interface{})  { logJSON("info", msg, kv...) }
func logWarn(msg string, kv ...interface{})  { logJSON("warning", msg, kv...) }
func logError(msg string, kv ...interface{}) { logJSON("error", msg, kv...) }
func logFatal(msg string, kv ...interface{}) {
	logJSON("error", msg, kv...)
	os.Exit(1)
}

func scanBlockDevices() ([]string, error) {
	devices := []string{}
	paths, err := ioutil.ReadDir("/dev")
	if err != nil {
		logError("Failed to read /dev", "error", err)
		return nil, err
	}
	for _, p := range paths {
		n := p.Name()
		if strings.HasPrefix(n, "sd") || strings.HasPrefix(n, "nvme") || strings.HasPrefix(n, "loop") {
			full := "/dev/" + n
			if _, err := os.Stat(full); err == nil {
				devices = append(devices, full)
			}
		}
	}
	return devices, nil
}

func runLvmCmd(args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logError("LVM command failed", "args", args, "output", string(output), "error", err)
		return fmt.Errorf("LVM cmd failed: %v, output: %s", err, string(output))
	}
	return nil
}

// Get current size of the device in bytes
func getDeviceSize(dev string) (int64, error) {
	out, err := exec.Command("blockdev", "--getsize64", dev).CombinedOutput()
	if err != nil {
		logError("Failed to get device size", "device", dev, "output", string(out), "error", err)
		return 0, fmt.Errorf("failed to get size for %s: %v, out: %s", dev, err, string(out))
	}
	var size int64
	_, err = fmt.Sscanf(string(out), "%d", &size)
	return size, err
}

// Map PV device to current size
func getPVDeviceSizes(pvs []string) map[string]int64 {
	result := make(map[string]int64)
	for _, dev := range pvs {
		sz, err := getDeviceSize(dev)
		if err != nil {
			logWarn("Couldn't get device size", "device", dev, "error", err)
		} else {
			result[dev] = sz
		}
	}
	return result
}

func createAll(cfg *Config) {
	for _, vg := range cfg.VolumeGroups {
		var pvArgs []string
		for _, pv := range vg.Devices {
			logInfo("Ensuring PV", "device", pv)
			err := runLvmCmd("pvcreate", "-ff", "-y", pv)
			if err != nil {
				if strings.Contains(fmt.Sprint(err), "is already a physical volume") {
					logInfo("PV exists", "device", pv)
				} else {
					logFatal("pvcreate failed", "device", pv, "error", err)
				}
			}
			pvArgs = append(pvArgs, pv)
		}
		logInfo("Ensuring VG", "vg", vg.Name)
		args := append([]string{"vgcreate", vg.Name}, pvArgs...)
		err := runLvmCmd(args...)
		if err != nil {
			if strings.Contains(fmt.Sprint(err), "already exists") {
				logInfo("VG exists", "vg", vg.Name)
			} else {
				logFatal("vgcreate failed", "vg", vg.Name, "error", err)
			}
		}
	}
}

func destroyAll(cfg *Config) {
	deadline := time.Now().Add(3 * time.Minute)
	vgs := map[string]bool{}
	for _, vg := range cfg.VolumeGroups {
		vgs[vg.Name] = true
	}

	for {
		allGone := true
		// Try to remove VGs + LVs first (idempotent)
		for _, vg := range cfg.VolumeGroups {
			if lvm.VgExists(vg.Name) {
				allGone = false
				logInfo("Destroying VG", "vg", vg.Name)
				// Remove all LVs (force)
				runLvmCmd("lvremove", "-A", "n", "-f", vg.Name)
				// Now try VG remove
				err := runLvmCmd("vgremove", "-f", vg.Name)
				if err != nil {
					logWarn("vgremove failed", "vg", vg.Name, "error", err)
				}
			}
			// Remove PVs (idempotent)
			for _, pv := range vg.Devices {
				if lvm.PvExists(pv) {
					allGone = false
					logInfo("Destroying PV", "device", pv)
					err := runLvmCmd("pvremove", "-ff", "-y", pv)
					if err != nil {
						logWarn("pvremove failed", "device", pv, "error", err)
					}
				}
			}
		}
		if allGone || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Second)
	}
}

// Resize PVs when underlying block device increases in size
func resizeAll(cfg *Config, lastSizes map[string]int64) map[string]int64 {
	logInfo("Checking for PV/underlying size changes...")
	outSizes := getPVDeviceSizes(flattenDevices(cfg))
	for _, vg := range cfg.VolumeGroups {
		for _, pv := range vg.Devices {
			oldSz := lastSizes[pv]
			newSz := outSizes[pv]
			if newSz > 0 && newSz > oldSz {
				logInfo("Resizing PV", "device", pv, "oldSizeBytes", oldSz, "newSizeBytes", newSz)
				if err := runLvmCmd("pvresize", pv); err != nil {
					logWarn("pvresize failed", "device", pv, "error", err)
				}
			}
		}
	}
	return outSizes
}

func flattenDevices(cfg *Config) []string {
	devs := []string{}
	for _, vg := range cfg.VolumeGroups {
		for _, dev := range vg.Devices {
			devs = append(devs, dev)
		}
	}
	return devs
}

func main() {
	if len(os.Args) < 2 {
		logFatal("Usage: lvminit config.yaml")
	}
	cfgFile := os.Args[1]
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		logFatal("Failed to read config", "error", err)
	}
	var cfg Config
	if err := yaml.UnmarshalStrict(data, &cfg); err != nil {
		logFatal("Failed to parse config", "error", err)
	}

	mode := "create"
	if cfg.Mode != "" {
		mode = strings.ToLower(cfg.Mode)
	}

	if devs, err := scanBlockDevices(); err == nil {
		logInfo("Available block devices", "devices", devs)
	}

	switch mode {
	case "create":
		createAll(&cfg)
		logInfo("LVM create/setup completed.")

		lastSizes := getPVDeviceSizes(flattenDevices(&cfg))
		for {
			time.Sleep(60 * time.Second)
			lastSizes = resizeAll(&cfg, lastSizes)
		}
	case "destroy":
		for {
			destroyAll(&cfg)
			logInfo("LVM destroy pass completed.")
			time.Sleep(60 * time.Second)
		}
	default:
		logFatal(fmt.Sprintf("Unknown mode '%s'. Valid: create (default), destroy", mode))
	}
}
