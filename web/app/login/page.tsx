import Link from "next/link";
import { getSession } from "@/lib/session";
import LoginForm from "./LoginForm";

export default async function LoginPage() {
  const session = await getSession();

  if (session) {
    return (
      <div className="mx-auto max-w-sm text-center">
        <p className="text-neutral-300">
          이미 로그인되어 있습니다.
          <br />
          <code className="text-xs text-neutral-500">{session.customerId}</code>
        </p>
        <div className="mt-6 flex justify-center gap-3 text-sm">
          <Link href="/orders" className="rounded-lg bg-white/5 px-4 py-2 hover:bg-white/10">
            내 주문 보기
          </Link>
          <Link href="/" className="rounded-lg bg-white/5 px-4 py-2 hover:bg-white/10">
            쇼핑 계속하기
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div>
      <h1 className="mb-8 text-center text-2xl font-bold">로그인 / 회원가입</h1>
      <LoginForm />
    </div>
  );
}
