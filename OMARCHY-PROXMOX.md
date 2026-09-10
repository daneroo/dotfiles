# Omarchy on Proxmox

Living runbook for installing Omarchy on `hilbert` and using it remotely from
`galois` with Sunshine/Moonlight.

Last updated: 2026-09-10

## Daily quick reminders

Moonlight from `galois` to `omoxy` is working at 1080p/60 with Super-key input.

- To switch back to Mac desktops without ending the stream: press
  **Control–Option–Shift–Z** to release Moonlight input, then use
  **Control–Left/Right** to change macOS Spaces. Click the Moonlight window to
  recapture input.
- With the swapped PC keyboard, use the physical key that macOS treats as
  **Option** (currently the Windows-labelled key).
- To quit the stream: **Control–Option–Shift–Q**.
- `Control–Option–Shift–D` (minimize) is unreliable in this Borderless-windowed
  setup; it briefly shrinks and immediately returns.
- If `omoxy` reboots (including after a host restart or power loss), open its
  Proxmox noVNC console and enter the LUKS passphrase. Until then, Linux has
  not started, so SSH and Moonlight cannot connect.

## Top TODO: unattended LUKS unlock

- [ ] Make `omoxy` capable of rebooting without someone entering the LUKS
      passphrase in the Proxmox console.
- [ ] Evaluate binding the encrypted volume to a Proxmox virtual TPM while
      retaining a tested recovery key.
- [ ] If TPM-backed unlock is unsuitable on Proxmox VE 7.4, compare other safe
      unattended-unlock approaches with reinstalling the disposable VM without
      disk encryption.
- [ ] Investigate the recurring post-unlock message
      `Unable to resume from device /dev/mapper/root`. It predates GPU
      passthrough, does not prevent a normal boot, and may be relevant when
      designing LUKS unlock and hibernation behavior.
- [ ] Do not remove encryption or change LUKS key slots until VM 103 has a
      verified backup and recovery procedure.

### Options (keep LUKS)

The preferred path is to add a Proxmox virtual TPM 2.0 to VM 103 and enroll a
TPM2 token in the existing LUKS2 container. This adds an unlock method; it does
not replace the current passphrase or decrypt the disk. Keep the current
passphrase and add a separately stored LUKS recovery key. If TPM unlock cannot
be used, systemd should fall back to a passphrase/recovery key at the console.

The important Proxmox caveat is that a vTPM is an emulated device. Its state is
stored as a normal VM volume, so it improves convenience and protects against
someone possessing only the encrypted guest disk, but it is not protection
against a compromised Proxmox host. A snapshot/backup is only complete for
this boot arrangement if it includes the VM disk, `efidisk0`, and the TPM state
disk. Restoring the encrypted disk without its matching TPM state should still
be recoverable with the manual LUKS passphrase/recovery key; restoring all
three preserves automatic TPM unlock. We must test both cases before relying
on unattended boot.

Other LUKS-preserving choices:

- Remote initramfs unlock (for example, SSH/dropbear): encryption remains
  strong, but a person still types the passphrase remotely. Tailscale cannot
  be assumed before the root filesystem is unlocked.
- FIDO2 security key: strong additional factor, but it requires the key/touch
  at boot and is not unattended. It also introduces USB availability and
  passthrough considerations.
- Tang/Clevis network-bound unlock: can unlock automatically when a trusted
  on-LAN Tang server is reachable, but adds a boot-time network/service
  dependency. Tailscale is generally too late in the boot sequence for this.
- A host-injected keyfile or QEMU secret: fully unattended, but the Proxmox
  host can obtain the key. This is the weakest separation and is not the
  recommended default.

Staged plan: take and verify a fresh backup; record the current LUKS header and
confirm the existing passphrase; add `tpmstate0` version 2.0; verify the guest
TPM device; enroll TPM2 plus a recovery key without removing the old slot;
test clean boot, VM stop/start, host reboot, manual recovery, and backup
restore with and without TPM state; only then consider VM autostart/headless
operation. The recurring `/dev/mapper/root` resume warning is separate from
LUKS unlock and remains an investigation item.

Reference documentation:

- <https://www.freedesktop.org/software/systemd/man/250/crypttab.html>
- <https://man7.org/linux/man-pages/man1/systemd-cryptenroll.1.html>
- <https://github.com/proxmox/pve-docs/blob/master/qm.adoc>

## Goal

- Run Omarchy 4.0.3 as VM 103 (`omoxy`) on `hilbert`.
- Initially install and recover through the Proxmox VirtIO/noVNC console.
- Ultimately connect from the Mac mini M2 Pro (`galois`) with Moonlight.
- Make GPU and physical USB passthrough independently switchable.
- Prefer GPU-only passthrough for Sunshine; Moonlight supplies remote input.

## Current status

- [x] Upload `omarchy-4.0.3.iso` to Proxmox ISO storage.
- [x] Inspect the host, storage, bridge, and passthrough hardware.
- [x] Create VM 103 and name it `omoxy`.
- [x] Boot the ISO with VirtIO display and no PCI/USB passthrough.
- [x] Complete the interactive Omarchy installation in the Proxmox console
      (1 minute 53 seconds).
- [x] Reboot into the installed system, unlock LUKS, and reach the Omarchy
      desktop through the Proxmox console.
- [x] Complete the initial Omarchy system update; `checkupdates` reports no
      pending packages.
- [x] Establish LAN access and SSH-key login to the guest.
- [ ] Deferred: reserve `192.168.2.69` in the Bell router and create
      `omoxy.imetrical.com`. This is not required for the current experiment.
- [x] Install and connect Tailscale.
- [x] Create and validate `omoxy.ts.imetrical.com` for tailnet access.
- [x] Install and sign in to 1Password using QR-code enrollment.
- [x] Install and verify `qemu-guest-agent` and `libva-utils`.
- [x] Install and start Sunshine while the VirtIO console remains available.
- [x] Create the Sunshine administrator login through an SSH localhost tunnel.
- [x] Pair Moonlight with Sunshine over Tailscale.
- [x] Test the VirtIO/software-encoder baseline: the stream connects, but the
      image is completely garbled even in Moonlight's low-resolution mode.
- [x] Retry with Moonlight forced to software decoding and H.264 at 1280x720,
      30 FPS, and 5 Mbps; the stream remained completely garbled.
- [ ] After the baseline test, evaluate upgrading Sunshine from Omarchy's
      packaged version to the current upstream stable Arch package and retest.
- [x] Test Moonlight video and remote input successfully from `galois`.
- [x] Attach the RX 570 as a secondary PCIe device while retaining VirtIO.
- [x] Unlock LUKS and verify the AMD driver, render node, and VA-API encoder.
- [x] Configure Sunshine to encode with the Radeon through VA-API.
- [x] Retest Moonlight against the hybrid VirtIO-capture/Radeon-encode setup:
      it showed only a blank screen, then capture failed and Moonlight exited
      with error `-1`; no usable desktop frame was displayed.
- [x] Make the Radeon Hyprland's primary renderer while retaining VirtIO as a
      secondary recovery output; Moonlight then displays a usable desktop.
- [x] Enable Moonlight's system-key capture so Omarchy Super shortcuts work.
- [x] Validate 1920x1080 at 60 FPS and make the guest resolution persistent.
- [ ] Later, evaluate GPU-only passthrough after the hybrid test works.
- [ ] Configure a reliable headless Hyprland output.
- [ ] Add optional Titan Ridge USB-controller passthrough.
- [ ] Remove stale/conflicting passthrough entries from VM 100 when VM 103 is
      proven stable.

## Host inventory

| Item           | Value                                            |
| -------------- | ------------------------------------------------ |
| Proxmox node   | `hilbert`                                        |
| Proxmox VE     | 7.4-20                                           |
| QEMU           | 7.2.10                                           |
| CPU            | Intel Core i9-9900K, 8 cores / 16 threads        |
| RAM            | 46 GiB                                           |
| VM storage     | `local-zfs`                                      |
| Network bridge | `vmbr0`                                          |
| ISO storage ID | `pve-storage_backups-isos`                       |
| ISO volume     | `pve-storage_backups-isos:iso/omarchy-4.0.3.iso` |

Proxmox VE 7.4 is outdated, but its QEMU version supports the OVMF, q35,
VirtIO, and PCI-passthrough features required here. Upgrade the host separately
after the Omarchy experiment; it is not required for the initial installation.

## VM 103

The VM was created from Omarchy's documented Proxmox shape, adjusted for
`hilbert`'s storage names:

| Setting              | Value                                                    |
| -------------------- | -------------------------------------------------------- |
| VM ID / Proxmox name | `103` / `omoxy`                                          |
| Guest hostname       | `omoxy` (set during installation)                        |
| Firmware / machine   | OVMF / q35                                               |
| Secure Boot keys     | Disabled                                                 |
| CPU                  | Host type, 4 cores                                       |
| Memory               | 8192 MiB                                                 |
| Disk                 | 40 GiB thin ZFS, VirtIO SCSI single, discard + IO thread |
| Network              | VirtIO on `vmbr0`, Proxmox firewall flag enabled         |
| Install display      | VirtIO GPU through Proxmox noVNC                         |
| Autostart            | Disabled                                                 |
| Guest agent          | Enabled in Proxmox; must also be installed in Omarchy    |

Current profile: **hybrid encoding test** (plain `vga: virtio` retained for
noVNC/LUKS recovery, with the RX 570 attached as a secondary PCIe device).

Useful checks:

```bash
ssh root@hilbert qm status 103
ssh root@hilbert qm config 103
```

The installation completed in 1 minute 53 seconds. Select **Reboot Now** in the
Proxmox console. The VM boot order is `scsi0;ide2`, so the installed system disk
is tried before the still-attached installer ISO.

First boot succeeded: LUKS unlocked and the Omarchy desktop appeared correctly
through the VirtIO/noVNC display. Every boot since installation has displayed
`Unable to resume from device /dev/mapper/root` after LUKS unlock, but startup
continues normally. This predates PCI passthrough. The kernel command line
contains `resume=/dev/mapper/root resume_offset=1614914`, and the active swap
devices are an encrypted-root Btrfs swapfile plus zram. Leave this unchanged
for now and investigate it together with unattended LUKS unlock.

Proxmox noVNC from macOS does not reliably forward the Super/Command key, which
blocks many default Hyprland shortcuts. Use noVNC for LUKS unlock and visual
recovery, but run administrative commands over SSH. Moonlight should carry the
remote keyboard more naturally once Sunshine is working.

Installer account choices confirmed:

| Field    | Value             |
| -------- | ----------------- |
| Username | `daniel`          |
| Hostname | `omoxy`           |
| Timezone | `America/Toronto` |
| Keyboard | `us`              |

Passwords and account credentials are deliberately not recorded here.

## Network identity and DNS

| Scope                   | Name/address             |
| ----------------------- | ------------------------ |
| VM NIC MAC              | `76:43:FF:F0:9B:AF`      |
| Current LAN lease       | `192.168.2.69`           |
| Intended LAN name       | `omoxy.imetrical.com`    |
| Tailnet MagicDNS suffix | `tail62209.ts.net`       |
| MagicDNS name           | `omoxy.tail62209.ts.net` |
| Tailscale IPv4          | `100.124.82.116`         |
| Tailnet alias           | `omoxy.ts.imetrical.com` |

The LAN gateway and DHCP server is the Bell Home Hub/Giga Hub at
`192.168.2.1`. Reserve the existing address by opening **My Devices**, locating
the Ethernet device with MAC `76:43:FF:F0:9B:AF`, editing it, and changing its
IP type from **Dynamic** to **Reserved** while retaining `192.168.2.69`.

This reservation could not be completed in the Bell Giga Hub interface and is
deferred. Do not make it a prerequisite for Sunshine: use Tailscale for stable
management access after the initial LAN bootstrap.

`imetrical.com` uses Hover authoritative DNS. Existing host records establish
the convention: `<host>.imetrical.com` is an A record containing its
`192.168.2.x` LAN address, and `<host>.ts.imetrical.com` is an A record
containing its `100.x` Tailscale address. At discovery time,
`omoxy.imetrical.com` had no explicit record and fell through to Hover's
existing `216.40.34.41` wildcard/forwarding destination.

Create the LAN record after reserving the DHCP lease:

```text
omoxy.imetrical.com. A 192.168.2.69
```

The matching Hover record was created after Tailscale enrollment:

```text
omoxy.ts.imetrical.com. A 100.124.82.116
```

Validation from `galois` confirmed Hover's authoritative answer, SSH through the
alias, and a direct Tailscale path via `192.168.2.69:41641` at approximately
1 ms rather than a DERP relay. Tailscale A records should be audited after a
machine is re-enrolled because its address can change.

## Backups

VM 103 is included in the enabled daily Proxmox backup job
`backup-11e05820-469e`:

| Setting      | Value                                             |
| ------------ | ------------------------------------------------- |
| Schedule     | Daily at 21:00                                    |
| Storage      | `pve-storage_backups-isos`                        |
| Mode         | Snapshot                                          |
| Compression  | Zstandard                                         |
| Notification | Always                                            |
| Included VMs | `101, 102, 103, 120, 121`                         |
| Retention    | 10 last, 10 daily, 4 weekly, 12 monthly, 5 yearly |

Snapshot mode provides a live backup with low downtime. `qemu-guest-agent` is
installed and running inside `omoxy`, so Proxmox can freeze and thaw its
filesystems to improve backup consistency.

On current Arch the guest-agent service is a static, VirtIO-device-bound unit;
it is not enabled manually. Its verified state is `static` and `active`.
Proxmox successfully queried its ping, hostname, LAN interface, and Tailscale
interface without rebooting the VM.

## Passthrough inventory

### AMD GPU

The card is described generically by PCI as RX 470/480/570/580/590, while its
subsystem identifies it specifically as a Sapphire RX 570 Pulse 4 GB.

| Function   | PCI address | PCI ID      | Current host driver |
| ---------- | ----------- | ----------- | ------------------- |
| GPU        | `01:00.0`   | `1002:67df` | `vfio-pci`          |
| HDMI audio | `01:00.1`   | `1002:aaf0` | `vfio-pci`          |

Both functions are isolated together in IOMMU group 15. Passing
`0000:01:00` assigns both functions to the guest.

### Physical USB

The Titan Ridge USB controller is currently:

| PCI address | PCI ID      | IOMMU group | Current host driver |
| ----------- | ----------- | ----------- | ------------------- |
| `3c:00.0`   | `8086:15ec` | 24          | `xhci_hcd`          |

The old VM 100 note says `3b:00.0`; that address is stale after PCI
re-enumeration. The controller currently owns USB buses 3 and 4, and no devices
were attached when inspected. Passing the controller allows hot-plugging its
keyboard, mouse, and audio devices without individual USB mappings.

## Hardware profiles

All profile changes require VM 103 to be shut down first.

| Profile          | Proxmox display | AMD GPU | Titan Ridge USB | Intended use                         |
| ---------------- | --------------- | ------- | --------------- | ------------------------------------ |
| Install/fallback | VirtIO/noVNC    | No      | No              | Installation and recovery            |
| Hybrid encode    | VirtIO/noVNC    | Yes     | No              | Radeon encoding without rewiring     |
| Sunshine         | None            | Yes     | No              | Normal remote use from Moonlight     |
| USB test         | VirtIO/noVNC    | No      | Yes             | Test physical input independently    |
| Full physical    | None            | Yes     | Yes             | Monitor and peripherals at `hilbert` |

The hybrid encoding profile was applied with VM 103 stopped:

```bash
ssh root@hilbert qm set 103 -hostpci0 0000:01:00,pcie=1
```

The shortened `0000:01:00` address passes both GPU function `01:00.0` and HDMI
audio function `01:00.1`. The `x-vga` option is deliberately absent, leaving
VirtIO as the primary/recovery display. To roll this change back, first shut
down VM 103 and run:

```bash
ssh root@hilbert qm set 103 -delete hostpci0
```

Do not start VM 100 while VM 103 has either shared PCI device attached.

## Sunshine/Moonlight plan

Terminology: **Sunshine** is the streaming host on `omoxy`; **Moonlight** is the
client on `galois`. Moonlight 6.1.0 is already installed on `galois`.

Validation is deliberately split into two stages:

| Stage       | VM display         | What it proves                                                          |
| ----------- | ------------------ | ----------------------------------------------------------------------- |
| Baseline    | VirtIO/noVNC       | Pairing, capture, firewall, LAN/Tailscale path, audio, and remote input |
| Performance | RX 570 passthrough | AMD VA-API hardware encoding and usable streaming latency/quality       |

The VirtIO baseline is not expected to prove hardware-encoding performance.
Pairing and the Tailscale transport succeeded, but the first Desktop stream was
completely garbled. Moonlight's low-resolution mode did not improve it. The VM
is configured with plain `vga: virtio`, not VirtIO-GL. Sunshine selected the
Wayland `wlgrab` capture path for QEMU's `Virtual-1` output at 1280x800 and used
`libx264` software H.264. Its log also contained:

```text
KMS: DRM_IOCTL_MODE_CREATE_DUMB failed: Permission denied
Error: Failed to create GBM buffer
MESA-EGL: warning: egl: failed to create dri2 screen
```

This makes the VirtIO/GBM capture-and-encode path the leading suspect; reducing
the streamed resolution does not repair a corrupt source frame or buffer. The
client-side isolation test forced Moonlight's video decoder to **Software**,
forced **H.264**, left HDR and YUV 4:4:4 disabled, and used 1280x720 at 30 FPS
and 5 Mbps. It remained completely garbled. The VirtIO baseline is therefore
closed as failed; do not spend more time tuning bitrate or resolution.

To end a Moonlight session from macOS, press
**Control-Option-Shift-Q**. Use **Command-Q** if the special stream shortcut is
not received; **Option-Command-Escape** is the force-quit fallback.

Moonlight 6.1.0 provides shortcuts intended to return temporarily to macOS
without ending the host session:

- **Control-Option-Shift-D** is intended to minimize the Moonlight stream
  window, but in this macOS Borderless-windowed setup it shrinks only
  momentarily and immediately restores. Do not rely on it here.
- **Control-Option-Shift-Z** releases/toggles mouse and keyboard capture. After
  release, use the normal macOS desktop shortcut and click Moonlight to capture
  input again.

These are logical macOS modifier names. Because `galois` swaps Command and
Option for its standard PC keyboard, the physical key labelled **Windows** may
be needed where the shortcut says **Option**.

For Omarchy's Super shortcuts, Moonlight is configured with **Capture system
keyboard shortcuts: always**. The earlier **off** setting caused macOS to keep
Command combinations such as the Raycast shortcut instead of forwarding the
GUI/Meta key as Linux Super.

`galois` is a Mac mini and therefore has no built-in native display resolution.
Its current macOS desktop workspace is 2048x1152. The guest's VirtIO
`Virtual-1` advertises 1920x1080@60 and 2048x1152@60 among its available modes.
For the first quality increase, `Virtual-1` was tested dynamically at
1920x1080@60 with scale 1:

```bash
hyprctl eval 'hl.monitor({ output = "Virtual-1", mode = "1920x1080@60", position = "0x0", scale = 1 })'
```

Moonlight was then validated successfully at 1080p and 60 FPS. The equivalent
rule is now persistent in `~/.config/hypr/monitors.lua`:

```lua
hl.monitor({ output = "Virtual-1", mode = "1920x1080@60", position = "0x0", scale = 1 })
```

Later, test 2048x1152 as a custom client resolution if desired.

For the first GPU experiment, `vga: virtio` and noVNC remain available for
LUKS/recovery, while the RX 570 is attached as a secondary PCI device for
VA-API encoding. This does not require a physical monitor, KVM rewiring, or USB
passthrough. After LUKS unlock, confirm H.264 and then HEVC encoding plus
Moonlight performance statistics before increasing resolution. The Polaris GPU
does not support AV1 encoding.

The guest loaded `amdgpu` for `01:00.0` and `snd_hda_intel` for `01:00.1`.
VirtIO remains `/dev/dri/renderD128`; the Radeon is `/dev/dri/renderD129`.
`vainfo` on the Radeon confirmed H.264 High and HEVC Main encoding. Sunshine's
non-secret configuration at `~/.config/sunshine/sunshine.conf` is:

```text
encoder = vaapi
adapter_name = /dev/dri/renderD129
```

After restarting its user service, Sunshine successfully created
`h264_vaapi` and `hevc_vaapi` encoders while capturing the VirtIO `Virtual-1`
desktop through Wayland `wlgrab`. Failures while probing AV1 and HEVC Main10
are expected capability tests for this Polaris GPU; Sunshine reports H.264 and
HEVC Main as available afterward.

The first hybrid stream displayed only a blank screen and then Moonlight
terminated with error `-1`; it never displayed a usable desktop frame.
Sunshine repeatedly logged `[wayland] Failed to create buffer from params`.
This is different from the earlier garbled software-encoded output: the Radeon
encoder starts, but Wayland buffer creation/import fails before a valid desktop
frame reaches Moonlight. Hyprland had opened both GPUs but selected VirtIO's
`renderD128` first, making the cross-GPU import of a VirtIO-produced Wayland
buffer into the Radeon encoder the leading cause.

Omarchy launches Hyprland as a UWSM-managed
`wayland-wm@hyprland.desktop.service`. The next test makes the Radeon the
primary renderer and retains VirtIO second using stable PCI paths in
`~/.config/uwsm/env-hyprland`:

```bash
export AQ_NO_KMS_REQUIREMENT=1
export AQ_DRM_DEVICES="/dev/dri/by-path/pci-0000:01:00.0-card:/dev/dri/by-path/pci-0000:00:01.0-card"
```

Important boot constraint: Sunshine starts only after Linux and Hyprland are
running, so Moonlight cannot enter the pre-boot LUKS passphrase. Before making
GPU-only/headless mode the default, choose one of these recovery paths:

1. Configure and test unattended LUKS unlock.
2. Keep a virtual display available for Proxmox-console unlock while assigning
   the Radeon as a secondary/render GPU.
3. Use the physical KVM plus GPU and USB-controller passthrough to unlock LUKS.

4. Finish Omarchy installation using the VirtIO/noVNC console.
5. Confirm the guest hostname is `omoxy`, then install and enable SSH plus
   `qemu-guest-agent` in the guest.
6. Install and connect Tailscale, then install and sign in to 1Password. Do
   this while the Proxmox recovery console is still available.
7. Install Sunshine using the native Arch package:

   ```bash
   sudo pacman -S sunshine
   ```

8. Run Sunshine inside the Hyprland user session so it inherits the Wayland
   environment. Pair Moonlight before removing the recovery display.
9. Shut down the VM and select the GPU-only profile.
10. Verify the AMD render node and VA-API encoder in the guest:

    ```bash
    ls /dev/dri/renderD*
    vainfo --display drm --device /dev/dri/renderD128
    ```

11. Confirm Sunshine is using AMD hardware encoding rather than software
    encoding.
12. If the Radeon has no active physical monitor, use Hyprland's headless output
    support. A simple initial test is:

    ```bash
    hyprctl output create headless sunshine
    ```

    Make the selected resolution and refresh rate persistent only after the
    dynamic test works. An HDMI dummy plug remains a useful fallback.

Moonlight transports keyboard, mouse, controller, video, and audio over the
network, so physical USB passthrough is unnecessary for normal remote use.

### Omarchy service installers

Run these from a terminal inside the Omarchy desktop so sudo and graphical
authentication prompts are visible. Install Tailscale before Sunshine: the
Sunshine installer adds interface-specific firewall rules when `tailscale0`
already exists.

```bash
omarchy install service tailscale
omarchy install service 1password
omarchy pkg add qemu-guest-agent libva-utils
sudo systemctl enable --now qemu-guest-agent
omarchy install service sunshine
```

Omarchy's Sunshine installer enables the user service, opens the required UFW
ports for private LANs and Tailscale, creates a Sunshine Admin web app, and adds
Sunshine to Hyprland autostart.

In Omarchy 4.0.3, the Sunshine command and installer are present but Sunshine
is not exposed in the default **Install -> Service** menu. Invoke
`omarchy install service sunshine` from an Omarchy terminal. This is the shipped
first-party installer, not a manual package-install workaround.

The Omarchy installer currently calls the obsolete `sunshine.service` unit
name, while Sunshine 2026.516.143833-4.1 ships
`app-dev.lizardbyte.app.Sunshine.service`. This caused the installer to stop
after package installation. Enabling and starting the canonical unit resolved
the failure:

```bash
systemctl --user enable --now app-dev.lizardbyte.app.Sunshine.service
```

Verified baseline state:

- Canonical Sunshine user service is enabled and active.
- Wayland capture finds `Virtual-1` at 1280x800.
- VirtIO VA-API initialization fails as expected; Sunshine falls back to
  `libx264` software H.264.
- Sunshine TCP ports are reachable from `galois` through
  `omoxy.ts.imetrical.com`, but blocked on the raw LAN address by UFW.
- Configuration UI listens at `https://localhost:47990` in the guest and is
  also reachable over Tailscale at `https://omoxy.ts.imetrical.com:47990`.

Sunshine Admin reports upstream stable `v2026.906.222525`, while the current
Omarchy repository still provides `2026.516.143833-4.1` and `checkupdates`
reports no upgrade. Keep Omarchy's package through the first Moonlight baseline
so package changes do not become another test variable. After the baseline,
consider the official upstream Arch package, verify its published checksum,
and repeat the test before enabling GPU passthrough.

Opening the Web UI directly through the custom Tailscale hostname triggers
Sunshine's CSRF protection because that origin is not trusted by default. For
initial setup and occasional administration, keep the default allowlist and use
an SSH localhost tunnel from `galois`:

```bash
ssh -N -L 47990:127.0.0.1:47990 omoxy.ts.imetrical.com
```

While that command is running, open `https://localhost:47990` on `galois`.
Close the tunnel afterward with Ctrl+C. This avoids permanently expanding
`csrf_allowed_origins`; normal Moonlight streaming does not require the admin
Web UI origin.

Verified 1Password installation:

- Desktop app `1password` 8.12.34-36 is installed and running under Wayland.
- CLI `op` 2.39.0-1 is installed.
- The managed Chromium extension configuration is present.
- Account enrollment was completed through Omarchy's QR-code flow; no account
  secrets are recorded here.

## SSH bootstrap

Enable OpenSSH from Omarchy or run this in the guest:

```bash
sudo pacman -S --needed openssh
sudo systemctl enable --now sshd
hostname -I
```

Then copy `galois`'s normal Ed25519 public key, using the guest IP shown by the
last command:

```bash
ssh-copy-id -i ~/.ssh/id_ed25519.pub daniel@<omoxy-ip>
ssh daniel@<omoxy-ip>
```

`ssh-copy-id` creates `~/.ssh` and `authorized_keys` with OpenSSH's required
permissions. Do not copy a private key into the VM.

Verified state:

- `authorized_keys` contains only `galois`'s Ed25519 key.
- The local and remote fingerprints match:
  `SHA256:EF6L9Blun8pldDd9N1i51zXUT2uJV4x1r89M2RhUe2s`.
- `~/.ssh` is mode `0700`; `authorized_keys` is mode `0600`; both are owned by
  `daniel:daniel`.

## References

- Omarchy unattended installs and Proxmox example:
  <https://github.com/basecamp/omarchy/blob/quattro/manual/51-unattended-installs.md>
- Sunshine getting started:
  <https://github.com/LizardByte/Sunshine/blob/master/docs/getting_started.md>
- Sunshine configuration:
  <https://github.com/LizardByte/Sunshine/blob/master/docs/configuration.md>
- Hyprland virtual GPU and headless display:
  <https://github.com/hyprwm/hyprland-wiki/blob/main/content/configuring/extra/virtual-gpu.md>
