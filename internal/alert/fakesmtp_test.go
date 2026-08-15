package alert

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
)

// smtpMessage 是假服务器收到的一封邮件。
type smtpMessage struct {
	from string
	to   []string
	data string
}

// fakeSMTPServer：进程内最小 SMTP 服务端（net/smtp 对测）。应答标准序列
// （220→EHLO→MAIL/RCPT→DATA→QUIT），记录邮件；failMailN 让前 n 次 MAIL FROM
// 返回 550，驱动投递失败重试路径。连接同步串行处理（客户端逐封串行投递）。
type fakeSMTPServer struct {
	ln net.Listener

	mu       sync.Mutex
	msgs     []*smtpMessage
	failMail int
	mailSeen int
	closed   bool
}

func newFakeSMTP(t *testing.T) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("fake smtp listen: %v", err)
	}
	s := &fakeSMTPServer{ln: ln}
	go s.acceptLoop()
	t.Cleanup(s.close)
	return s
}

func (s *fakeSMTPServer) addr() string { return s.ln.Addr().String() }

func (s *fakeSMTPServer) failMailN(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failMail = n
}

func (s *fakeSMTPServer) messages() []*smtpMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*smtpMessage(nil), s.msgs...)
}

func (s *fakeSMTPServer) mailAttempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mailSeen
}

func (s *fakeSMTPServer) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	_ = s.ln.Close()
}

func (s *fakeSMTPServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return // listener closed
		}
		s.serve(conn)
	}
}

func (s *fakeSMTPServer) serve(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	var curFrom string
	var curTo []string
	send := func(line string) bool {
		_, err := fmt.Fprintf(conn, "%s\r\n", line)
		return err == nil
	}
	if !send("220 fake.local ESMTP arbcn") {
		return
	}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		switch {
		case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
			if !send("250-fake.local") || !send("250 AUTH PLAIN") {
				return
			}
		case strings.HasPrefix(line, "AUTH"):
			if !send("235 2.7.0 ok") {
				return
			}
		case strings.HasPrefix(line, "MAIL FROM:"):
			s.mu.Lock()
			s.mailSeen++
			reject := s.failMail > 0
			if reject {
				s.failMail--
			}
			s.mu.Unlock()
			if reject {
				if !send("550 5.1.0 rejected") {
					return
				}
				continue
			}
			curFrom = angleAddr(line)
			if !send("250 2.1.0 ok") {
				return
			}
		case strings.HasPrefix(line, "RCPT TO:"):
			curTo = append(curTo, angleAddr(line))
			if !send("250 2.1.5 ok") {
				return
			}
		case strings.HasPrefix(line, "DATA"):
			if !send("354 end data with <CR><LF>.<CR><LF>") {
				return
			}
			var data strings.Builder
			for {
				l, err := br.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(l, "\r\n") == "." {
					break
				}
				data.WriteString(l)
			}
			s.mu.Lock()
			s.msgs = append(s.msgs, &smtpMessage{from: curFrom, to: append([]string(nil), curTo...), data: data.String()})
			s.mu.Unlock()
			if !send("250 2.0.0 queued") {
				return
			}
		case strings.HasPrefix(line, "QUIT"):
			send("221 2.0.0 bye")
			return
		default:
			if !send("250 2.0.0 ok") {
				return
			}
		}
	}
}

// angleAddr 提取 "CMD:<addr>" 尖括号内的地址。
func angleAddr(s string) string {
	i := strings.IndexByte(s, '<')
	j := strings.IndexByte(s, '>')
	if i >= 0 && j > i {
		return s[i+1 : j]
	}
	return s
}
