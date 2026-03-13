#!/usr/bin/env bash
set -euo pipefail

command -v docker >/dev/null 2>&1 || {
  echo "error: docker not found"
  exit 1
}

TTL="${1:-2h}"
TAG="${2:-}"

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${DIR}/.." && pwd)"
MANIFEST="${ROOT_DIR}/k8s/deployment.yaml"

if [ ! -f "${ROOT_DIR}/Dockerfile" ]; then
  echo "error: Dockerfile not found at ${ROOT_DIR}/Dockerfile"
  exit 1
fi

if [ ! -f "${MANIFEST}" ]; then
  echo "error: deployment manifest not found at ${MANIFEST}"
  exit 1
fi

if [ -n "${TAG}" ]; then
  IMAGE="ttl.sh/wolfword-${TAG}:${TTL}"
else
  IMAGE="ttl.sh/wolfword:${TTL}"
fi

echo "==> Building wolfword image"
docker build -t "${IMAGE}" "${ROOT_DIR}"

echo "==> Pushing image to ttl.sh"
docker push "${IMAGE}"

echo "==> Patching deployment manifest"
sed -i "s|image: [^ ]*|image: ${IMAGE}|" "${MANIFEST}"

echo
echo "Image pushed:"
echo "  ${IMAGE}"
echo
echo "Deploy:"
echo "  kubectl apply -f ${ROOT_DIR}/k8s/deployment.yaml -f ${ROOT_DIR}/k8s/service.yaml"
