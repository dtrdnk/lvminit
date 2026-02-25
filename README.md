# LVMInit
#### This project was created to prepare disks for TopoLVM running in Talos Linux
#### Inspired by [lvm-disk-watcher](https://github.com/trueforge-org/containerforge/tree/main/apps/lvm-disk-watcher) and [csi-driver-lvm](https://github.com/metal-stack/csi-driver-lvm)
**lvminit** is an LVM initialization and automation tool designed for Kubernetes clusters, supporting creation and destruction of LVM volume groups on block devices (including loop devices for test/dev use). It can be run as a DaemonSet and is fully testable with Kind and Bats for E2E scenarios, including resizing and destruction of LVM groups.

---

## Features

- **Automated LVM Initialization:** Creates physical volumes and volume groups on specified block devices.
- **Destruction Mode:** Removes specified LVM volume groups and associated physical volumes when configured.
- **Loop Device Support:** Designed for use with loopback devices for local or CI-based testing.
- **Resizing:** Detects device size increases and grows PVs automatically.
- **JSON Logging:** Emits structured logs for better parsing/debugging.
- **Comprehensive E2E Tests:** Test lifecycle with Bats, Kind, and Helm integration.

---

## Usage Overview

1. **Build lvminit:**
   ```sh
   make build
   # or
   GOOS=linux CGO_ENABLED=0 go build -o lvminit main.go
