set -euo pipefail
TAG='v0.10.0-debug-0708'
IMAGE="swr.cn-southwest-2.myhuaweicloud.com/mabing/kruise-game-manager:${TAG}"
docker build . -f Dockerfile.debug -t "${IMAGE}"
docker save "${IMAGE}" -o /tmp/${TAG}.tar.gz
