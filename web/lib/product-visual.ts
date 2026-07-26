// 상품 썸네일용 비주얼(자체 완결 — 외부 이미지 의존 없음).
// 이름으로 아이콘을, product_id 해시로 그라디언트를 고른다(같은 상품은 늘 같은 모습).

function pickGlyph(name: string): string {
  const table: [RegExp, string][] = [
    [/이어폰|헤드|버즈|오디오/, "🎧"],
    [/키보드/, "⌨️"],
    [/마우스/, "🖱️"],
    [/허브|어댑터|케이블|충전|독/, "🔌"],
    [/스탠드|거치/, "💻"],
    [/모니터|디스플레이/, "🖥️"],
    [/가방|백팩|파우치/, "🎒"],
    [/카메라|웹캠/, "📷"],
    [/스피커/, "🔊"],
  ];
  for (const [re, glyph] of table) if (re.test(name)) return glyph;
  return "📦";
}

// tailwind JIT 가 인식하도록 완전한 클래스 문자열 리터럴로 둔다(동적 조합 금지).
const gradients = [
  "from-blue-500/30 to-indigo-500/5",
  "from-emerald-500/30 to-teal-500/5",
  "from-amber-500/30 to-orange-500/5",
  "from-pink-500/30 to-rose-500/5",
  "from-violet-500/30 to-purple-500/5",
  "from-cyan-500/30 to-sky-500/5",
];

function pickGradient(id: string): string {
  let h = 0;
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) >>> 0;
  return gradients[h % gradients.length];
}

export function productVisual(p: { product_id: string; name: string }): {
  glyph: string;
  gradient: string;
} {
  return { glyph: pickGlyph(p.name), gradient: pickGradient(p.product_id) };
}
