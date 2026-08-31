package model

import (
	"sort"
	"testing"
)

// TestDetermineStatus は残日数・期限切れ・エラーからのステータス判定の境界を検証する。
func TestDetermineStatus(t *testing.T) {
	th := DefaultThresholds() // warn=30, critical=14, urgent=7

	cases := []struct {
		name     string
		days     int
		expired  bool
		hasError bool
		want     Status
	}{
		{"error優先", 100, false, true, StatusError},
		{"expired優先(エラー無し)", -1, true, false, StatusExpired},
		{"error は expired より優先", 0, true, true, StatusError},
		{"urgent 境界未満(6)", 6, false, false, StatusUrgent},
		{"urgent 境界(7)はcritical", 7, false, false, StatusCritical},
		{"critical 境界未満(13)", 13, false, false, StatusCritical},
		{"critical 境界(14)はwarn", 14, false, false, StatusWarn},
		{"warn 境界未満(29)", 29, false, false, StatusWarn},
		{"warn 境界(30)はwarn", 30, false, false, StatusWarn},
		{"ok 境界(31)", 31, false, false, StatusOK},
		{"ok 十分先", 365, false, false, StatusOK},
		{"残0日はurgent", 0, false, false, StatusUrgent},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DetermineStatus(c.days, c.expired, c.hasError, th)
			if got != c.want {
				t.Errorf("DetermineStatus(days=%d, expired=%v, err=%v) = %s, want %s",
					c.days, c.expired, c.hasError, got, c.want)
			}
		})
	}
}

// TestStatusRankOrder はソート重みが ERROR を最上位グループに置くことを検証する。
func TestStatusRankOrder(t *testing.T) {
	// 入力をシャッフル状態で与え、ランク昇順に並べる。
	in := []Status{StatusOK, StatusWarn, StatusCritical, StatusUrgent, StatusExpired, StatusError}
	want := []Status{StatusError, StatusExpired, StatusUrgent, StatusCritical, StatusWarn, StatusOK}

	got := make([]Status, len(in))
	copy(got, in)
	sort.SliceStable(got, func(i, j int) bool {
		return StatusRank(got[i]) < StatusRank(got[j])
	})

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ソート順が不正: got=%v want=%v", got, want)
		}
	}
}

// TestIsAlert はアラート対象ステータスの判定を検証する。
func TestIsAlert(t *testing.T) {
	alert := []Status{StatusExpired, StatusUrgent, StatusCritical, StatusError}
	notAlert := []Status{StatusWarn, StatusOK}

	for _, s := range alert {
		if !s.IsAlert() {
			t.Errorf("%s は IsAlert=true であるべき", s)
		}
	}
	for _, s := range notAlert {
		if s.IsAlert() {
			t.Errorf("%s は IsAlert=false であるべき", s)
		}
	}
}
