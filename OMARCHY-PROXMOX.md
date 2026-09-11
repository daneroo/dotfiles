# Omarchy on Proxmox

Living runbook for installing Omarchy on `hilbert` and using it remotely from
`galois` with Sunshine/Moonlight.

Last updated: 2026-09-11

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
- `omoxy` now unlocks LUKS automatically through its restored/tested vTPM.
  Keep noVNC plus the original passphrase and 1Password recovery key as the
  manual fallback.

## Remaining work (authoritative)

- [x] Implement automatic LUKS unlock without removing encryption.
- [x] Prove VM-level backup/restore with and without the matching TPM state.
- [x] Prove RX 580 HDMI output plus all four Greathtek socket mappings and
      guest-side USB hotplug/disconnect through the physical KVM.
- [x] Prove the physical SF-558 microphone end-to-end: Voxtype successfully
      converted live speech to text inside Omarchy.
- [x] Confirm physical keyboard/mouse control in the Omarchy desktop.
- [ ] Complete the active Display and mirror setup checklist below, including
      boot visibility, Sunshine capture selection, and keyboard mapping.
- [ ] Prove Moonlight still works after the final four-path USB configuration.
- [ ] Investigate the recurring nonfatal boot message `Unable to resume from
      device /dev/mapper/root`.
- [ ] Schedule a Proxmox host reboot and prove unattended TPM unlock afterward;
      this affects the other guests and must not be done implicitly.
- [ ] After the host-reboot proof, decide whether to enable VM 103 autostart.
- [ ] At the end, consolidate this living log into a concise end-state runbook.

Deferred/outside this experiment: the UCG Fibre LAN/DNS migration, upgrading
Hilbert sequentially from Proxmox VE 7 to 8 to 9, a clean-install vTPM test,
and alternative GPU-only/headless/Titan-Ridge profiles.

## Display and mirror setup — active checklist

Goal: one Omarchy desktop presented consistently through the physical RX 580,
VirtIO/noVNC, and (once verified) Moonlight.

### Resolution and Hyprland

- [x] Identify the two guest outputs: RX 580 `HDMI-A-2` and VirtIO `Virtual-1`.
- [x] Persist both at `1920x1080@60`, with `Virtual-1` mirroring `HDMI-A-2`.
- [x] Reboot VM 103 and verify the mirror configuration is applied at startup.
- [ ] After the shared-display behavior is settled, evaluate native
      `2560x1440` with scaling or the common `2048x1152` mode.

### Proxmox display and passthrough

- [x] Retain `vga: virtio` so the Proxmox noVNC console remains available.
- [x] Retain RX 580 PCI passthrough for physical HDMI and Radeon encoding.
- [x] Retain the four physical Greathtek socket-path mappings for KVM input.
- [ ] Verify Moonlight's actual Sunshine capture output after mirroring.

### Boot sequence and stated preference

- [x] Confirm the OVMF/Proxmox boot-options screen appears on both outputs.
- [x] Confirm Limine/early Linux output currently appears through VirtIO/noVNC;
      the RX 580 becomes visible when the graphical desktop starts.
- [ ] Determine whether stock Limine can show its graphical menu on both
      independent GOP framebuffers, or whether a custom boot-stage solution is
      required.
- [ ] Determine whether early Linux boot/status output can also be presented on
      both outputs; Hyprland mirroring is too late for this stage.

Preferred end state: Limine and useful boot visibility on both outputs, followed
by one mirrored desktop. Fallback end state: preserve noVNC as the authoritative
boot/recovery display if dual-output Limine is not achievable.

## LUKS implementation checklist

- [x] Take a fresh pre-LUKS Proxmox backup and verify it is readable
      (`vzdump-qemu-103-2026_09_10-23_50_30.vma.zst`, note
      `omoxy - pre-luks baseline`, Proxmox task `TASK OK`). VM 103 restarted
      successfully and was left stopped.
- [x] Establish a pre-LUKS restore baseline (VM 203 restored from the
      2026-09-10 backup):
      - create `~/HEARTBEAT.md` containing `I was created/updated at
        <ISO-8601 timestamp>`;
      - take a snapshot/backup of VM 103;
      - shut down VM 103 and restore that point-in-time backup as VM 203;
      - boot VM 203 and assert the expected disk state, the exact
        `HEARTBEAT.md` content/timestamp, hostname/guest identity, SSH access,
        qemu-guest-agent operation, and successful Moonlight streaming;
      - remove or power off VM 203 after the proof, without modifying VM 100.
- [x] Confirm the existing LUKS passphrase and record the current slot/header
      state (without recording secrets): LUKS2 on `/dev/sda2`, UUID
      `8891ddc0-b972-4ddf-8beb-d59b352caf3f`, one active passphrase slot
      (`slot 0`), and no tokens.
- [x] Add a Proxmox vTPM 2.0 state disk to VM 103.
- [x] Verify the TPM device inside Omarchy (`/dev/tpm0` and `/dev/tpmrm0`
      present; systemd 261 has TPM2 support).
- [x] Enroll and test a separately stored recovery key in slot 1; retain the
      existing passphrase in slot 0. The previously exposed key was removed.
- [x] Enroll a TPM2 LUKS2 token while retaining slots 0 and 1. The guest
      reports slots 0 `password`, 1 `recovery`, and 2 `tpm2`.
- [x] Integrate TPM unlock into the UKI: switch Omarchy's active
      `/etc/mkinitcpio.conf.d/omarchy_hooks.conf` override from legacy
      `udev`/`encrypt` to `systemd`/`sd-encrypt`, and add an
      `/etc/crypttab.initramfs` entry for mapper `root` with
      `tpm2-device=auto`. Retain the known-working Limine `cryptdevice=`
      command line while testing.
- [x] Rebuild through Omarchy's supported `limine-mkinitcpio` wrapper and
      verify the BLAKE2b hash embedded in the active `path:` entry exactly
      matches `/boot/EFI/Linux/omarchy_linux.efi`. Raw `mkinitcpio` rebuilds
      the UKI but bypasses this Limine history/hash synchronization.
- [x] Test a clean reboot with no keyboard input: TPM unlocked LUKS,
      `/dev/mapper/root[/@]` mounted as Btrfs, SSH and qemu-guest-agent became
      active, graphical boot completed in 20.228 seconds, and Moonlight still
      streamed the desktop successfully.
- [x] Test a clean cold VM shutdown/start with unattended TPM unlock.
- Host-level reboot proof remains intentionally deferred and is tracked in the
  authoritative checklist above.
- [x] Restore the post-LUKS backup as VM 203 with its matching EFI and TPM
      state disks: automatic unlock, exact HEARTBEAT timestamp/hash, encrypted
      root, SSH, qemu-guest-agent, Tailscale, and Moonlight all passed.
- [x] Detach restored VM 203's TPM state (retained temporarily as `unused0`),
      boot the encrypted disk, and verify the original passphrase still unlocks
      it; exact HEARTBEAT state, encrypted root, SSH, and qemu-guest-agent all
      passed. The separately stored slot-1 recovery key was already validated
      directly with `cryptsetup --test-passphrase`.
- [x] Complete both VM 103 → VM 203 recovery paths: automatic unlock with the
      matching restored TPM state and manual unlock without TPM state.
- VM autostart remains intentionally deferred and is tracked above.
- Optional future experiment: install a disposable Omarchy VM with a vTPM
  already attached and observe whether the installer configures TPM-backed
  LUKS automatically. Consider running Codex inside that guest so its bundled
  Omarchy skills are available.

Implemented state: VM 103 has a Proxmox vTPM 2.0 state disk. LUKS slot 0 is
the original passphrase, slot 1 is the tested recovery key stored in 1Password,
and slot 2 is TPM2. Omarchy's active `omarchy_hooks.conf` uses `systemd` and
`sd-encrypt`; `/etc/crypttab.initramfs` maps LUKS UUID
`8891ddc0-b972-4ddf-8beb-d59b352caf3f` to `root` with
`tpm2-device=auto`. Rebuild this UKI with `sudo limine-mkinitcpio`, not raw
`mkinitcpio`, so Limine refreshes its BLAKE2b path hash and enrolled config.

The verified post-LUKS backup is
`vzdump-qemu-103-2026_09_11-00_54_15.vma.zst` (9,404,647,155 bytes). Proxmox
included `scsi0`, `efidisk0`, and `tpmstate0`. Restoring it as temporary VM 203
proved automatic unlock with the restored TPM state and manual passphrase
fallback after detaching that TPM state. VM 203 and its temporary volumes were
then deleted; the backup archive remains available.

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
- Deferred outside this experiment: create `omoxy.imetrical.com` after the UCG
  Fibre gateway migration.
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
- [x] Keep Omarchy's packaged Sunshine; the current stream is working.
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
- [x] Persist a single 1920x1080@60 Hyprland desktop on both outputs by
      mirroring VirtIO `Virtual-1` from RX 580 `HDMI-A-2`; reboot verification
      confirmed the configuration is applied at startup.
- [x] Restore VM 103's pre-LUKS backup as VM 203 and verify encrypted disk,
      heartbeat, hostname, SSH, qemu-guest-agent, Tailscale, and Moonlight.
- [x] Confirm the RX 580 HDMI output works through the physical KVM.
- [x] Reboot VM 103 and confirm unattended TPM unlock plus return of the
      persistent mirrored display configuration.
- [x] Confirm retaining `vga: virtio` preserves the Proxmox noVNC boot console,
      including Limine snapshot selection and recovery control.
- [x] Dedicate the convenient lower blue rear USB port for the KVM upstream
      cable and identify its parent hub as `1-7.3` on PCH controller `00:14.0`.
- [x] Prove the four Greathtek socket paths: general USB 2 ports `1-7.3.1` and
      `1-7.3.2`; keyboard/mouse ports `1-7.3.4.1` and `1-7.3.4.2`.
- [x] Confirm a red Type-A port is also PCH-owned: the complete KVM tree
      appeared below hub `1-4` and disconnected together on a KVM switch.
- [x] Map all four stable Greathtek socket paths to VM 103:
      `usb0=1-7.3.1`, `usb1=1-7.3.4.1`, `usb2=1-7.3.4.2`, and
      `usb3=1-7.3.2`.
- [x] Test KVM disconnect/reconnect from inside `omoxy`: the illuminated
      keyboard, Logitech receiver/mouse, and SF-558 microphone appeared through
      the guest XHCI controller and disconnected cleanly on switch-back.
- [x] Prove the passed-through SF-558 microphone at application level with
      successful Voxtype speech-to-text input.

GPU-only, headless Hyprland, and Titan Ridge controller passthrough remain
unneeded alternative profiles rather than unfinished current-profile work.

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

Current profile: **hybrid physical/remote** (plain `vga: virtio` retained for
the noVNC boot/recovery console, with the RX 580 attached for physical HDMI and
Radeon-encoded Sunshine streaming).

Useful checks:

```bash
ssh root@hilbert qm status 103
ssh root@hilbert qm config 103
```

### Annotated current `qm` configuration

Snapshot from `qm config 103` on 2026-09-11. Lines beginning with `#` are
documentation annotations, not part of the captured Proxmox configuration.

```text
# QEMU guest agent is enabled on the Proxmox side.
agent: enabled=1

# OVMF/q35 VM; boot the installed disk before the attached installer ISO.
bios: ovmf
boot: order=scsi0;ide2
machine: q35
ostype: l26

# Host CPU model, four vCPUs, and 8 GiB RAM.
cores: 4
cpu: host
memory: 8192

# OVMF variables, encrypted root disk, installer ISO, and restored/tested vTPM.
efidisk0: local-zfs:vm-103-disk-0,efitype=4m,pre-enrolled-keys=0,size=1M
scsi0: local-zfs:vm-103-disk-1,discard=on,iothread=1,size=40G
scsihw: virtio-scsi-single
ide2: pve-storage_backups-isos:iso/omarchy-4.0.3.iso,media=cdrom,size=6113920K
tpmstate0: local-zfs:vm-103-disk-2,size=4M,version=v2.0

# Entire Radeon GPU plus HDMI-audio function; VirtIO remains for noVNC recovery.
hostpci0: 0000:01:00,pcie=1
vga: virtio

# VirtIO LAN on Hilbert's primary bridge.
net0: virtio=76:43:FF:F0:9B:AF,bridge=vmbr0,firewall=1

# Greathtek physical socket paths under the selected blue-port prefix 1-7.3.
# These match whatever non-hub device occupies each socket, not device IDs.
usb0: host=1-7.3.1
usb1: host=1-7.3.4.1
usb2: host=1-7.3.4.2
usb3: host=1-7.3.2

# Serial recovery socket and VM identity/state metadata.
serial0: socket
smbios1: uuid=1eab55ce-298e-4d53-bfae-da94ad292d72
vmgenid: f1bb1e8f-4dff-4f93-8a8c-c84c25d03594
meta: creation-qemu=7.2.10,ctime=1789068698
name: omoxy
onboot: 0
```

The installation completed in 1 minute 53 seconds. Select **Reboot Now** in the
Proxmox console. The VM boot order is `scsi0;ide2`, so the installed system disk
is tried before the still-attached installer ISO.

First boot succeeded: LUKS unlocked and the Omarchy desktop appeared correctly
through the VirtIO/noVNC display. Every boot since installation has displayed
`Unable to resume from device /dev/mapper/root` after LUKS unlock, but startup
continues normally. This predates PCI passthrough. The kernel command line
contains `resume=/dev/mapper/root resume_offset=1614914`, and the active swap
devices are an encrypted-root Btrfs swapfile plus zram. TPM-backed unattended
LUKS unlock is complete; investigate this resume warning separately.

Retaining VirtIO was revalidated. During a guest reboot, the Proxmox/OVMF boot
options screen appeared on both the physical RX 580 output and noVNC. After
that screen, the physical output went blank until the Omarchy desktop returned,
while noVNC continued to show the normal Omarchy/Arch boot path, including the
Limine snapshot selector. This is the observed behavior; the likely explanation
is that OVMF initializes both GOP outputs but Limine continues on the VirtIO
framebuffer. Preserve noVNC for Limine selection and recovery control.

After Hyprland starts, the two outputs are now one mirrored 1920x1080@60
desktop. The persistent rules are in `~/.config/hypr/monitors.lua` and were
verified after a clean Proxmox-managed VM reboot:

```lua
hl.monitor({ output = "HDMI-A-2", mode = "1920x1080@60", position = "0x0", scale = 1 })
hl.monitor({ output = "Virtual-1", mode = "1920x1080@60", position = "0x0", scale = 1, mirror = "HDMI-A-2" })
```

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
| Hover DNS alias for Tailscale IP | `omoxy.ts.imetrical.com` |

`omoxy.ts.imetrical.com` is not a Tailscale alias or a Tailscale naming
feature. It is simply a manually created Hover A record pointing at the
machine's Tailscale address. Tailscale's own MagicDNS name remains
`omoxy.tail62209.ts.net`.

The Bell Home Hub/Giga Hub DHCP reservation failure is a known router software
bug and is not expected to be fixable in this setup. The network is moving to
the existing UCG Fibre passthrough gateway; that migration is outside the
scope of VM 103. Until then, use Tailscale for stable management access.

`imetrical.com` uses Hover authoritative DNS. Existing host records establish
the convention: `<host>.imetrical.com` is an A record containing its
`192.168.2.x` LAN address, and `<host>.ts.imetrical.com` is an A record
containing its `100.x` Tailscale address. At discovery time,
`omoxy.imetrical.com` had no explicit record and fell through to Hover's
existing `216.40.34.41` wildcard/forwarding destination.

If/when the future gateway provides a stable LAN lease, create the optional LAN
record:

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

`hilbert` has two PCI USB controllers:

| Controller | PCI address | Buses | IOMMU group | Physical scope |
| ---------- | ----------- | ----- | ----------- | -------------- |
| Z390 PCH xHCI | `00:14.0` | 1/2 | 5, shared with PCH SRAM `00:14.2` | All rear Type-A ports (black/blue/yellow/red), internal USB headers, and onboard USB devices such as Bluetooth |
| Titan Ridge xHCI | `3c:00.0` | 3/4 | 24, isolated | The two rear Thunderbolt/USB-C ports |

Therefore the colored Type-A groups are not separately assignable PCI
controllers. Passing `00:14.0` would give the VM the blue KVM port, but would
also remove the top host keyboard/mouse USB pair and the other PCH USB paths
from Proxmox. Its shared IOMMU group also makes it a poor passthrough target.
The motherboard PS/2 port would remain independent.

Decision: keep the KVM on the selected blue Type-A port and map all four
Greathtek downstream socket paths. The mapping is unfortunately motherboard-
port-specific, but is stable in normal use, does not identify particular device
models, and is easy to relocate by rediscovering and replacing its prefix.

Passing `3c:00.0` is the clean whole-controller option. It is isolated in IOMMU
group 24, but the KVM upstream cable would have to move to one of the rear
USB-C ports (using the appropriate cable or adapter). The guest would then own
both rear USB-C ports and receive normal downstream hotplug events.

For the present blue Type-A connection, Proxmox/QEMU cannot pass the physical
Greathtek hub recursively. Map its downstream physical sockets instead:

```text
usb0: host=1-7.3.1    # general USB 2 socket 1
usb3: host=1-7.3.2    # general USB 2 socket 2
usb1: host=1-7.3.4.1  # keyboard/mouse socket 1
usb2: host=1-7.3.4.2  # keyboard/mouse socket 2
```

These are physical topology paths rather than transient USB device addresses,
so they pass whatever non-hub device occupies each Greathtek socket. Ordinary
device replacement, enumeration, and KVM switching should not alter them. The
prefix `1-7.3` pins the setup to the selected blue motherboard port. Relocating
the upstream cable only requires discovering a new prefix and updating all four
entries; the red-port test, for example, changed the prefix to `1-4` while
preserving the Greathtek suffixes `.1`, `.2`, `.4.1`, and `.4.2`.

The Titan Ridge USB controller is currently:

| PCI address | PCI ID      | IOMMU group | Current host driver |
| ----------- | ----------- | ----------- | ------------------- |
| `3c:00.0`   | `8086:15ec` | 24          | `xhci_hcd`          |

The old VM 100/Feynman note recorded this same Titan Ridge USB function as
`3b:00.0`. That was its valid PCI bus address at the time, not a different
controller or a physical Type-A port. PCI bridge bus numbering later shifted:
bus 3b is now an empty Thunderbolt downstream range and the USB function is now
`3c:00.0`. Always rediscover the BDF before reusing an old passthrough recipe.
The controller currently owns USB buses 3 and 4, and no devices were attached
when inspected. Passing it allows hot-plugging keyboard, mouse, and audio
devices connected through the two rear USB-C ports without individual USB
mappings. Passing only this xHCI function does not assign Titan Ridge's separate
Thunderbolt NHI function or its PCIe-tunnelling bridge tree.

VM 100's current configuration was inspected read-only and contains no active
`hostpci` or `usb` entries; its notes describe a historical configuration. VM
100 remains immutable for this project.

VM 100 (`feynman-production`, the Hackintosh) is immutable for this project:
we will never edit or clean up its configuration. If it must be restarted while
VM 103 owns the RX 580, shut down VM 103 first, then start VM 100.

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
The initial shared-display target is deliberately 1920x1080@60: both guest
outputs support it, and the physical RX 580 and VirtIO/noVNC views now show the
same desktop after reboot. Native 2560x1440 plus scaling, and the common
2048x1152 mode, are deferred until the shared-display behavior and Moonlight
capture path are fully settled.

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
