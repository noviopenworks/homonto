#!/bin/sh
# Suite: release-packaging — run the real release build and verify the published
# artifacts: all three stamped binaries for every target (18 archives), the
# checksum file, extraction, stamped version strings, and a disposable-home
# smoke of an EXTRACTED binary (not a fresh local build).
set -eu
SUITE=release-packaging
. "$(dirname "$0")/lib.sh"

SRC="${SRC:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "$SRC"
VERSION="${REL_VERSION:-v0.0.0-e2e}"

log "build-release.sh cross-compiles all three binaries for every target"
rm -rf dist
bash scripts/build-release.sh "$VERSION" >/dev/null
ok "build-release.sh completed for $VERSION"

log "18 archives present (3 os x 2 arch x 3 binaries) + SHA256SUMS"
for oa in linux_amd64 linux_arm64 darwin_amd64 darwin_arm64; do
  is_file "dist/homonto_${VERSION}_${oa}.tar.gz"
  is_file "dist/onto_${VERSION}_${oa}.tar.gz"
  is_file "dist/to_${VERSION}_${oa}.tar.gz"
done
for oa in windows_amd64 windows_arm64; do
  is_file "dist/homonto_${VERSION}_${oa}.zip"
  is_file "dist/onto_${VERSION}_${oa}.zip"
  is_file "dist/to_${VERSION}_${oa}.zip"
done
is_file "dist/SHA256SUMS"
n="$(grep -c . dist/SHA256SUMS)"
[ "$n" = "18" ] || fail "SHA256SUMS should list 18 archives, lists $n"
ok "all 18 archives + an 18-line SHA256SUMS present"

log "checksums verify"
( cd dist && sha256sum -c SHA256SUMS >/dev/null ) || fail "SHA256SUMS did not verify"
ok "SHA256SUMS verifies every archive"

log "extract the native linux/amd64 archives and smoke all three binaries"
EX="$(mktemp -d)"
tar -C "$EX" -xzf "dist/homonto_${VERSION}_linux_amd64.tar.gz"
tar -C "$EX" -xzf "dist/onto_${VERSION}_linux_amd64.tar.gz"
tar -C "$EX" -xzf "dist/to_${VERSION}_linux_amd64.tar.gz"
HB="$EX/homonto_${VERSION}_linux_amd64/homonto"
OB="$EX/onto_${VERSION}_linux_amd64/onto"
TB="$EX/to_${VERSION}_linux_amd64/to"
is_file "$HB"; is_file "$OB"; is_file "$TB"
"$HB" version 2>&1 | grep -q "$VERSION" || fail "extracted homonto did not report the stamped version"
"$OB" version 2>&1 | grep -q "$VERSION" || fail "extracted onto did not report the stamped version"
"$TB" version 2>&1 | grep -q "$VERSION" || fail "extracted to did not report the stamped version"
ok "all three extracted binaries report the stamped version $VERSION"

log "disposable-home smoke of the EXTRACTED homonto (version/init/plan/apply/status)"
SH="$(mktemp -d)"; SW="$(mktemp -d)"
HOME="$SH" "$HB" init "$SW" >/dev/null
is_file "$SW/homonto.toml"
( cd "$SW" && HOME="$SH" "$HB" plan  >/dev/null )
( cd "$SW" && HOME="$SH" "$HB" apply --yes >/dev/null )
( cd "$SW" && HOME="$SH" "$HB" status 2>&1 | grep -q 'drift' ) || fail "status did not report a drift state"
ok "extracted-binary init/plan/apply/status smoke clean"

rm -rf dist
printf '\nSUITE PASS: %s\n' "$SUITE"
