package model

import "time"

// Status は証明書の有効期限ステータスを表す。
type Status string

// ステータス定数。残日数およびエラー有無で判定される。
const (
	StatusOK       Status = "OK"       // 31日以上
	StatusWarn     Status = "WARN"     // 30日未満
	StatusCritical Status = "CRITICAL" // 14日未満
	StatusUrgent   Status = "URGENT"   // 7日未満
	StatusExpired  Status = "EXPIRED"  // 期限切れ
	StatusError    Status = "ERROR"    // 接続不可・証明書取得失敗
)

// Thresholds はステータス判定のしきい値（残日数）を保持する。
// CLIフラグで上書き可能。
type Thresholds struct {
	WarnDays     int // この日数未満で WARN（既定 30）
	CriticalDays int // この日数未満で CRITICAL（既定 14）
	UrgentDays   int // この日数未満で URGENT（既定 7）
}

// DefaultThresholds は既定のしきい値を返す。
func DefaultThresholds() Thresholds {
	return Thresholds{WarnDays: 30, CriticalDays: 14, UrgentDays: 7}
}

// CertResult は1つの証明書チェック結果（Edge または Origin）を表す。
type CertResult struct {
	ServiceID   string    // サービスID（分離CSV利用時の紐付けキー）
	ServiceName string    // サービス名
	Kind        string    // "Edge" または "Origin"
	Host        string    // 接続先ホスト
	Port        int       // 接続先ポート（通常 443）
	SNI         string    // 接続時に提示した SNI
	Subject     string    // 証明書の Subject
	Issuer      string    // 証明書の Issuer
	DNSNames    []string  // SAN（DNS Names）
	NotBefore   time.Time // 有効期間開始
	NotAfter    time.Time // 有効期間終了
	DaysLeft    int       // 残日数（NotAfter まで）。エラー時は意味を持たない
	Status      Status    // 判定ステータス
	Error       string    // エラー内容（正常時は空）

	// 監視対象由来のメタ情報（レポートに転記）
	Owner  string
	Vendor string
	Notes  string

	// 人間の対応管理情報（任意）
	ActionStatus string
	ActionOwner  string
	VendorTicket string
	DueDate      string
	Comment      string
}

// DetermineStatus は残日数・期限切れ・エラー有無からステータスを判定する。
// 優先順位: ERROR → EXPIRED → URGENT → CRITICAL → WARN → OK。
//
// 境界（既定値の場合）:
//   - URGENT  : 残日数 < 7   （7日未満）
//   - CRITICAL: 残日数 < 14  （14日未満）
//   - WARN    : 残日数 <= 30 （= warn-days 以下）
//   - OK      : 残日数 >= 31 （= warn-days より大きい / 「31日以上」）
//
// URGENT/CRITICAL は仕様「7日未満 / 14日未満」に合わせ「未満（<）」判定。
// OK は仕様「31日以上」に合わせ、warn-days(30) ちょうどは WARN とする（OK は warn-days より大きい場合のみ）。
func DetermineStatus(daysLeft int, expired bool, hasError bool, th Thresholds) Status {
	switch {
	case hasError:
		return StatusError
	case expired:
		return StatusExpired
	case daysLeft < th.UrgentDays:
		return StatusUrgent
	case daysLeft < th.CriticalDays:
		return StatusCritical
	case daysLeft <= th.WarnDays:
		return StatusWarn
	default:
		return StatusOK
	}
}

// StatusRank はソート用の重みを返す（小さいほど上位＝優先表示）。
// ERROR / EXPIRED / URGENT / CRITICAL を最上位グループとして上に集める。
func StatusRank(s Status) int {
	switch s {
	case StatusError:
		return 0
	case StatusExpired:
		return 1
	case StatusUrgent:
		return 2
	case StatusCritical:
		return 3
	case StatusWarn:
		return 4
	case StatusOK:
		return 5
	default:
		return 6
	}
}

// IsAlert は「対応が必要」とみなすステータスかどうかを返す。
// 終了コードの非0判定（EXPIRED / URGENT / CRITICAL / ERROR）に使う。
func (s Status) IsAlert() bool {
	switch s {
	case StatusExpired, StatusUrgent, StatusCritical, StatusError:
		return true
	default:
		return false
	}
}
