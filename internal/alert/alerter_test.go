package alert

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"arbcn/internal/store"
)

// testAlerter：毫秒级轮询/退避 + 静默日志的确定性 Alerter。
func testAlerter(st store.Store, smtp SMTPConfig) *Alerter {
	return &Alerter{
		St:      st,
		SMTP:    smtp,
		Poll:    2 * time.Millisecond,
		Backoff: func(int) time.Duration { return time.Millisecond },
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// TestCompose：标题含规则名/level/时间，正文含消息与关键值。
func TestCompose(t *testing.T) {
	ts := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	msg := string(compose(SMTPConfig{From: "arbcn@local", To: []string{"a@x.test", "b@x.test"}}, store.Alert{
		RuleName: "funding_critical", Ts: ts, Level: store.LevelCritical,
		Message: "funding_critical active: BTC@binance=21, ETH@okx=21.5",
	}))

	for _, want := range []string{
		"From: arbcn@local\r\n",
		"To: a@x.test, b@x.test\r\n",
		"Subject: [arbcn][critical] funding_critical 2026-08-15T12:00:00Z\r\n",
		"rule: funding_critical\n",
		"level: critical\n",
		"time: 2026-08-15T12:00:00Z\n",
		"message: funding_critical active: BTC@binance=21, ETH@okx=21.5\n",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("mail missing %q:\n%s", want, msg)
		}
	}
}

// TestAlerterSendMarkDelivered：消费 → 假 SMTP 投递 → 标记 delivered 全路径；
// 多收件人逐一 RCPT；无积压时空闲轮询；取消后 Run 及时返回。
func TestAlerterSendMarkDelivered(t *testing.T) {
	srv := newFakeSMTP(t)
	st := newMemStore()
	ts := time.Now().UTC().Truncate(time.Second)
	if err := st.InsertAlert(context.Background(), store.Alert{
		RuleID: 1, RuleName: "funding_critical", Ts: ts, Level: store.LevelCritical,
		Message: "funding_critical active: BTC@binance=21",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertAlert(context.Background(), store.Alert{
		RuleID: 2, RuleName: "usdcnh_buy_line", Ts: ts.Add(time.Second), Level: store.LevelWarn,
		Message: "usdcnh_buy_line active: USDCNH@sina=6.55",
	}); err != nil {
		t.Fatal(err)
	}

	a := testAlerter(st, SMTPConfig{Host: srv.addr(), From: "arbcn@local", To: []string{"a@x.test", "b@x.test"}})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = a.Run(ctx); close(done) }()

	// 等 delivered（因果序上晚于服务器记录邮件：标记发生在 QUIT 之后）。
	waitFor(t, 2*time.Second, func() bool { return st.deliveredCount() == 2 })
	msgs := srv.messages()
	if len(msgs) != 2 {
		t.Fatalf("received mails = %d, want 2", len(msgs))
	}
	wantSubj := "Subject: [arbcn][critical] funding_critical " + ts.Format(time.RFC3339)
	if !strings.Contains(msgs[0].data, wantSubj) {
		t.Errorf("mail 0 subject missing %q:\n%s", wantSubj, msgs[0].data)
	}
	if !strings.Contains(msgs[0].data, "message: funding_critical active: BTC@binance=21") {
		t.Errorf("mail 0 body:\n%s", msgs[0].data)
	}
	if msgs[0].from != "arbcn@local" || len(msgs[0].to) != 2 || msgs[0].to[0] != "a@x.test" || msgs[0].to[1] != "b@x.test" {
		t.Errorf("mail 0 envelope = %q/%v, want arbcn@local/[a@x.test b@x.test]", msgs[0].from, msgs[0].to)
	}
	if !strings.Contains(msgs[1].data, "[arbcn][warn] usdcnh_buy_line") {
		t.Errorf("mail 1 subject/level:\n%s", msgs[1].data)
	}

	// 排空后进入空闲轮询（继续调 PendingAlerts），且投递数不再增长。
	waitFor(t, 500*time.Millisecond, func() bool { return st.pendCallCount() >= 3 })
	if n := st.deliveredCount(); n != 2 {
		t.Fatalf("delivered drifted = %d, want 2", n)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

// TestAlerterRetriesAfterFailure：投递失败（550）独立退避重试，不阻塞同轮其他行；
// 恢复后积压全部排空。
func TestAlerterRetriesAfterFailure(t *testing.T) {
	srv := newFakeSMTP(t)
	srv.failMailN(2)
	st := newMemStore()
	ts := time.Now().UTC()
	for i := 0; i < 2; i++ {
		if err := st.InsertAlert(context.Background(), store.Alert{
			RuleID: int64(i + 1), RuleName: "r" + string(rune('a'+i)), Ts: ts.Add(time.Duration(i) * time.Second),
			Level: store.LevelWarn, Message: "r" + string(rune('a'+i)) + " active",
		}); err != nil {
			t.Fatal(err)
		}
	}

	a := testAlerter(st, SMTPConfig{Host: srv.addr(), From: "arbcn@local", To: []string{"a@x.test"}})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = a.Run(ctx); close(done) }()

	// 前 2 次 MAIL FROM 被拒 → 下轮重试成功 → 两行全投递。
	waitFor(t, 2*time.Second, func() bool { return st.deliveredCount() == 2 })
	if got := srv.mailAttempts(); got != 4 {
		t.Errorf("mail attempts = %d, want 4 (2 拒绝 + 2 成功)", got)
	}
	if got := len(srv.messages()); got != 2 {
		t.Errorf("received mails = %d, want 2", got)
	}
	cancel()
	<-done
}

// TestAlerterStoreErrorBacksOffAndRecovers：读待投递列表失败走退避（不崩不堵），
// 存储恢复后积压照常排空。
func TestAlerterStoreErrorBacksOffAndRecovers(t *testing.T) {
	srv := newFakeSMTP(t)
	st := newMemStore()
	st.setPendErr(errors.New("db down"))
	if err := st.InsertAlert(context.Background(), store.Alert{
		RuleID: 1, RuleName: "funding_warn", Ts: time.Now().UTC(), Level: store.LevelWarn, Message: "funding_warn active",
	}); err != nil {
		t.Fatal(err)
	}

	a := testAlerter(st, SMTPConfig{Host: srv.addr(), From: "arbcn@local", To: []string{"a@x.test"}})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = a.Run(ctx); close(done) }()

	waitFor(t, time.Second, func() bool { return st.pendCallCount() >= 3 })
	if st.deliveredCount() != 0 {
		t.Fatal("alert delivered while store broken")
	}

	st.setPendErr(nil)
	waitFor(t, time.Second, func() bool { return st.deliveredCount() == 1 })
	cancel()
	<-done
}

// TestAlerterRequiresSMTPConfig：投递参数不齐 fail fast（不进入消费循环）。
func TestAlerterRequiresSMTPConfig(t *testing.T) {
	cases := []SMTPConfig{
		{From: "x@local", To: []string{"a@x.test"}},                  // 缺 host
		{Host: "h:25", To: []string{"a@x.test"}},                     // 缺 from
		{Host: "h:25", From: "x@local"},                              // 缺 to
		{Host: "badhost", From: "x@local", To: []string{"a@x.test"}}, // 缺端口
	}
	for i, c := range cases {
		a := &Alerter{St: newMemStore(), SMTP: c}
		if err := a.Run(context.Background()); err == nil {
			t.Errorf("case %d: Run = nil error, want fail fast", i)
		}
	}
}
