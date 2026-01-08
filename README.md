# VMware Tuner (Enterprise)

**The Ultimate Swiss Army Knife for VMware Linux VMs.**

`vmware-tuner` is a comprehensive, safe, and interactive tool designed to optimize, audit, maintain, and troubleshoot Linux virtual machines running on VMware ESXi/Workstation.

**🚀 Enterprise Edition (Air-Gap Ready)**

It supports **Debian, Ubuntu, RHEL, CentOS, AlmaLinux, and Rocky Linux**.

## 🚀 Key Features

This Enterprise version adds specific features for secured environments:
*   **🔌 Hybrid Connectivity / Offline Mode**: Automatically detects internet access using HTTP HEAD (no ICMP). Falls back gracefully if offline.
*   **📦 Air-Gap Ready**: Can be compiled with all dependencies (`vendor/`) for isolated servers.
*   **⏪ Native Rollback**: Zero-dependency rollback system using a JSON manifest. No generated scripts.
*   **🔒 Security Hardened**: Robust execution paths, non-interactive apt, and safe inputs.

The tool provides a unified interactive menu with **16 modules**:

### 🛠️ Optimization & Tuning
*   **[1] Optimize this VM**: Applies industry-standard tuning:
    *   **GRUB**: Optimizes I/O scheduler (`noop`/`none`) and memory pages.
    *   **Sysctl**: Tunes `swappiness`, `dirty_ratio`, and network buffers.
    *   **Network**: Enables `tx-checksumming`, `tso`, `gso` for VMXNET3.
    *   **Disk**: Optimizes `fstab` (noatime) and block device settings (Robust `lsblk -J` parsing).
    *   **VMware Tools**: Ensures `open-vm-tools` is installed and running.
    *   **Debloat**: (Optional) Disables unused services (Server Slim mode).

### 🛡️ Safety & Backup
*   **[2] Restore a Backup**: Every change is backed up. You can rollback to any previous state instantly via the Manifest system.
*   **[3] Audit System**: Scans the VM and gives an optimization score (0-100).
*   **[16] Safe System Update**: Checks disk space (>1GB) before running `apt/dnf update`.

### 🔧 Maintenance & Tools
*   **[4] Expand Disk**: Safely expands the root partition and filesystem (`ext4`/`xfs`) after increasing disk size in vSphere.
*   **[5] Fix Time Sync**: Detects NTP conflicts and ensures accurate timekeeping.
*   **[6] Clean System**: Frees space safely (Package cache, Journal vacuum).
*   **[13] Manage Swap**: Creates a 2GB swapfile if missing (prevents OOM crashes).
*   **[8] Schedule Maintenance**: Creates a Cron job for daily time sync and weekly cleaning.

### 🔍 Troubleshooting & Info
*   **[9] System Info**: Dashboard with OS, Kernel, CPU, RAM, and IP stats.
*   **[10] Network Benchmark**: Tests latency and download speed (100MB test file, auto-deleted).
*   **[12] Check Virtual Hardware**: Verifies you are using `vmxnet3` and `pvscsi` drivers.
*   **[14] Scan Logs for Errors**: Scans `dmesg` and `syslog` for critical errors (OOM, I/O, SCSI).
*   **[15] Optimize Docker**: Configures log rotation to prevent disk saturation and offers system prune.

### ⚡ Expert
*   **[7] Secure SSH**: Hardens SSH config (Disable Root/Password) with auto-rollback if syntax check fails.
*   **[11] Seal VM for Template**: Prepares the VM for cloning (Resets Machine ID, SSH Keys, Logs). **Destructive!**

---

## 📥 Installation

### Option 1: Download Binary (Air-Gap)
Download the latest `linux-amd64` release and transfer it to your server.

```bash
chmod +x vmware-tuner-v1.1.0-linux-amd64
sudo mv vmware-tuner-v1.1.0-linux-amd64 /usr/local/bin/vmware-tuner
sudo vmware-tuner
```

See [INSTALL.md](INSTALL.md) for detailed instructions.

### Option 2: Build from Source
Requires Go 1.21+.

```bash
git clone https://github.com/julo0211/vmware-tuner.git
cd vmware-tuner
go mod tidy
go build -o vmware-tuner ./cmd/vmware-tuner
sudo ./vmware-tuner
```

---

## 📖 Usage

Simply run the tool as root:

```bash
sudo ./vmware-tuner
```

You will see the interactive menu:

```text
  [1] Optimize this VM (Tuning)
  [2] Restore a backup (Rollback)
  [3] Audit System (Score)
  ...
  [16] Safe System Update
  [0]  Exit
```

### Non-Interactive Mode (Automation)
You can also use flags for automation scripts (Ansible, etc.):

```bash
# Apply all optimizations automatically
sudo ./vmware-tuner --dry-run=false --install-tools=true

# Show current config
sudo ./vmware-tuner show

# Verify optimizations
sudo ./vmware-tuner verify
```

---

## ⚠️ Safety First

This tool is designed for **Production**.
1.  **Backups**: Configuration files (`grub`, `sysctl.conf`, `sshd_config`) are backed up before modification.
2.  **Checks**: Destructive actions (Disk Expand, Seal VM) require explicit confirmation.
3.  **Validation**: SSH config is verified (`sshd -t`) before restart.

## License

MIT License
