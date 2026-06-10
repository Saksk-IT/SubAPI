#!/usr/bin/env bash
# 构建/发布镜像的快速脚本，始终使用仓库根目录作为 Docker build context。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

IMAGE="${IMAGE:-sub2api:latest}"
PLATFORM="${PLATFORM:-}"
PUSH="${PUSH:-0}"
GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
GOSUMDB="${GOSUMDB:-sum.golang.google.cn}"

is_push_enabled() {
    case "${PUSH}" in
        1|true|TRUE|yes|YES|y|Y) return 0 ;;
        *) return 1 ;;
    esac
}

if [ -n "${PLATFORM}" ] || is_push_enabled; then
    build_cmd=(docker buildx build)
    if [ -n "${PLATFORM}" ]; then
        build_cmd+=(--platform "${PLATFORM}")
    fi
    if is_push_enabled; then
        build_cmd+=(--push)
    else
        build_cmd+=(--load)
    fi
else
    build_cmd=(docker build)
fi

"${build_cmd[@]}" \
    -t "${IMAGE}" \
    --build-arg "GOPROXY=${GOPROXY}" \
    --build-arg "GOSUMDB=${GOSUMDB}" \
    -f "${REPO_ROOT}/Dockerfile" \
    "${REPO_ROOT}"
