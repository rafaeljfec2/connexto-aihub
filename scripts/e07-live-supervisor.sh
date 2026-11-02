#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
	cat <<'EOF'
Usage:
  scripts/e07-live-supervisor.sh --preflight
  scripts/e07-live-supervisor.sh --run-live

--preflight runs only bounded local E07 checks against synthetic endpoints.
--run-live runs one real-provider E07 pass and requires separate authorization.
EOF
}

mode="${1:-}"
if [[ "$mode" != "--preflight" && "$mode" != "--run-live" ]]; then
	usage >&2
	exit 2
fi

if [[ "${2:-}" == "--supervised" ]]; then
	watchdog_parent=""
	watchdog_command=()
	read -r watchdog_parent <"/proc/$PPID/comm" || true
	watchdog_valid=false
	if [[ "${AIHUB_E07_WATCHDOG_ACTIVE:-}" == "1" && "$watchdog_parent" == "timeout" ]] &&
		mapfile -d '' -t watchdog_command <"/proc/$PPID/cmdline"; then
		if [[ ${#watchdog_command[@]} -eq 7 &&
			"${watchdog_command[0]}" == "/usr/bin/timeout" &&
			"${watchdog_command[1]}" == "--signal=KILL" &&
			"${watchdog_command[2]}" == "30m" &&
			"${watchdog_command[3]}" == "/bin/bash" &&
			"${watchdog_command[4]}" == "$0" &&
			"${watchdog_command[5]}" == "$mode" &&
			"${watchdog_command[6]}" == "--supervised" ]]; then
			watchdog_valid=true
		fi
	fi
	if [[ "$watchdog_valid" != "true" ]]; then
		printf 'E07 supervised mode requires the outer 30-minute watchdog\n' >&2
		exit 2
	fi
elif [[ $# -eq 1 ]]; then
	exec /usr/bin/env -i "PATH=${PATH:-/usr/bin:/bin}" "HOME=${HOME:-/home/rafael}" \
		AIHUB_E07_WATCHDOG_ACTIVE=1 \
		/usr/bin/timeout --signal=KILL 30m \
		/bin/bash "$0" "$mode" --supervised
	else
	usage >&2
	exit 2
fi

repository_root="$(git rev-parse --show-toplevel)"
script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
go_path="$(command -v go)"
codex_path="$(command -v codex)"
module_cache="$(go env GOMODCACHE)"
go_bin_dir="$(dirname "$(readlink -f "$go_path")")"
codex_bin_dir="$(dirname "$(readlink -f "$codex_path")")"
numeric_uid="$(id -u)"
numeric_gid="$(id -g)"

for required in /usr/bin/bwrap /usr/bin/timeout /usr/bin/env /usr/bin/python3 /bin/sh; do
	if [[ ! -x "$required" ]]; then
		printf 'E07 supervisor prerequisite missing: %s\n' "${required##*/}" >&2
		exit 1
	fi
done

if [[ ! -r "$repository_root/.env" || ! -d "$module_cache" || ! -d "$codex_bin_dir" ]]; then
	env_file_readable=false
	module_cache_available=false
	codex_binary_available=false
	[[ -r "$repository_root/.env" ]] && env_file_readable=true
	[[ -d "$module_cache" ]] && module_cache_available=true
	[[ -d "$codex_bin_dir" ]] && codex_binary_available=true
	printf 'E07 supervisor preflight failed: env=%s module_cache=%s codex_binary=%s\n' \
		"$env_file_readable" "$module_cache_available" "$codex_binary_available" >&2
	exit 1
fi

private_root=/run/e07/private
go_cache="$private_root/go-cache"
test_binary="$private_root/bin/ai-hub-e07.test"

bwrap_args=(
	--unshare-pid
	--unshare-ipc
	--unshare-uts
	--ro-bind
	/
	/
	--dev /dev
	--proc /proc
	--tmpfs /run
	--dir /run/e07
	--dir /run/e07/gomodcache
	--ro-bind "$module_cache" /run/e07/gomodcache
	--remount-ro /run/e07/gomodcache
	--dir /run/e07/codex-bin
	--ro-bind "$codex_bin_dir" /run/e07/codex-bin
	--remount-ro /run/e07/codex-bin
	--tmpfs "$private_root"
	--remount-ro /run
	--tmpfs /tmp
	--dir /tmp/e07
	--ro-bind "$repository_root" /tmp/e07/repository
	--tmpfs /home
	--dir /home/e07-homologation
	--tmpfs /home/e07-homologation/.codex
	--remount-ro /home/e07-homologation/.codex
	--ro-bind-data 3 /etc/passwd
	--die-with-parent
	--chdir /tmp/e07/repository
	--
)

inner_env=(
	"PATH=$go_bin_dir:/usr/bin:/bin:/run/e07/codex-bin"
	LANG=C.UTF-8
	HOME="$private_root/home"
	CODEX_HOME="$private_root/codex-home"
	CODEX_SQLITE_HOME="$private_root/codex-home"
	TMPDIR="$private_root/tmp"
	TEMP="$private_root/tmp"
	TMP="$private_root/tmp"
	XDG_CONFIG_HOME="$private_root/xdg/config"
	XDG_DATA_HOME="$private_root/xdg/data"
	XDG_CACHE_HOME="$private_root/xdg/cache"
	XDG_STATE_HOME="$private_root/xdg/state"
	XDG_RUNTIME_DIR="$private_root/xdg/runtime"
	GOCACHE="$go_cache"
	GOMODCACHE=/run/e07/gomodcache
	GOPATH="$private_root/gopath"
	GOTOOLCHAIN=local
	AIHUB_E07_PRIVATE_ROOT="$private_root"
)

run_status=0
if /usr/bin/bwrap "${bwrap_args[@]}" 3< <(
	printf 'e07-homologation:x:%s:%s:E07 Homologation:/home/e07-homologation:/bin/sh\n' \
		"$numeric_uid" "$numeric_gid"
) /usr/bin/env -i "${inner_env[@]}" /bin/sh -ceu '
	mode="$1"
	private_root="$AIHUB_E07_PRIVATE_ROOT"
	mkdir -p "$HOME" "$CODEX_HOME" "$TMPDIR" \
		"$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_CACHE_HOME" \
		"$XDG_STATE_HOME" "$XDG_RUNTIME_DIR" "$GOCACHE" "$GOPATH" \
		"$private_root/bin"
	chmod 700 "$XDG_RUNTIME_DIR"
	python3 -c "import errno, os; path=\".env\"; os.stat(path); fd=None
try:
 fd=os.open(path, os.O_WRONLY)
except OSError as error:
 if error.errno not in (errno.EROFS, errno.EACCES): raise
else:
 os.close(fd)
 raise SystemExit(\"repository dotenv mount accepted a writable open\")
print(\"E07 repository dotenv mount: read-only\")"
	go test -c -o "$private_root/bin/ai-hub-e07.test" ./cmd/ai-hub

	if [ "$mode" = "--preflight" ]; then
		exec /usr/bin/env AIHUB_E07_CANARY=1 AIHUB_E07_CREDENTIAL_PREFLIGHT=1 \
			AIHUB_E07_MOUNT_BOUNDARY_MODE=parent \
			"$private_root/bin/ai-hub-e07.test" \
			-test.run="^TestE07" -test.count=1 -test.v -test.timeout=29m
	fi

	exec /usr/bin/env AIHUB_E07_LIVE=1 "$private_root/bin/ai-hub-e07.test" \
		-test.run="^TestE07LiveCodexOpenCodeGo$" -test.count=1 -test.v -test.timeout=29m
' e07-inner "$mode"; then
	run_status=0
else
	run_status=$?
fi

exit "$run_status"
