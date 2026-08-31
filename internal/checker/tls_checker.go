// Package checker は TLS 接続によるサーバ証明書の取得と
// 監視対象（Edge / Origin）ごとのチェックを担う。
package checker

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"cert-watcher/internal/model"
)

// defaultPort は HTTPS の標準ポート。
const defaultPort = 443

// CheckMode はどの系統の証明書をチェックするかを表す。
// Origin チェックは社内NW / AWS内でのみ到達可能なことがあるため、
// 実行環境ごとに Edge / Origin を分けて実行できるようにする。
type CheckMode string

const (
	CheckEdge   CheckMode = "edge"   // Edge 証明書のみ
	CheckOrigin CheckMode = "origin" // Origin 証明書のみ
	CheckBoth   CheckMode = "both"   // 両方（既定）
)

// ParseCheckMode は文字列を CheckMode に変換する。不正値はエラー。
func ParseCheckMode(s string) (CheckMode, error) {
	switch CheckMode(s) {
	case CheckEdge:
		return CheckEdge, nil
	case CheckOrigin:
		return CheckOrigin, nil
	case CheckBoth:
		return CheckBoth, nil
	default:
		return "", fmt.Errorf("不正な --check 値 %q（edge / origin / both のいずれか）", s)
	}
}

// CheckCert は host:port へ TLS 接続し、提示された SNI でハンドシェイクして
// サーバのリーフ証明書を返す。
//
// InsecureSkipVerify=true としているのは意図的:
// 期限切れ・自己署名・名前不一致でも「証明書情報そのもの」を取得したいため。
// 有効期限の判定は取得した証明書をもとに自前で行う。
func CheckCert(host, sni string, port int, timeout time.Duration) (*x509.Certificate, error) {
	if host == "" {
		return nil, fmt.Errorf("接続先ホストが空です")
	}
	if port == 0 {
		port = defaultPort
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))

	dialer := &net.Dialer{Timeout: timeout}
	conf := &tls.Config{
		ServerName:         sni,  // SNI を明示（Edge/Origin で異なる）
		InsecureSkipVerify: true, // 検証エラーで取得失敗にしない（理由は上記）
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, conf)
	if err != nil {
		return nil, fmt.Errorf("TLS接続に失敗 (%s, SNI=%s): %w", address, sni, err)
	}
	defer conn.Close()

	// ハンドシェイク全体に対する保険のデッドライン。
	_ = conn.SetDeadline(time.Now().Add(timeout))

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("サーバ証明書を取得できませんでした (%s)", address)
	}
	// PeerCertificates[0] がリーフ（サーバ）証明書。
	return state.PeerCertificates[0], nil
}

// CheckTarget は1つの監視対象について、mode に応じた系統の証明書をチェックする。
// mode=both なら Edge/Origin の2件、edge/origin ならそれぞれ1件を返す。
// 接続失敗時はその対象を ERROR として記録し、例外で全体を止めない。
func CheckTarget(t model.Target, th model.Thresholds, timeout time.Duration, mode CheckMode) []model.CertResult {
	var results []model.CertResult

	if mode == CheckEdge || mode == CheckBoth {
		results = append(results, checkOne(checkSpec{
			serviceName: t.ServiceName,
			kind:        "Edge",
			host:        t.EdgeFQDN,
			sni:         t.EdgeFQDN, // Edge は FQDN をそのまま SNI に使う
			port:        defaultPort,
			target:      t,
		}, th, timeout))
	}

	if mode == CheckOrigin || mode == CheckBoth {
		results = append(results, checkOne(checkSpec{
			serviceName: t.ServiceName,
			kind:        "Origin",
			host:        t.OriginHost,
			sni:         t.OriginSNI,
			port:        t.OriginPort,
			target:      t,
		}, th, timeout))
	}

	return results
}

// checkSpec は checkOne への入力をまとめた内部構造体。
type checkSpec struct {
	serviceName string
	kind        string
	host        string
	sni         string
	port        int
	target      model.Target
}

// checkOne は1接続分のチェックを行い CertResult を生成する。
func checkOne(spec checkSpec, th model.Thresholds, timeout time.Duration) model.CertResult {
	res := model.CertResult{
		ServiceID:   spec.target.ServiceID,
		ServiceName: spec.serviceName,
		Kind:        spec.kind,
		Host:        spec.host,
		Port:        spec.port,
		SNI:         spec.sni,
		Owner:       spec.target.Owner,
		Vendor:      spec.target.Vendor,
		Notes:       spec.target.Notes,
	}
	if res.Port == 0 {
		res.Port = defaultPort
	}

	cert, err := CheckCert(spec.host, spec.sni, res.Port, timeout)
	if err != nil {
		// 接続失敗・取得失敗は ERROR として記録（全体は継続）。
		res.Status = model.StatusError
		res.Error = err.Error()
		return res
	}

	// 証明書情報を詰める。
	res.Subject = cert.Subject.String()
	res.Issuer = cert.Issuer.String()
	res.DNSNames = cert.DNSNames
	res.NotBefore = cert.NotBefore
	res.NotAfter = cert.NotAfter

	// 残日数を算出（切り上げではなく、現在時刻から NotAfter までの日数を床関数で）。
	now := time.Now()
	expired := now.After(cert.NotAfter)
	res.DaysLeft = daysUntil(now, cert.NotAfter)
	res.Status = model.DetermineStatus(res.DaysLeft, expired, false, th)

	return res
}

// daysUntil は from から until までの残日数を返す。
// 期限を過ぎている場合は負の値になる。
func daysUntil(from, until time.Time) int {
	d := until.Sub(from).Hours() / 24
	return int(math.Floor(d))
}

// FormatDNSNames は SAN を CSV/HTML 用に1文字列へ連結する（区切りは ";"）。
func FormatDNSNames(names []string) string {
	return strings.Join(names, ";")
}
