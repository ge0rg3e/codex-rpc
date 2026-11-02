#!/bin/sh
# Install the latest native codex-rpc package and enable user-session autostart.
set -eu

fail() { printf '%s\n' "$*" >&2; exit 1; }
[ "$(uname -s)" = Linux ] || fail 'This installer is for Linux.'
[ "$(id -u)" -ne 0 ] || fail 'Run this as your normal user, without sudo. The package manager will request permission.'
case "$(uname -m)" in
  x86_64|amd64) ;;
  *) fail 'The current Linux release supports x86_64 only.' ;;
esac
command -v curl >/dev/null 2>&1 || fail 'Install curl first.'
[ -r /etc/os-release ] || fail 'Cannot detect this distribution: /etc/os-release is missing.'
. /etc/os-release
family=" ${ID:-} ${ID_LIKE:-} "
case "$family" in
  *' debian '*|*' ubuntu '*|*' linuxmint '*) kind=deb; asset=codex-rpc.deb ;;
  *' arch '*|*' manjaro '*|*' endeavouros '*) kind=arch; asset=codex-rpc.pkg.tar.zst ;;
  *' fedora '*|*' rhel '*|*' centos '*) kind=rpm; asset=codex-rpc.rpm ;;
  *) fail "Unsupported distribution: ${ID:-unknown}. Use the AppImage from GitHub Releases." ;;
esac
case "$kind" in
  deb) manager=apt-get ;;
  arch) manager=pacman ;;
  rpm) manager=dnf ;;
esac
command -v "$manager" >/dev/null 2>&1 || fail "The expected package manager ($manager) was not found."
command -v sudo >/dev/null 2>&1 || fail 'sudo is required to install the system package.'

workdir=$(mktemp -d)
trap 'rm -rf -- "$workdir"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
printf 'Downloading codex-rpc for %s (%s)...\n' "${PRETTY_NAME:-$ID}" "$kind"
curl --fail --location --show-error --proto '=https' --tlsv1.2 \
  "https://github.com/ge0rg3e/codex-rpc/releases/latest/download/$asset" -o "$workdir/$asset"
chmod 755 "$workdir"
chmod 644 "$workdir/$asset"
case "$kind" in
  deb) sudo apt-get install -y "$workdir/$asset" ;;
  arch) sudo pacman -U --needed --noconfirm "$workdir/$asset" ;;
  rpm) sudo dnf install -y "$workdir/$asset" ;;
esac
[ -x /usr/bin/codex-rpc ] || fail 'Package installation finished, but /usr/bin/codex-rpc is missing.'

config_dir=${XDG_CONFIG_HOME:-"$HOME/.config"}
mkdir -p "$config_dir/autostart"
startup_tmp=$(mktemp "$config_dir/autostart/.codex-rpc.XXXXXX")
cat > "$startup_tmp" <<'DESKTOP'
[Desktop Entry]
Type=Application
Name=codex-rpc
Comment=Discord presence for Codex
Exec=/usr/bin/codex-rpc
TryExec=/usr/bin/codex-rpc
Icon=codex-rpc
Terminal=false
X-GNOME-Autostart-enabled=true
DESKTOP
chmod 644 "$startup_tmp"
mv -f "$startup_tmp" "$config_dir/autostart/codex-rpc.desktop"
printf '\nInstalled. codex-rpc will start automatically when you sign in.\n'
printf 'To disable automatic startup, remove: %s/autostart/codex-rpc.desktop\n' "$config_dir"

if [ -n "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ]; then
  if command -v pgrep >/dev/null 2>&1 && pgrep -u "$(id -u)" -x codex-rpc >/dev/null; then
    printf 'codex-rpc is already running. Restart it to use the installed version.\n'
  else
    printf 'Opening codex-rpc...\n'
    nohup /usr/bin/codex-rpc --show </dev/null >/dev/null 2>&1 &
  fi
else
  printf 'Launch codex-rpc from your applications menu after signing in.\n'
fi
