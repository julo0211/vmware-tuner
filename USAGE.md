# VMware Tuner - Complete Usage Guide

## Quick Start

```bash
# 1. Build the tool
git clone https://github.com/dtouzeau/vmware-tuner.git
cd vmware-tuner
go build -ldflags="-s -w" -o vmware-tuner

# 2. Install (optional)
sudo ./install.sh

# 3. Preview what will be changed
sudo ./vmware-tuner --dry-run

# 4. Show current configuration
sudo ./vmware-tuner show

# 5. Apply tuning
sudo ./vmware-tuner

# 6. Reboot (if GRUB was modified)
sudo reboot
```

## Command Reference

### Main Commands

| Command | Description |
|---------|-------------|
| `vmware-tuner` | Apply all optimizations (interactive) |
| `vmware-tuner --dry-run` | Preview changes without applying |
| `vmware-tuner show` | Display current system configuration |
| `vmware-tuner verify` | Check if tuning has been applied |
| `vmware-tuner --help` | Show help message |
| `vmware-tuner --version` | Show version |

### Selective Tuning Flags

Skip specific optimization modules:

```bash
sudo vmware-tuner --no-grub      # Skip GRUB (no reboot needed)
sudo vmware-tuner --no-sysctl    # Skip sysctl tuning
sudo vmware-tuner --no-fstab     # Skip filesystem optimization
sudo vmware-tuner --no-io        # Skip I/O scheduler
sudo vmware-tuner --no-network   # Skip network tuning
```

Combine multiple flags:
```bash
sudo vmware-tuner --no-grub --no-network
```

## Usage Examples

### Example 1: First-Time Tuning

```bash
# Step 1: Check current configuration
sudo vmware-tuner show

# Step 2: Preview changes
sudo vmware-tuner --dry-run

# Step 3: Apply all optimizations
sudo vmware-tuner
# Answer "yes" when prompted

# Step 4: Reboot
sudo reboot
```

### Example 2: Tune Without Reboot

If you want to apply optimizations that don't require a reboot:

```bash
sudo vmware-tuner --no-grub
```

This will:
- ✓ Apply sysctl parameters (immediate)
- ✓ Optimize fstab and remount (immediate)
- ✓ Configure I/O scheduler (immediate)
- ✓ Setup network tuning (immediate)
- ✗ Skip GRUB (requires reboot)

### Example 3: Check What's Been Applied

```bash
sudo vmware-tuner verify
```

Output:
```
✓ Sysctl configuration file exists
✓ I/O scheduler udev rules exist
✓ Network tuning service exists
✓ Network tuning service is enabled
✓ All tuning configurations are present
```

### Example 4: Review Changes Before Applying

```bash
sudo vmware-tuner --dry-run 2>&1 | less
```

This shows exactly what will be changed without making any modifications.

### Example 5: Custom Tuning (Network Only)

```bash
sudo vmware-tuner --no-grub --no-sysctl --no-fstab --no-io
```

This applies only network optimizations.

## Understanding the Output

### During Execution

```
✓ Green checkmark  = Success
✗ Red X            = Error
⚠ Yellow warning   = Warning (non-critical)
ℹ Blue info        = Information
▶ Purple arrow     = Section header
```

### Example Output

```
╔══════════════════════════════════════════════════════════╗
║           VMware VM Performance Tuner                    ║
╚══════════════════════════════════════════════════════════╝

✓ Detected VMware virtual machine

▶ Tuning Summary
────────────────────────────────────────────────────────
The following optimizations will be applied:

  1. GRUB boot parameters
  2. Sysctl kernel parameters
  3. Filesystem mount options
  4. I/O scheduler configuration
  5. Network interface optimization

Continue with tuning? (yes/no): yes

✓ Backup directory created: /root/.vmware-tuner-backups/20251113-174500

▶ Optimizing GRUB boot parameters
────────────────────────────────────────────────────────
ℹ Current cmdline: quiet
ℹ New cmdline: quiet elevator=noop tsc=reliable ...
✓ Updated /etc/default/grub
ℹ Running update-grub...
✓ GRUB configuration updated
⚠ REBOOT REQUIRED for boot parameter changes to take effect
```

## Backup and Rollback

### Backup Location

All backups are stored in:
```
/root/.vmware-tuner-backups/<timestamp>/
```

Each backup contains:
- Original configuration files
- README.txt with instructions
- rollback.sh script

### View Backups

```bash
ls -la /root/.vmware-tuner-backups/
```

### Automatic Rollback

```bash
cd /root/.vmware-tuner-backups/20251113-174500/
sudo bash rollback.sh
```

### Manual Rollback

Restore individual files:

```bash
cd /root/.vmware-tuner-backups/20251113-174500/

# Restore GRUB
sudo cp grub /etc/default/grub
sudo update-grub

# Restore fstab
sudo cp fstab /etc/fstab

# Remove sysctl config
sudo rm /etc/sysctl.d/99-vmware-performance.conf
sudo sysctl --system

# Remove I/O scheduler rules
sudo rm /etc/udev/rules.d/60-scheduler.rules
sudo udevadm control --reload-rules

# Remove network service
sudo systemctl stop network-tuning.service
sudo systemctl disable network-tuning.service
sudo rm /etc/systemd/system/network-tuning.service
sudo systemctl daemon-reload

# Reboot
sudo reboot
```

## What Gets Changed

### Files Modified

| File | Changes | Backup? | Reboot? |
|------|---------|---------|---------|
| `/etc/default/grub` | Boot parameters | ✓ | ✓ |
| `/etc/fstab` | Mount options | ✓ | △ |
| `/etc/sysctl.d/99-vmware-performance.conf` | Created | N/A | ✗ |
| `/etc/udev/rules.d/60-scheduler.rules` | Created | N/A | △ |
| `/etc/systemd/system/network-tuning.service` | Created | N/A | ✗ |

△ = May require reboot if live changes fail

### System Changes

**GRUB Boot Parameters:**
```
elevator=noop
transparent_hugepage=madvise
vsyscall=emulate
clocksource=tsc
tsc=reliable
intel_idle.max_cstate=0
processor.max_cstate=1
nmi_watchdog=0
pcie_aspm=off
nvme_core.default_ps_max_latency_us=0
```

**Sysctl Parameters:**
```
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5
vm.vfs_cache_pressure = 50
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_congestion_control = bbr
... and more
```

**Filesystem Mount Options:**
```
noatime,nodiratime,commit=60
```

**I/O Scheduler:**
```
none (or noop on older kernels)
nr_requests = 256
read_ahead_kb = 256
```

**Network:**
```
Ring buffers: RX/TX = 4096
Hardware offload: enabled
Interrupt coalescing: optimized
```

## Troubleshooting

### Issue: "must be run as root"

**Solution:**
```bash
sudo vmware-tuner
```

### Issue: "not a VMware VM" warning

The tool detects if you're running on VMware. If you see this warning but ARE on VMware:
```bash
# Continue anyway when prompted
# Or check detection:
cat /sys/class/dmi/id/product_name
lsmod | grep vmw
```

### Issue: BBR congestion control not available

**Solution:**
```bash
# Check available algorithms
sysctl net.ipv4.tcp_available_congestion_control

# If BBR is missing, it will fall back to current algorithm
# This is not critical
```

### Issue: Filesystem remount failed

The tool will warn you and the changes will take effect on next boot.

**Manual remount:**
```bash
sudo mount -o remount /
```

### Issue: Network service failed to start

Changes will apply on next boot. This is normal if interfaces aren't ready yet.

**Manual start:**
```bash
sudo systemctl start network-tuning.service
sudo systemctl status network-tuning.service
```

## Performance Verification

### Before and After Comparison

**Before tuning:**
```bash
# Capture baseline
sudo vmware-tuner show > /tmp/before.txt
```

**After tuning:**
```bash
# Capture after state
sudo vmware-tuner show > /tmp/after.txt

# Compare
diff /tmp/before.txt /tmp/after.txt
```

### Check Active Settings

```bash
# Sysctl
sysctl vm.swappiness
sysctl net.ipv4.tcp_congestion_control

# I/O Scheduler
cat /sys/block/sda/queue/scheduler

# Mount options
mount | grep " / "

# Network
ethtool -g ens192
ethtool -k ens192 | grep offload
```

### Benchmark Performance

```bash
# Disk I/O
sudo dd if=/dev/zero of=/tmp/test bs=1M count=1000 oflag=direct

# Network throughput (requires iperf3)
iperf3 -c <server> -t 60

# CPU latency
cyclictest -p 80 -t 1 -n -i 10000 -l 1000
```

## Advanced Usage

### Customize Tuning

Edit source files to customize parameters:

1. Edit the desired file:
```bash
git clone https://github.com/dtouzeau/vmware-tuner.git
nano vmware-tuner/sysctl.go
```

2. Rebuild:
```bash
go build -ldflags="-s -w" -o vmware-tuner
```

3. Reinstall:
```bash
sudo ./install.sh
```

### Integration with Configuration Management

**Ansible:**
```yaml
- name: Tune VMware VM
  command: /usr/local/bin/vmware-tuner
  args:
    creates: /etc/sysctl.d/99-vmware-performance.conf
```

**Puppet:**
```puppet
exec { 'vmware-tuner':
  command => '/usr/local/bin/vmware-tuner',
  creates => '/etc/sysctl.d/99-vmware-performance.conf',
}
```

### Automated Deployment

```bash
#!/bin/bash
# Deploy to multiple VMs

for host in vm1 vm2 vm3; do
  echo "Tuning $host..."
  scp vmware-tuner root@$host:/tmp/
  ssh root@$host "/tmp/vmware-tuner"
done
```

## Best Practices

1. **Always use --dry-run first** to preview changes
2. **Test on one VM** before rolling out to production
3. **Keep backups** accessible (note the timestamp)
4. **Schedule reboot** during maintenance window
5. **Verify after reboot** with `vmware-tuner verify`
6. **Monitor performance** for 24-48 hours after tuning
7. **Document changes** in your change management system

## FAQ

**Q: Will this work on physical servers?**
A: No, these optimizations are specifically for VMware VMs.

**Q: Can I revert changes?**
A: Yes, use the rollback.sh script in the backup directory.

**Q: Do I need to reboot?**
A: Only if GRUB parameters were changed. Use `--no-grub` to avoid reboot.

**Q: Is it safe for production?**
A: Yes, but test on dev/staging first. All changes are backed up.

**Q: What if something breaks?**
A: Boot into recovery mode and restore from backup.

**Q: How much performance improvement?**
A: Typically 15-30% for I/O, 10-20% for network. YMMV.

**Q: Can I run it multiple times?**
A: Yes, it's idempotent. Already-optimized settings won't be changed again.
