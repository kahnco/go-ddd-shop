#!/usr/bin/env bash
# 암호화된 시크릿을 복호화해 클러스터에 적용한다 — 평문은 디스크에 남기지 않는다.
#   SOPS_AGE_KEY_FILE=deploy/sops/age.key bash deploy/sops/apply.sh
# 실서비스에서는 age 개인키를 파일이 아니라 KMS/키체인/CI 시크릿에서 주입한다.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ENC="$ROOT/deploy/k8s/secrets.enc.yaml"

: "${SOPS_AGE_KEY_FILE:=$ROOT/deploy/sops/age.key}"
export SOPS_AGE_KEY_FILE

echo "[sops] $ENC 복호화 → kubectl apply (평문은 파이프로만 흐른다)"
sops --decrypt "$ENC" | kubectl apply -f -
