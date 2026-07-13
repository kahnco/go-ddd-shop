package eventbus

// 스키마 진화(schema evolution) 지원.
//
// 이벤트는 JetStream 에 영구 저장되고, 소비자는 과거 이벤트까지 재생(replay)한다.
// 그래서 시간이 지나 이벤트 모양이 바뀌면, 스트림 속 "옛 버전" 이벤트도 최신 소비자가
// 읽을 수 있어야 한다. 두 가지로 다룬다.
//
//   - 더하기(additive) 변경: 필드 추가. 관용적 리더(tolerant reader)면 그냥 통과한다 —
//     Go 의 json.Unmarshal 은 모르는 필드를 무시하고, 빠진 필드는 zero-value 가 된다.
//     이런 변경엔 업캐스터가 필요 없다.
//   - 깨는(breaking) 변경: 필드 이름 변경·타입 변경·의미 변경. 버전을 올리고,
//     "옛 버전 → 새 버전"으로 바꿔 주는 업캐스터(upcaster)를 등록한다.
//
// 소비 직전에 upcast 가 등록된 업캐스터를 차례로 적용해, 핸들러는 늘 최신 모양만 본다.

// Upcaster 는 한 버전의 봉투를 다음 버전으로 변환한다(주로 Data JSON 을 손본다).
type Upcaster func(Envelope) Envelope

// name → fromVersion → 업캐스터. 시작 시(구독 전) 한 번 등록하고, 이후엔 읽기만 한다.
var upcasters = map[string]map[int]Upcaster{}

// name → 최신 스키마 버전(등록된 업캐스터로부터 계산).
var latestSchema = map[string]int{}

// RegisterUpcaster 는 "이벤트 name 의 fromVersion 을 fromVersion+1 로 올리는" 변환을 등록한다.
// 반드시 서비스 기동 시(구독을 걸기 전)에 호출해야 한다 — 이후엔 동시 읽기만 일어난다.
func RegisterUpcaster(name string, fromVersion int, fn Upcaster) {
	if upcasters[name] == nil {
		upcasters[name] = map[int]Upcaster{}
	}
	upcasters[name][fromVersion] = fn
	if to := fromVersion + 1; to > latestSchema[name] {
		latestSchema[name] = to
	}
}

// latestVersion 은 이 이벤트의 최신 스키마 버전을 돌려준다(업캐스터가 없으면 1).
func latestVersion(name string) int {
	if v, ok := latestSchema[name]; ok {
		return v
	}
	return 1
}

// upcast 는 봉투를 최신 버전까지 끌어올린다. 등록된 업캐스터가 없으면 그대로 둔다.
// 업캐스터는 멱등하게 작성하는 게 안전하다(이미 새 필드가 있으면 건드리지 않기).
func upcast(env Envelope) Envelope {
	v := env.SchemaVersion
	if v == 0 {
		v = 1 // 버전 없는 옛 이벤트는 v1 로 본다
	}
	for {
		fn, ok := upcasters[env.Name][v]
		if !ok {
			break
		}
		env = fn(env)
		v++
		env.SchemaVersion = v
	}
	return env
}
