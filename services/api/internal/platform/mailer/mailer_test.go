package mailer

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
)

// fakeSMTP accepts one message and records the envelope and data.
func fakeSMTP(t *testing.T) (string, int, chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	out := make(chan string, 1)
	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		w := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		var transcript strings.Builder
		w("220 fake ESMTP")
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			transcript.WriteString(line)
			if inData {
				if line == ".\r\n" {
					inData = false
					w("250 queued")
				}
				continue
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				w("250 fake")
			case strings.HasPrefix(cmd, "DATA"):
				inData = true
				w("354 go ahead")
			case strings.HasPrefix(cmd, "QUIT"):
				w("221 bye")
				out <- transcript.String()
				return
			default:
				w("250 ok")
			}
		}
	}()
	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port, out
}

func TestSMTPUsesBareEnvelopeAddresses(t *testing.T) {
	host, port, transcript := fakeSMTP(t)
	m := SMTP{Host: host, Port: port, From: "Hearth <no-reply@hearth.local>"}
	if err := m.Send(context.Background(), Message{To: "lou@example.org", Subject: "Ton code de réinitialisation", Body: "Code : ABCD2345"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	got := <-transcript
	for _, want := range []string{"MAIL FROM:<no-reply@hearth.local>", "RCPT TO:<lou@example.org>", "Code : ABCD2345", "=?utf-8?q?"} {
		if !strings.Contains(got, want) {
			t.Errorf("transcript missing %q:\n%s", want, got)
		}
	}
}

func TestSMTPRejectsHeaderInjection(t *testing.T) {
	m := SMTP{Host: "127.0.0.1", Port: 1, From: "no-reply@hearth.local"}
	if err := m.Send(context.Background(), Message{To: "a@b.org\r\nBcc: evil@x.org", Subject: "x"}); err == nil {
		t.Fatal("expected header injection to be rejected")
	}
}
