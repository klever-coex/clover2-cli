package agent

import (
	"strings"
	"testing"
)

func TestRenderUnit(t *testing.T) {
	unit := renderUnit("pi")

	for _, want := range []string{
		"User=pi",
		"ExecStart=/usr/bin/clover2 agent run --listen 0.0.0.0:7550 --token-file /etc/clover2/agent.token",
		"Restart=on-failure",
		"KillMode=mixed",
		"WantedBy=multi-user.target",
		"After=network-online.target",
	} {
		if !strings.Contains(unit, want) {
			t.Errorf("unit missing %q:\n%s", want, unit)
		}
	}
}

func TestParseFirstLoginUser(t *testing.T) {
	passwd := "root:x:0:0:root:/root:/bin/bash\n" +
		"daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin\n" +
		"systemd-network:x:998:998:systemd Network Management:/:/usr/sbin/nologin\n" +
		"pi:x:1000:1000:,,,:/home/pi:/bin/bash\n" +
		"dev:x:1001:1001:,,,:/home/dev:/bin/bash\n" +
		"nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin\n"

	name, err := parseFirstLoginUser(passwd)
	if err != nil {
		t.Fatal(err)
	}
	if name != "pi" {
		t.Errorf("first login user = %q, want pi", name)
	}

	if _, err := parseFirstLoginUser("root:x:0:0:root:/root:/bin/bash\n"); err == nil {
		t.Error("no regular user must error, got nil")
	}
}
