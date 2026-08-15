// Package alert：告警投递（SMTP）与元监控心跳发射（docs/design/02-monitor-architecture.md
// §4/§7/§10）。Alerter 消费 alerts 表未投递行 → 邮件 → 标记 delivered；
// Heartbeat 是 collector 心跳发射方（M1-e 契约定稿，dialogue #26）。
package alert

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"arbcn/internal/store"
)

// SMTPConfig 是 SMTP 告警通道参数（环境变量注入，无密钥入库，§1 铁律）。
type SMTPConfig struct {
	Host string   // host:port（如 smtp.qq.com:587）
	User string   // 空 = 不认证
	Pass string   // 空 = 无密码
	From string   // 发件地址
	To   []string // 收件人（逗号分隔多收件人）
}

// Configured 报告投递所需最小参数齐备（Host/From/To）。
func (c SMTPConfig) Configured() bool {
	return c.Host != "" && c.From != "" && len(c.To) > 0
}

// Validate 校验投递参数格式（D-032：main 接线在启动 Alerter 前调用；
// 非法即降级禁用，不退出进程）。只查格式不连服务端，真实投递失败由
// Run 重试路径处理。
func (c SMTPConfig) Validate() error {
	if c.Host == "" {
		return errors.New("smtp host empty")
	}
	if _, err := smtpHost(c.Host); err != nil {
		return err
	}
	if c.From == "" {
		return errors.New("smtp from empty")
	}
	if len(c.To) == 0 {
		return errors.New("smtp to empty")
	}
	return nil
}

// smtpHost 拆出 host:port 的 host 部分并校验格式（PLAIN 认证与 TLS ServerName 用）。
// 端口须为 1..65535 十进制数——net.SplitHostPort 不拦空/非数字/越界端口，
// 此处补上（D-032：非法配置启动期降级，而非投递期反复失败）。
func smtpHost(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("alert: smtp host %q: %w", addr, err)
	}
	if host == "" {
		return "", fmt.Errorf("alert: smtp host %q: host empty", addr)
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("alert: smtp host %q: invalid port %q", addr, port)
	}
	return host, nil
}

// Alerter 轮询消费 alerts 未投递行：逐条 SMTP 投递，成功标记 delivered；
// 失败独立退避重试（至少一次语义：标记失败会重发，重复邮件优于漏报）。
// 与 collect.Scheduler / rule.Engine 同构：Run 阻塞至 ctx 取消，不中断进程。
type Alerter struct {
	St      store.Store                     // 必填
	SMTP    SMTPConfig                      // 必填（Run 时校验）
	Poll    time.Duration                   // 无积压时轮询间隔；0 = 10s
	Backoff func(attempt int) time.Duration // 失败退避；0 = 1s..32s 封顶
	Log     *slog.Logger                    // 0 = slog.Default()
}

// batchSize 单轮最多消费的告警数。
const batchSize = 100

// Start 是 main 接线入口（dialogue #27 门控 + D-032 修订）：SMTP 未配置或配置
// 非法 = warn + 降级禁用（不启动消费循环，告警行留在 alerts 表排队，配置修正后
// 重启补投，返回 false）；合法 = 启动 Run 消费循环（返回 true）。Run 的错误回传
// errCh（与 Scheduler/Engine 同款接线）。进程永不因 SMTP 配置问题退出。
func (a *Alerter) Start(ctx context.Context, errCh chan<- error) bool {
	if !a.SMTP.Configured() {
		a.log().Warn("SMTP not configured, alerts stay queued in DB",
			"hint", "set ARBCN_SMTP_HOST / ARBCN_SMTP_FROM / ARBCN_SMTP_TO")
		return false
	}
	if err := a.SMTP.Validate(); err != nil {
		a.log().Warn("SMTP config invalid, alerter disabled, alerts stay queued in DB", "err", err)
		return false
	}
	go func() { errCh <- a.Run(ctx) }()
	return true
}

// Run 消费循环：有积压紧转排空（不睡眠），全部投递后按 Poll 空闲轮询。
func (a *Alerter) Run(ctx context.Context) error {
	if a.St == nil {
		return errors.New("alert: alerter: nil store")
	}
	if err := a.SMTP.Validate(); err != nil {
		return fmt.Errorf("alert: alerter: %w", err)
	}
	attempt := 0
	for {
		sent, failed, err := a.sendRound(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			attempt++
			wait := a.backoff(attempt)
			a.log().Warn("alerter round failed, backing off", "attempt", attempt, "retry_in", wait.String(), "err", err)
			if !sleepCtx(ctx, wait) {
				return nil
			}
			continue
		}
		if failed > 0 {
			attempt++
			if !sleepCtx(ctx, a.backoff(attempt)) {
				return nil
			}
			continue
		}
		attempt = 0
		if sent == 0 && !sleepCtx(ctx, a.poll()) {
			return nil
		}
	}
}

// sendRound 消费一轮：逐条投递，成功标记；失败计数留待下轮重试。
// 返回 (sent, failed, err)；err = 读取待投递列表失败（存储层错误）。
func (a *Alerter) sendRound(ctx context.Context) (int, int, error) {
	alerts, err := a.St.PendingAlerts(ctx, batchSize)
	if err != nil {
		return 0, 0, fmt.Errorf("pending alerts: %w", err)
	}
	sent, failed := 0, 0
	for _, al := range alerts {
		if err := sendMail(ctx, a.SMTP, compose(a.SMTP, al)); err != nil {
			failed++
			a.log().Warn("alert send failed, will retry", "id", al.ID, "rule", al.RuleName, "err", err)
			continue
		}
		if err := a.St.MarkAlertDelivered(ctx, al.ID); err != nil {
			failed++ // 已投递但标记失败 → 下轮重发（至少一次语义）
			a.log().Warn("mark delivered failed, will resend", "id", al.ID, "err", err)
			continue
		}
		sent++
	}
	return sent, failed, nil
}

func (a *Alerter) poll() time.Duration {
	if a.Poll > 0 {
		return a.Poll
	}
	return 10 * time.Second
}

func (a *Alerter) backoff(attempt int) time.Duration {
	if a.Backoff != nil {
		return a.Backoff(attempt)
	}
	return time.Duration(1<<min(attempt, 5)) * time.Second // 1s..32s 封顶
}

func (a *Alerter) log() *slog.Logger {
	if a.Log != nil {
		return a.Log
	}
	return slog.Default()
}

// sleepCtx 可中断等待；ctx 取消返回 false（collect/rule 包同款）。
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// compose 组装告警邮件（RFC 5322）：标题 = 级别 + 规则名 + 时间；
// 正文 = 规则/级别/时间/消息（关键值在消息内——规则引擎 activeMsg 已含命中实体与数值）。
func compose(cfg SMTPConfig, al store.Alert) []byte {
	ts := al.Ts.UTC().Format(time.RFC3339)
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", cfg.From)
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(cfg.To, ", "))
	fmt.Fprintf(&b, "Subject: [arbcn][%s] %s %s\r\n", al.Level, al.RuleName, ts)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "rule: %s\n", al.RuleName)
	fmt.Fprintf(&b, "level: %s\n", al.Level)
	fmt.Fprintf(&b, "time: %s\n", ts)
	fmt.Fprintf(&b, "message: %s\n", al.Message)
	return []byte(b.String())
}

// smtpTimeout 边界整段 SMTP 会话（Dial 超时 + 连接 deadline 防服务端挂死拖住优雅退出）。
const smtpTimeout = 30 * time.Second

// sendMail 用 net/smtp 投递一封邮件。标准路径：EHLO → STARTTLS（服务端宣告才升级，
// 再 EHLO）→ AUTH PLAIN（配置 user 才认证；服务端不支持则报错）→ MAIL/RCPT/DATA。
// DialContext 响应 ctx 取消；真实 SMTP 联测待业主提供凭据后人工验证（任务约定）。
func sendMail(ctx context.Context, cfg SMTPConfig, msg []byte) error {
	host, err := smtpHost(cfg.Host)
	if err != nil {
		return err
	}
	conn, err := (&net.Dialer{Timeout: smtpTimeout}).DialContext(ctx, "tcp", cfg.Host)
	if err != nil {
		return err
	}
	_ = conn.SetDeadline(time.Now().Add(smtpTimeout))
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if err := c.Hello("arbcn"); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
		if err := c.Hello("arbcn"); err != nil {
			return err
		}
	}
	if cfg.User != "" {
		if ok, _ := c.Extension("AUTH"); !ok {
			return errors.New("alert: smtp: server does not support AUTH")
		}
		if err := c.Auth(smtp.PlainAuth("", cfg.User, cfg.Pass, host)); err != nil {
			return err
		}
	}
	if err := c.Mail(cfg.From); err != nil {
		return err
	}
	for _, rcpt := range cfg.To {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
