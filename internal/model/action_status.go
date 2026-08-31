package model

import (
	"fmt"
	"strings"
)

// ActionStatus は人間が更新する対応管理表の1行を表す。
type ActionStatus struct {
	ServiceID    string
	Kind         string
	ActionStatus string
	ActionOwner  string
	VendorTicket string
	DueDate      string
	Comment      string
}

var requiredActionHeaders = []string{
	"service_id",
	"kind",
}

// LoadActionStatuses は action_status.csv を読み込む。
func LoadActionStatuses(path string) ([]ActionStatus, error) {
	rows, idx, err := readCSVWithHeader(path, requiredActionHeaders)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	actions := make([]ActionStatus, 0, len(rows))
	for _, row := range rows {
		kind, err := normalizeActionKind(getCSV(row, idx, "kind"))
		if err != nil {
			return nil, err
		}
		action := ActionStatus{
			ServiceID:    getCSV(row, idx, "service_id"),
			Kind:         kind,
			ActionStatus: getCSV(row, idx, "action_status"),
			ActionOwner:  getCSV(row, idx, "action_owner"),
			VendorTicket: getCSV(row, idx, "vendor_ticket"),
			DueDate:      getCSV(row, idx, "due_date"),
			Comment:      getCSV(row, idx, "comment"),
		}
		if action.ServiceID == "" {
			return nil, fmt.Errorf("action_status.csv に service_id が空の行があります")
		}
		key := actionKey(action.ServiceID, action.Kind)
		if seen[key] {
			return nil, fmt.Errorf("action_status.csv の service_id=%q kind=%q が重複しています", action.ServiceID, action.Kind)
		}
		seen[key] = true
		actions = append(actions, action)
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("action_status.csv に有効な対応管理行が1件もありません")
	}
	return actions, nil
}

// ApplyActionStatuses は service_id + kind が一致するチェック結果へ対応管理情報を反映する。
func ApplyActionStatuses(results []CertResult, actions []ActionStatus) {
	actionByKey := make(map[string]ActionStatus, len(actions))
	for _, a := range actions {
		actionByKey[actionKey(a.ServiceID, a.Kind)] = a
	}

	for i := range results {
		if results[i].ServiceID == "" {
			continue
		}
		action, ok := actionByKey[actionKey(results[i].ServiceID, results[i].Kind)]
		if !ok {
			continue
		}
		results[i].ActionStatus = action.ActionStatus
		results[i].ActionOwner = action.ActionOwner
		results[i].VendorTicket = action.VendorTicket
		results[i].DueDate = action.DueDate
		results[i].Comment = action.Comment
	}
}

func normalizeActionKind(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "edge":
		return "Edge", nil
	case "origin":
		return "Origin", nil
	default:
		return "", fmt.Errorf("action_status.csv の kind=%q が不正です（Edge / Origin のいずれか）", kind)
	}
}

func actionKey(serviceID, kind string) string {
	return strings.ToLower(strings.TrimSpace(serviceID)) + "\x00" + strings.ToLower(strings.TrimSpace(kind))
}
