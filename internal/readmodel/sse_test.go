package readmodel

import (
	"testing"
	"time"
)

func TestHub_구독자에게_밀어준다(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe("cust-1")

	hub.Publish("cust-1", []byte("hello"))

	select {
	case d := <-ch:
		if string(d) != "hello" {
			t.Fatalf("받은 값 = %q", d)
		}
	case <-time.After(time.Second):
		t.Fatal("타임아웃: 구독자가 못 받음")
	}
}

func TestHub_다른_회원에게는_안_간다(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe("cust-1")

	hub.Publish("cust-2", []byte("x")) // 다른 회원

	select {
	case <-ch:
		t.Fatal("다른 회원의 갱신이 오면 안 된다")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHub_해지하면_더는_안_받는다(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe("cust-1")
	hub.Unsubscribe("cust-1", ch)

	// 해지 시 채널을 닫으므로, 읽으면 닫힘(open=false)이어야 한다.
	if _, open := <-ch; open {
		t.Fatal("해지 후 채널은 닫혀 있어야 한다")
	}
	hub.Publish("cust-1", []byte("x")) // 구독자 없음 — 패닉 없이 통과해야
}
