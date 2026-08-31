package model

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempNamedCSV(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("一時CSVの作成に失敗: %v", err)
	}
	return path
}

func TestLoadSites_Basic(t *testing.T) {
	csv := "service_id,service_name,edge_fqdn,owner,vendor,notes,business_impact,enabled\n" +
		"corp,コーポレートサイト,www.example.com,情シス,Akamai,本番,中,true\n" +
		"old,旧サイト,old.example.com,情シス,Akamai,停止済み,低,false\n"

	got, err := LoadSites(writeTempNamedCSV(t, "sites.csv", csv))
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("有効サイト数 = %d, want 1", len(got))
	}
	if got[0].ServiceID != "corp" || got[0].BusinessImpact != "中" || !got[0].Enabled {
		t.Fatalf("sites.csv のマッピング不一致: %+v", got[0])
	}
}

func TestLoadOrigins_Basic(t *testing.T) {
	csv := "service_id,origin_host,origin_sni,origin_port,check_location,site_shield,notes_private\n" +
		"corp,origin.example.internal,origin.example.internal,8443,AWS VPC内,yes,private\n"

	got, err := LoadOrigins(writeTempNamedCSV(t, "origins.secure.csv", csv))
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Origin数 = %d, want 1", len(got))
	}
	if got[0].OriginPort != 8443 || got[0].CheckLocation != "AWS VPC内" {
		t.Fatalf("origins CSV のマッピング不一致: %+v", got[0])
	}
}

func TestMergeSiteOrigins_RequiresOrigin(t *testing.T) {
	sites := []Site{{ServiceID: "corp", ServiceName: "corp", EdgeFQDN: "www.example.com", Enabled: true}}
	_, err := MergeSiteOrigins(sites, nil, true)
	if err == nil {
		t.Fatal("Origin必須で対応行がない場合はエラーになるべき")
	}
}

func TestMergeSiteOrigins_EdgeOnlyAllowsMissingOrigin(t *testing.T) {
	sites := []Site{{ServiceID: "corp", ServiceName: "corp", EdgeFQDN: "www.example.com", Enabled: true}}
	got, err := MergeSiteOrigins(sites, nil, false)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if len(got) != 1 || got[0].OriginHost != "" {
		t.Fatalf("Edgeのみ結合結果が不正: %+v", got)
	}
}

func TestMergeSiteOrigins_Basic(t *testing.T) {
	sites := []Site{{
		ServiceID:      "corp",
		ServiceName:    "コーポレートサイト",
		EdgeFQDN:       "www.example.com",
		Owner:          "情シス",
		Vendor:         "Akamai",
		BusinessImpact: "中",
		Notes:          "本番",
		Enabled:        true,
	}}
	origins := []OriginConfig{{
		ServiceID:  "corp",
		OriginHost: "origin.example.internal",
		OriginSNI:  "origin.example.internal",
		OriginPort: 8443,
	}}

	got, err := MergeSiteOrigins(sites, origins, true)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("結合結果数 = %d, want 1", len(got))
	}
	w := got[0]
	if w.ServiceID != "corp" || w.OriginHost != "origin.example.internal" ||
		w.OriginPort != 8443 || w.Notes != "本番 / business_impact=中" {
		t.Fatalf("結合結果が不正: %+v", w)
	}
}
