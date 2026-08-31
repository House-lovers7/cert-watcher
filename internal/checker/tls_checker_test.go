package checker

import (
	"testing"
	"time"

	"cert-watcher/internal/model"
)

// TestParseCheckMode は --check の値解析を検証する。
func TestParseCheckMode(t *testing.T) {
	ok := map[string]CheckMode{"edge": CheckEdge, "origin": CheckOrigin, "both": CheckBoth}
	for in, want := range ok {
		got, err := ParseCheckMode(in)
		if err != nil || got != want {
			t.Errorf("ParseCheckMode(%q) = (%v, %v), want (%v, nil)", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "EDGE", "all", "foo"} {
		if _, err := ParseCheckMode(bad); err == nil {
			t.Errorf("ParseCheckMode(%q) はエラーになるべき", bad)
		}
	}
}

// TestDaysUntil は残日数計算（床関数・負値）を検証する。
func TestDaysUntil(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		until time.Time
		want  int
	}{
		{"ちょうど10日後", base.Add(10 * 24 * time.Hour), 10},
		{"10日後+12時間は床関数で10", base.Add(10*24*time.Hour + 12*time.Hour), 10},
		{"1日未満は0", base.Add(5 * time.Hour), 0},
		{"過去は負値", base.Add(-2 * 24 * time.Hour), -2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := daysUntil(base, c.until); got != c.want {
				t.Errorf("daysUntil = %d, want %d", got, c.want)
			}
		})
	}
}

// TestCheckTarget_ModeFiltersKinds は mode に応じて返る件数/種別が変わることを検証する。
// 到達不能ホストを使うため接続は ERROR になるが、件数と Kind の検証が目的。
func TestCheckTarget_ModeFiltersKinds(t *testing.T) {
	target := model.Target{
		ServiceName: "svc",
		EdgeFQDN:    "no-such-host.invalid",
		OriginHost:  "no-such-host.invalid",
		OriginSNI:   "no-such-host.invalid",
	}
	th := model.DefaultThresholds()
	timeout := 2 * time.Second

	t.Run("edge は Edge 1件", func(t *testing.T) {
		rs := CheckTarget(target, th, timeout, CheckEdge)
		if len(rs) != 1 || rs[0].Kind != "Edge" {
			t.Fatalf("edge モード結果が不正: %+v", rs)
		}
	})
	t.Run("origin は Origin 1件", func(t *testing.T) {
		rs := CheckTarget(target, th, timeout, CheckOrigin)
		if len(rs) != 1 || rs[0].Kind != "Origin" {
			t.Fatalf("origin モード結果が不正: %+v", rs)
		}
	})
	t.Run("both は 2件 (Edge, Origin)", func(t *testing.T) {
		rs := CheckTarget(target, th, timeout, CheckBoth)
		if len(rs) != 2 || rs[0].Kind != "Edge" || rs[1].Kind != "Origin" {
			t.Fatalf("both モード結果が不正: %+v", rs)
		}
	})
}

// TestCheckTarget_UnreachableIsError は到達不能ホストが ERROR として記録され、
// パニックせず結果が返ることを検証する。
func TestCheckTarget_UnreachableIsError(t *testing.T) {
	target := model.Target{
		ServiceName: "svc",
		EdgeFQDN:    "no-such-host.invalid",
		OriginHost:  "no-such-host.invalid",
		OriginSNI:   "no-such-host.invalid",
	}
	rs := CheckTarget(target, model.DefaultThresholds(), 2*time.Second, CheckEdge)
	if len(rs) != 1 {
		t.Fatalf("件数 = %d, want 1", len(rs))
	}
	if rs[0].Status != model.StatusError {
		t.Errorf("Status = %s, want ERROR", rs[0].Status)
	}
	if rs[0].Error == "" {
		t.Error("Error メッセージが空であってはならない")
	}
}

// TestCheckTarget_OriginPort は Target の OriginPort が結果に反映されることを検証する。
func TestCheckTarget_OriginPort(t *testing.T) {
	target := model.Target{
		ServiceName: "svc",
		OriginHost:  "no-such-host.invalid",
		OriginSNI:   "no-such-host.invalid",
		OriginPort:  8443,
	}
	rs := CheckTarget(target, model.DefaultThresholds(), 2*time.Second, CheckOrigin)
	if len(rs) != 1 {
		t.Fatalf("件数 = %d, want 1", len(rs))
	}
	if rs[0].Port != 8443 {
		t.Fatalf("Port = %d, want 8443", rs[0].Port)
	}
}
