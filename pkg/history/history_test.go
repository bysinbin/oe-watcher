package history

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"medya-filter/pkg/models"
)

func TestHistoryManager(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// 1. Initial state
	if len(mgr.GetAllItems()) != 0 {
		t.Fatalf("expected 0 items initially, got %d", len(mgr.GetAllItems()))
	}

	// 2. Record items
	item1 := models.HistoryItem{
		Action: "auto_skipped",
		Track: models.MediaInfo{
			App:    "Spotify",
			Artist: "Ahmet Kaya",
			Title:  "Kum Gibi",
		},
		Person: models.Person{
			DisplayName: "Ahmet Kaya",
			Tags:        []string{"Şarkıcı"},
		},
	}
	mgr.Record(item1)

	item2 := models.HistoryItem{
		Action: "manual_skip",
		Track: models.MediaInfo{
			App:    "YouTube Music",
			Artist: "Grup Yorum",
			Title:  "Büyü",
		},
		Person: models.Person{
			DisplayName: "Grup Yorum",
			Tags:        []string{"Müzik Grubu"},
		},
	}
	mgr.Record(item2)

	// Verify items count
	items := mgr.GetAllItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Verify stats
	stats := mgr.GetStats()
	if stats.TotalDetections != 2 {
		t.Errorf("expected 2 detections, got %d", stats.TotalDetections)
	}
	if stats.TotalAutoSkips != 1 {
		t.Errorf("expected 1 auto skip, got %d", stats.TotalAutoSkips)
	}
	if stats.TotalManualSkips != 1 {
		t.Errorf("expected 1 manual skip, got %d", stats.TotalManualSkips)
	}
	if stats.TopApps["Spotify"] != 1 || stats.TopApps["YouTube Music"] != 1 {
		t.Errorf("app counts mismatch: %+v", stats.TopApps)
	}

	// 3. Test filtering & search
	res, total := mgr.GetItems(10, 0, "auto_skipped", "")
	if total != 1 || len(res) != 1 || res[0].Person.DisplayName != "Ahmet Kaya" {
		t.Errorf("filtering by auto_skipped failed: total=%d, len=%d", total, len(res))
	}

	resSearch, totalSearch := mgr.GetItems(10, 0, "", "Yorum")
	if totalSearch != 1 || len(resSearch) != 1 || resSearch[0].Person.DisplayName != "Grup Yorum" {
		t.Errorf("searching 'Yorum' failed: total=%d", totalSearch)
	}

	// 4. Test CSV export
	csvBytes, err := mgr.ExportCSV()
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}
	// Check UTF-8 BOM
	if !bytes.HasPrefix(csvBytes, []byte{0xEF, 0xBB, 0xBF}) {
		t.Error("CSV missing UTF-8 BOM")
	}
	csvStr := string(csvBytes)
	if !strings.Contains(csvStr, "Ahmet Kaya") || !strings.Contains(csvStr, "Kum Gibi") {
		t.Errorf("CSV missing item content: %s", csvStr)
	}

	// 5. Test JSON export
	jsonBytes, err := mgr.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}
	if !strings.Contains(string(jsonBytes), "total_auto_skips") {
		t.Errorf("JSON export missing stats: %s", string(jsonBytes))
	}

	// 6. Test Persistence across reload
	// Force save to ensure disk write
	if err := mgr.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check file exists on disk
	historyPath := filepath.Join(tempDir, HistoryFileName)
	if _, err := os.Stat(historyPath); err != nil {
		t.Fatalf("history.json was not created on disk: %v", err)
	}

	// Reload with a new manager instance
	reloadedMgr := NewManager(tempDir)
	reloadedItems := reloadedMgr.GetAllItems()
	if len(reloadedItems) != 2 {
		t.Fatalf("expected 2 items after reload, got %d", len(reloadedItems))
	}
	if reloadedItems[0].Person.DisplayName != "Grup Yorum" && reloadedItems[1].Person.DisplayName != "Grup Yorum" {
		t.Errorf("reloaded items do not match expected data")
	}

	// 7. Test Clear
	if err := reloadedMgr.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if len(reloadedMgr.GetAllItems()) != 0 {
		t.Errorf("expected 0 items after Clear, got %d", len(reloadedMgr.GetAllItems()))
	}
}
