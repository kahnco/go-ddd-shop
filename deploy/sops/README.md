# 시크릿 관리 — SOPS + age

DB 비밀번호·DSN·JWT 서명 키 같은 **비밀은 평문으로 git 에 커밋하지 않는다.**
값만 암호화한 [`../k8s/secrets.enc.yaml`](../k8s/secrets.enc.yaml) 로 분리해, 복호화는 배포 시점에만 한다.

## 왜 SOPS+age 인가

- **암호화된 채로 git 에** — 시크릿도 버전 관리·리뷰·롤백이 된다.
- **값만 암호화**(`encrypted_regex: ^(data|stringData)$`) — `kind`·`metadata` 는 평문이라 diff 가 읽힌다.
- **age** 는 KMS 없이도 쓸 수 있는 단순·현대적 키. 실서비스에선 recipient 를 AWS/GCP KMS 나
  조직 키로 바꾸면 그대로 확장된다.

## 키

- 공개 recipient 는 [`../../.sops.yaml`](../../.sops.yaml) 에 커밋한다(공개키는 안전).
- **개인키(`age.key`)는 절대 커밋하지 않는다** — `.gitignore` 에 있다.
  실서비스에서는 파일이 아니라 **볼트·키체인·CI 시크릿** 에서 `SOPS_AGE_KEY_FILE`(또는 `SOPS_AGE_KEY`)로 주입한다.

## 워크플로

```bash
# 0) (최초 1회) 자기 키를 만들고 recipient 를 .sops.yaml 에 넣는다
age-keygen -o deploy/sops/age.key            # "public key: age1..." 출력
#   → 그 recipient 로 .sops.yaml 의 age: 를 교체

# 1) 값 채워 암호화
cp deploy/k8s/secrets.example.yaml deploy/k8s/secrets.filled.yaml
$EDITOR deploy/k8s/secrets.filled.yaml       # 실제 값 입력(파일은 gitignore)
sops --encrypt deploy/k8s/secrets.filled.yaml > deploy/k8s/secrets.enc.yaml
rm deploy/k8s/secrets.filled.yaml

# 2) 편집(복호화→에디터→재암호화가 한 번에)
sops deploy/k8s/secrets.enc.yaml

# 3) 적용(평문은 파이프로만)
SOPS_AGE_KEY_FILE=deploy/sops/age.key bash deploy/sops/apply.sh
```

## 심층 방어

암호화만으론 부족하다 — 누군가 시크릿 주입을 깜빡할 수 있다. 그래서 서비스는 부팅 때
`APP_ENV=production` 인데 `JWT_SECRET` 이 없거나 개발용 기본값이면 **뜨는 걸 거부** 한다
(`internal/platform/auth`). 잘못된 배포가 조용히 약한 키로 도는 것보다, 시끄럽게 멈추는 게 낫다.
