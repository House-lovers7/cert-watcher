package model

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempCSV はテスト用の一時CSVを作成しパスを返す。
func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "targets.csv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("一時CSVの作成に失敗: %v", err)
	}
	return path
}

// TestLoadTargets_Basic は標準的なCSVを正しく読めることを検証する。
func TestLoadTargets_Basic(t *testing.T) {
	csv := "service_name,edge_fqdn,origin_host,origin_sni,owner,vendor,notes\n" +
		"svc1,edge1.example.com,origin1.example.com,sni1.example.com,owner1,vendor1,note1\n"
	got, err := LoadTargets(writeTempCSV(t, csv))
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("件数 = %d, want 1", len(got))
	}
	w := got[0]
	if w.ServiceName != "svc1" || w.EdgeFQDN != "edge1.example.com" ||
		w.OriginHost != "origin1.example.com" || w.OriginSNI != "sni1.example.com" ||
		w.Owner != "owner1" || w.Vendor != "vendor1" || w.Notes != "note1" {
		t.Errorf("マッピング不一致: %+v", w)
	}
}

// TestLoadTargets_HeaderOrderIndependent は列順が違ってもヘッダ名で解決することを検証する。
func TestLoadTargets_HeaderOrderIndependent(t *testing.T) {
	csv := "notes,owner,vendor,origin_sni,origin_host,edge_fqdn,service_name\n" +
		"note,own,ven,sni.example.com,origin.example.com,edge.example.com,svc\n"
	got, err := LoadTargets(writeTempCSV(t, csv))
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	w := got[0]
	if w.ServiceName != "svc" || w.EdgeFQDN != "edge.example.com" || w.OriginSNI != "sni.example.com" {
		t.Errorf("列順非依存マッピングに失敗: %+v", w)
	}
}

// TestLoadTargets_SkipsEmptyRows は空行をスキップすることを検証する。
func TestLoadTargets_SkipsEmptyRows(t *testing.T) {
	csv := "service_name,edge_fqdn,origin_host,origin_sni,owner,vendor,notes\n" +
		"svc1,e1,o1,s1,ow1,v1,n1\n" +
		"\n" +
		"svc2,e2,o2,s2,ow2,v2,n2\n"
	got, err := LoadTargets(writeTempCSV(t, csv))
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("空行スキップ後の件数 = %d, want 2", len(got))
	}
}

// TestLoadTargets_MissingHeader は必須カラム欠落でエラーになることを検証する。
func TestLoadTargets_MissingHeader(t *testing.T) {
	// origin_sni を欠落させる。
	csv := "service_name,edge_fqdn,origin_host,owner,vendor,notes\n" +
		"svc1,e1,o1,ow1,v1,n1\n"
	_, err := LoadTargets(writeTempCSV(t, csv))
	if err == nil {
		t.Fatal("必須カラム欠落でエラーになるべきだが nil")
	}
}

// TestLoadTargets_NoRows は有効行ゼロでエラーになることを検証する。
func TestLoadTargets_NoRows(t *testing.T) {
	csv := "service_name,edge_fqdn,origin_host,origin_sni,owner,vendor,notes\n"
	_, err := LoadTargets(writeTempCSV(t, csv))
	if err == nil {
		t.Fatal("有効行ゼロでエラーになるべきだが nil")
	}
}

// TestLoadTargets_FileNotFound は存在しないファイルでエラーになることを検証する。
func TestLoadTargets_FileNotFound(t *testing.T) {
	_, err := LoadTargets(filepath.Join(t.TempDir(), "nope.csv"))
	if err == nil {
		t.Fatal("存在しないファイルでエラーになるべきだが nil")
	}
}
