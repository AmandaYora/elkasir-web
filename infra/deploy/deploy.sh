#!/usr/bin/env bash
# Elkasir server deploy — pull a ready image from GHCR, migrate, (re)start the app.
# The 2 GB VPS NEVER builds: CI builds & pushes the image; this only pulls & runs.
#
# Usage (on the VPS, from ~/elkasir):
#   ./deploy.sh <git-sha>     # deploy an exact, immutable image (recommended)
#   ./deploy.sh latest        # quick test only — not for real rollbacks
#
# Rollback = re-run with a previous <git-sha> (DB migrations are forward-only).
set -euo pipefail
cd "$(dirname "$0")"

TAG="${1:-latest}"
export IMAGE_TAG="$TAG"
IMAGE="ghcr.io/amandayora/elkasir-web:${TAG}"
COMPOSE="docker compose -f docker-compose.prod.yml"

echo "▶ [1/4] pull image: ${IMAGE}"
$COMPOSE pull

echo "▶ [2/4] run DB migrations (same image, 'migrate up' subcommand)"
docker run --rm \
  --add-host=host.docker.internal:host-gateway \
  --env-file .env \
  "${IMAGE}" migrate up

echo "▶ [3/4] start app"
$COMPOSE up -d

echo "▶ [4/4] verify readiness (/readyz)"
ok=0
for i in $(seq 1 10); do
  if curl -fsS http://127.0.0.1:8081/readyz >/dev/null 2>&1; then ok=1; break; fi
  sleep 2
done
if [ "$ok" = "1" ]; then
  echo "  ✓ app is ready (image tag: ${TAG})"
else
  echo "  ✗ /readyz did not pass — recent logs:"
  docker logs --tail 50 elkasir-app || true
  exit 1
fi

docker image prune -f >/dev/null 2>&1 || true
echo "✓ done. deployed ${IMAGE}"
