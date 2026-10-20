#!/usr/bin/env bash
# Clean delivery zip for the locked ChitChat-App project.
# Allowlist + hard excludes: only working source ships, never runtime data.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAMP="$(date +%Y-%m-%d_%H%M)"
OUT="$ROOT/release/ChitChat-App-clean-$STAMP.zip"

cd "$ROOT"
mkdir -p release

# ---- build the file list (allowlist roots, then hard excludes) -------------
LIST="$(mktemp)"
find README-BANGLA.txt install.bat run-dashboard.bat test.bat _packaging source_code -type f \
  ! -path '*/.git/*' \
  ! -path 'source_code/data/*' \
  ! -path 'source_code/node_modules/*' \
  ! -path 'source_code/verification-output/*' \
  ! -path 'source_code/build/bin/*' \
  ! -path 'source_code/capture/capture-sessions/*' \
  ! -path 'source_code/capture/profiles/*' \
  ! -path '*/.gocache/*' \
  ! -path 'source_code/dist/*' \
  ! -name '*.exe' ! -name '*.dll' ! -name '*.zip' \
  ! -name '*.log' ! -name '*.tmp' ! -name '*.sst' ! -name '*.vlog' \
  ! -name 'logs.txt' ! -name 'install.log' \
  ! -name '.DS_Store' ! -name 'Thumbs.db' \
  | LC_ALL=C sort > "$LIST"

[ -s "$LIST" ] || { echo "ERROR: nothing to package" >&2; exit 1; }

# ---- sanity gate: refuse forbidden paths -----------------------------------
if grep -Eq '^(source_code/)?(data/|node_modules/|verification-output/|build/bin/)' "$LIST"; then
  echo "ERROR: forbidden runtime path in package list" >&2; exit 1
fi
if grep -Eq '\.(exe|dll|log|sst|vlog)$' "$LIST"; then
  echo "ERROR: binary/log file in package list" >&2; exit 1
fi

# ---- required files gate: the build must stay possible --------------------
REQUIRED="README-BANGLA.txt
_packaging/make-clean-zip.sh
install.bat
run-dashboard.bat
test.bat
source_code/install.bat
source_code/run-dashboard.bat
source_code/test.bat
source_code/verify.bat
source_code/wails.json
source_code/go.mod
source_code/go.sum
source_code/main.go
source_code/app.go
source_code/cmd/chitchatui/frontend/web/app.js
source_code/cmd/chitchatui/frontend/web/index.html
source_code/build/appicon.png
source_code/build/windows/icon.ico
source_code/build/windows/info.json
source_code/build/windows/wails.exe.manifest
source_code/internal/bot/eva_assets/README.txt
source_code/internal/bot/eva_assets/persona.txt"
while IFS= read -r need; do
  [ -n "$need" ] || continue
  grep -qxF "$need" "$LIST" || { echo "ERROR: required file missing: $need" >&2; exit 1; }
done <<< "$REQUIRED"

# ---- gate: no frontend assets under a "dist" folder (they get stripped) ---
if grep -q 'frontend/dist/' "$LIST"; then
  echo "ERROR: dashboard assets must live in frontend/web, not frontend/dist" >&2
  exit 1
fi

# ---- gate: every .bat must be CRLF ----------------------------------------
while IFS= read -r bat; do
  case "$bat" in
    *.bat)
      if ! head -n 1 "$bat" | grep -q $'\r'; then
        echo "ERROR: $bat must use CRLF line endings" >&2
        exit 1
      fi
      ;;
  esac
done < "$LIST"

# ---- zip (deterministic-ish, no OS metadata) ------------------------------
rm -f "$OUT"
zip -X -9 -q -@ "$OUT" < "$LIST"
rm -f "$LIST"

# ---- report ---------------------------------------------------------------
FILES=$(unzip -Z1 "$OUT" | wc -l | tr -d ' ')
SIZE=$(du -h "$OUT" | cut -f1)
RAW=$(unzip -l "$OUT" | tail -1 | awk '{print $1}')
echo "clean zip : $OUT"
echo "files     : $FILES"
echo "size      : $SIZE (uncompressed ${RAW} bytes)"
echo "top level :"
unzip -Z1 "$OUT" | awk -F/ '{print "  - "$1}' | sort -u
