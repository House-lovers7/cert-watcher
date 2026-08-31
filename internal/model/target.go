// Package model は cert-watcher で扱うデータ構造（監視対象・チェック結果）を定義する。
package model

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// Target は監視対象1サービス分の設定を表す。
// 入力CSVの1行に対応する。
type Target struct {
	ServiceID   string // サービスID（分離CSV利用時の紐付けキー）
	ServiceName string // サービス名（識別用ラベル）
	EdgeFQDN    string // Akamai Edge 側 FQDN（利用者→Edge の接続先 / SNI）
	OriginHost  string // Origin の接続先ホスト（Akamai→Origin の接続先）
	OriginSNI   string // Origin 接続時に提示する SNI
	OriginPort  int    // Origin 接続ポート（0 の場合は 443）
	Owner       string // 管理担当者
	Vendor      string // ベンダー
	Notes       string // 備考
}

// expectedHeaders は入力CSVに必須のカラム名。
// 列の並び順に依存しないよう、ヘッダ名でインデックスを解決する。
var expectedHeaders = []string{
	"service_name",
	"edge_fqdn",
	"origin_host",
	"origin_sni",
	"owner",
	"vendor",
	"notes",
}

// LoadTargets は指定パスのCSVを読み込み Target スライスを返す。
// ヘッダ行を検証し、列順に依存せずヘッダ名でマッピングする。
// 空行はスキップする。
func LoadTargets(path string) ([]Target, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("入力CSVを開けません (%s): %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // 列数のばらつきを許容（末尾の空欄など）
	r.TrimLeadingSpace = true

	// 1行目: ヘッダ
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("ヘッダ行を読み込めません: %w", err)
	}

	// ヘッダ名 → 列インデックスのマップを構築（小文字・前後空白除去で正規化）
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// 必須ヘッダの存在チェック
	for _, want := range expectedHeaders {
		if _, ok := idx[want]; !ok {
			return nil, fmt.Errorf("必須カラム %q がCSVヘッダに見つかりません（必要なヘッダ: %s）",
				want, strings.Join(expectedHeaders, ", "))
		}
	}

	// 行から指定カラムの値を安全に取り出すヘルパー。
	get := func(row []string, col string) string {
		i := idx[col]
		if i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	var targets []Target
	line := 1 // ヘッダ分
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			return nil, fmt.Errorf("CSV %d行目の読み込みに失敗: %w", line, err)
		}

		// 完全な空行（全カラム空）はスキップ
		if isEmptyRow(row) {
			continue
		}

		t := Target{
			ServiceName: get(row, "service_name"),
			EdgeFQDN:    get(row, "edge_fqdn"),
			OriginHost:  get(row, "origin_host"),
			OriginSNI:   get(row, "origin_sni"),
			Owner:       get(row, "owner"),
			Vendor:      get(row, "vendor"),
			Notes:       get(row, "notes"),
		}
		targets = append(targets, t)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("CSVに有効な監視対象が1件もありません")
	}
	return targets, nil
}

// isEmptyRow は全フィールドが空白の行かどうかを判定する。
func isEmptyRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
