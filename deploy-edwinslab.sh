#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

echo "======================================"
echo "       EDWINSLAB DEPLOYMENT"
echo "======================================"

# 1. Verify Git repository
git rev-parse --is-inside-work-tree >/dev/null
BRANCH=$(git branch --show-current)

if [[ -z "$BRANCH" ]]; then
  echo "ERROR: No active Git branch."
  exit 1
fi

echo "Branch: $BRANCH"
echo "Remote:"
git remote -v

# 2. Protect local secrets
if git ls-files | grep -E '(^|/)(config\.local\.h|\.env|\.env\..+|credentials\.json)$'; then
  echo "ERROR: Sensitive files are tracked by Git."
  echo "Review and remove them from Git before deployment."
  exit 1
fi

# 3. Backend tests
echo
echo "=== BACKEND VALIDATION ==="
cd "$ROOT/backend"
go test ./...
go vet ./...
go build ./...

# 4. Frontend build
echo
echo "=== FRONTEND VALIDATION ==="
cd "$ROOT/frontend"
npm ci
npm run build

# 5. Firmware build
echo
echo "=== ESP32 FIRMWARE BUILD ==="
cd "$ROOT/firmware/esp32"
pio run -e esp32dev

# 6. Review and stage changes
cd "$ROOT"
echo
echo "=== GIT STATUS ==="
git status --short

read -r -p "Stage and commit the listed project changes? (y/N): " answer
if [[ "$answer" != "y" && "$answer" != "Y" ]]; then
  echo "Stopped before committing."
  exit 0
fi

git add -A -- firmware/esp32 backend frontend FINGERPRINT_ENROLLMENT_FIX.md

# Refuse suspicious staged filenames
if git diff --cached --name-only | grep -Ei '(^|/)(\.env($|\.)|config\.local\.h$|credentials\.json$|.*secret.*|.*service.account.*|.*private.key$)'; then
  echo "ERROR: Potential secret staged. Review with git status."
  git reset
  exit 1
fi

if git diff --cached --quiet; then
  echo "No new changes to commit."
else
  git diff --cached --stat
  git commit -m "Complete biometric enrollment and live device monitoring"
fi

# 7. Push to GitHub
echo
echo "=== PUSHING TO GITHUB ==="
git push -u origin "$BRANCH"

echo
echo "GitHub push successful."
echo "Render and Vercel automatic deployments should now start."

# 8. Upload ESP32 firmware
echo
read -r -p "Is ESP32 connected on /dev/ttyUSB0? Upload firmware now? (y/N): " upload

if [[ "$upload" == "y" || "$upload" == "Y" ]]; then
  cd "$ROOT/firmware/esp32"
  pio run -e esp32dev -t upload --upload-port /dev/ttyUSB0
  echo "Firmware upload completed."
else
  echo "Firmware upload skipped."
fi

echo
echo "======================================"
echo "       WORKFLOW FINISHED"
echo "======================================"
echo "Check Render and Vercel deployment status."
echo "Database migrations were NOT executed."
echo "Physical enrollment testing is still required."
