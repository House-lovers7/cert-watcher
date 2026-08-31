package model

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Site は非エンジニア向けのサイト一覧CSVの1行を表す。
type Site struct {
	ServiceID      string
	ServiceName    string
	EdgeFQDN       string
	Owner          string
	Vendor         string
	BusinessImpact string
	Notes          string
	Enabled        bool
}

// OriginConfig は管理者向けOrigin情報CSVの1行を表す。
type OriginConfig struct {
	ServiceID      string
	OriginHost     string
	OriginSNI      string
	OriginPort     int
	CheckLocation  string
	SiteShield     string
	EnteredBy      string
	Source         string
	LastReviewedAt string
	NotesPrivate   string
}

var requiredSiteHeaders = []string{
	"service_id",
	"service_name",
	"edge_fqdn",
	"owner",
	"vendor",
	"notes",
}

var requiredOriginHeaders = []string{
	"service_id",
	"origin_host",
	"origin_sni",
}

// LoadSites は非エンジニア向け sites.csv を読み込む。
// enabled カラムが存在しない場合は true として扱う。
func LoadSites(path string) ([]Site, error) {
	rows, idx, err := readCSVWithHeader(path, requiredSiteHeaders)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	sites := make([]Site, 0, len(rows))
	for _, row := range rows {
		site := Site{
			ServiceID:      getCSV(row, idx, "service_id"),
			ServiceName:    getCSV(row, idx, "service_name"),
			EdgeFQDN:       getCSV(row, idx, "edge_fqdn"),
			Owner:          getCSV(row, idx, "owner"),
			Vendor:         getCSV(row, idx, "vendor"),
			BusinessImpact: getCSV(row, idx, "business_impact"),
			Notes:          getCSV(row, idx, "notes"),
			Enabled:        true,
		}
		if hasCSV(idx, "enabled") {
			enabled, err := parseEnabled(getCSV(row, idx, "enabled"))
			if err != nil {
				return nil, fmt.Errorf("sites.csv service_id=%q の enabled が不正です: %w", site.ServiceID, err)
			}
			site.Enabled = enabled
		}
		if site.ServiceID == "" {
			return nil, fmt.Errorf("sites.csv に service_id が空の行があります")
		}
		if site.ServiceName == "" {
			return nil, fmt.Errorf("sites.csv service_id=%q の service_name が空です", site.ServiceID)
		}
		if site.EdgeFQDN == "" {
			return nil, fmt.Errorf("sites.csv service_id=%q の edge_fqdn が空です", site.ServiceID)
		}
		if seen[site.ServiceID] {
			return nil, fmt.Errorf("sites.csv の service_id %q が重複しています", site.ServiceID)
		}
		seen[site.ServiceID] = true
		if site.Enabled {
			sites = append(sites, site)
		}
	}

	if len(sites) == 0 {
		return nil, fmt.Errorf("sites.csv に有効な監視対象が1件もありません")
	}
	return sites, nil
}

// LoadOrigins は管理者向け origins.secure.csv を読み込む。
// origin_port カラムが存在しない、または空の場合は 443 として扱う。
func LoadOrigins(path string) ([]OriginConfig, error) {
	rows, idx, err := readCSVWithHeader(path, requiredOriginHeaders)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	origins := make([]OriginConfig, 0, len(rows))
	for _, row := range rows {
		origin := OriginConfig{
			ServiceID:      getCSV(row, idx, "service_id"),
			OriginHost:     getCSV(row, idx, "origin_host"),
			OriginSNI:      getCSV(row, idx, "origin_sni"),
			OriginPort:     defaultHTTPSPort,
			CheckLocation:  getCSV(row, idx, "check_location"),
			SiteShield:     getCSV(row, idx, "site_shield"),
			EnteredBy:      getCSV(row, idx, "entered_by"),
			Source:         getCSV(row, idx, "source"),
			LastReviewedAt: getCSV(row, idx, "last_reviewed_at"),
			NotesPrivate:   getCSV(row, idx, "notes_private"),
		}
		if hasCSV(idx, "origin_port") && getCSV(row, idx, "origin_port") != "" {
			port, err := parsePort(getCSV(row, idx, "origin_port"))
			if err != nil {
				return nil, fmt.Errorf("origins CSV service_id=%q の origin_port が不正です: %w", origin.ServiceID, err)
			}
			origin.OriginPort = port
		}
		if origin.ServiceID == "" {
			return nil, fmt.Errorf("origins CSV に service_id が空の行があります")
		}
		if origin.OriginHost == "" {
			return nil, fmt.Errorf("origins CSV service_id=%q の origin_host が空です", origin.ServiceID)
		}
		if origin.OriginSNI == "" {
			return nil, fmt.Errorf("origins CSV service_id=%q の origin_sni が空です", origin.ServiceID)
		}
		if seen[origin.ServiceID] {
			return nil, fmt.Errorf("origins CSV の service_id %q が重複しています", origin.ServiceID)
		}
		seen[origin.ServiceID] = true
		origins = append(origins, origin)
	}

	if len(origins) == 0 {
		return nil, fmt.Errorf("origins CSV に有効なOrigin情報が1件もありません")
	}
	return origins, nil
}

// MergeSiteOrigins は sites.csv と origins.secure.csv を service_id で結合し、
// 既存チェックエンジンが扱う Target に変換する。
func MergeSiteOrigins(sites []Site, origins []OriginConfig, requireOrigin bool) ([]Target, error) {
	originByID := make(map[string]OriginConfig, len(origins))
	for _, o := range origins {
		if _, exists := originByID[o.ServiceID]; exists {
			return nil, fmt.Errorf("origins CSV の service_id %q が重複しています", o.ServiceID)
		}
		originByID[o.ServiceID] = o
	}

	targets := make([]Target, 0, len(sites))
	for _, s := range sites {
		o, ok := originByID[s.ServiceID]
		if requireOrigin && !ok {
			return nil, fmt.Errorf("sites.csv service_id=%q に対応するOrigin情報がありません", s.ServiceID)
		}

		notes := s.Notes
		if s.BusinessImpact != "" {
			notes = appendNote(notes, "business_impact="+s.BusinessImpact)
		}

		t := Target{
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			EdgeFQDN:    s.EdgeFQDN,
			Owner:       s.Owner,
			Vendor:      s.Vendor,
			Notes:       notes,
		}
		if ok {
			t.OriginHost = o.OriginHost
			t.OriginSNI = o.OriginSNI
			t.OriginPort = o.OriginPort
		}
		targets = append(targets, t)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("結合後の監視対象が1件もありません")
	}
	return targets, nil
}

const defaultHTTPSPort = 443

func readCSVWithHeader(path string, required []string) ([][]string, map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("CSVを開けません (%s): %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("ヘッダ行を読み込めません (%s): %w", path, err)
	}
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, want := range required {
		if _, ok := idx[want]; !ok {
			return nil, nil, fmt.Errorf("必須カラム %q がCSVヘッダに見つかりません（必要なヘッダ: %s）",
				want, strings.Join(required, ", "))
		}
	}

	var rows [][]string
	line := 1
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			return nil, nil, fmt.Errorf("CSV %d行目の読み込みに失敗: %w", line, err)
		}
		if isEmptyRow(row) {
			continue
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("CSVに有効な行が1件もありません")
	}
	return rows, idx, nil
}

func getCSV(row []string, idx map[string]int, col string) string {
	i, ok := idx[col]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func hasCSV(idx map[string]int, col string) bool {
	_, ok := idx[col]
	return ok
}

func parseEnabled(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "true", "yes", "y", "1", "on":
		return true, nil
	case "false", "no", "n", "0", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%q は true/false/yes/no/1/0 のいずれかで指定してください", s)
	}
}

func parsePort(s string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, err
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("ポート番号は1から65535で指定してください")
	}
	return port, nil
}

func appendNote(base, add string) string {
	if base == "" {
		return add
	}
	return base + " / " + add
}
