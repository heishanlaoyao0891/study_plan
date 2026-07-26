package services

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"study_plan_backend/config"
	"study_plan_backend/models"
)

func createMutationBase(t *testing.T, committed bool, expiresAt time.Time) (*gorm.DB, models.PlanningPreviewVersion, PlanPreview) {
	t.Helper()
	database := planningJobsTestDB(t)
	if err := database.AutoMigrate(&models.DailyTask{}); err != nil {
		t.Fatal(err)
	}
	config.App = &config.Config{JWTSecret: "preview-mutation-test-secret"}
	preview := PlanPreview{Title: "Plan", Summary: "Summary", Tasks: []PlanPreviewTask{
		{Identity: "task-a", Date: "2026-08-01", PlannedStart: "20:00", PlannedEnd: "20:30", Title: "A", Objective: "complete task A output", EstimatedMinutes: 30, Difficulty: "easy"},
		{Identity: "task-b", Date: "2026-08-01", PlannedStart: "20:30", PlannedEnd: "21:00", Title: "B", Objective: "complete task B output", EstimatedMinutes: 30, Difficulty: "medium"},
	}}
	previewJSON, _ := json.Marshal(preview)
	inputJSON, _ := json.Marshal(PlanGenerationInput{Goal: "Plan", Days: 2, HoursPerDay: 1, StartDate: "2026-08-01", AvailableTimeSlot: "20:00-21:00"})
	base := models.PlanningPreviewVersion{PreviewID: "66666666666666666666666666666666", Version: 1, UserID: 12, Source: "local", ContextFingerprint: "fingerprint", PreviewJSON: string(previewJSON), InputJSON: string(inputJSON), ProvenanceToken: "token", ExpiresAt: expiresAt}
	if committed {
		now := time.Now().UTC()
		base.CommittedAt = &now
	}
	if err := database.Create(&base).Error; err != nil {
		t.Fatal(err)
	}
	return database, base, preview
}

func TestDerivedPreviewAddRemoveSplitAndReorder(t *testing.T) {
	database, base, preview := createMutationBase(t, false, time.Now().UTC().Add(time.Hour))
	added, err := CreateDerivedPreviewVersion(context.Background(), database, base.UserID, base.PreviewID, base.Version, PreviewMutation{Operation: "add", InsertAfterIdentity: "task-b", Task: PlanPreviewTask{Date: "2026-08-02", PlannedStart: "20:00", PlannedEnd: "20:30", Title: "C", Objective: "complete task C output", EstimatedMinutes: 30, Difficulty: "easy"}})
	if err != nil || added.Version.Version != 2 || len(added.Preview.Tasks) != 3 {
		t.Fatalf("add mutation failed: result=%+v err=%v", added, err)
	}
	for index, task := range added.Preview.Tasks {
		if task.Identity == "" || (index < len(preview.Tasks) && task.Identity == preview.Tasks[index].Identity) {
			t.Fatalf("derived version did not issue fresh identities: %+v", added.Preview.Tasks)
		}
	}
	removed, err := CreateDerivedPreviewVersion(context.Background(), database, base.UserID, base.PreviewID, added.Version.Version, PreviewMutation{Operation: "remove", TaskIdentity: added.Preview.Tasks[2].Identity})
	if err != nil || len(removed.Preview.Tasks) != 2 {
		t.Fatalf("remove mutation failed: result=%+v err=%v", removed, err)
	}
	split, err := CreateDerivedPreviewVersion(context.Background(), database, base.UserID, base.PreviewID, removed.Version.Version, PreviewMutation{Operation: "split", TaskIdentity: removed.Preview.Tasks[0].Identity, FirstPartMinutes: 15})
	if err != nil || len(split.Preview.Tasks) != 3 || split.Preview.Tasks[0].PlannedEnd != split.Preview.Tasks[1].PlannedStart {
		t.Fatalf("split mutation failed: result=%+v err=%v", split, err)
	}
	identities := []string{split.Preview.Tasks[2].Identity, split.Preview.Tasks[0].Identity, split.Preview.Tasks[1].Identity}
	reordered, err := CreateDerivedPreviewVersion(context.Background(), database, base.UserID, base.PreviewID, split.Version.Version, PreviewMutation{Operation: "reorder", OrderedIdentities: identities})
	if err != nil || reordered.Version.Version != 5 || len(reordered.Preview.Tasks) != 3 {
		t.Fatalf("reorder mutation failed: result=%+v err=%v", reordered, err)
	}
	var persistedBase models.PlanningPreviewVersion
	database.Where("preview_id = ? AND version = 1", base.PreviewID).First(&persistedBase)
	if persistedBase.PreviewJSON != base.PreviewJSON {
		t.Fatal("base preview was mutated")
	}
}

func TestDerivedPreviewRejectsCommittedAndExpiredBases(t *testing.T) {
	for _, test := range []struct {
		name      string
		committed bool
		expiresAt time.Time
	}{{"committed", true, time.Now().UTC().Add(time.Hour)}, {"expired", false, time.Now().UTC().Add(-time.Minute)}} {
		t.Run(test.name, func(t *testing.T) {
			database, base, _ := createMutationBase(t, test.committed, test.expiresAt)
			if _, err := CreateDerivedPreviewVersion(context.Background(), database, base.UserID, base.PreviewID, 1, PreviewMutation{Operation: "remove", TaskIdentity: "task-b"}); err == nil {
				t.Fatal("stale base accepted")
			}
		})
	}
}

func TestDerivedPreviewConcurrentVersionsRemainUnique(t *testing.T) {
	database, base, _ := createMutationBase(t, false, time.Now().UTC().Add(time.Hour))
	mutations := []PreviewMutation{{Operation: "remove", TaskIdentity: "task-a"}, {Operation: "remove", TaskIdentity: "task-b"}}
	errorsSeen := make([]error, len(mutations))
	var wait sync.WaitGroup
	for index := range mutations {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			_, errorsSeen[index] = CreateDerivedPreviewVersion(context.Background(), database, base.UserID, base.PreviewID, 1, mutations[index])
		}(index)
	}
	wait.Wait()
	for _, err := range errorsSeen {
		if err != nil {
			t.Fatalf("concurrent derived version failed: %v", err)
		}
	}
	var versions []models.PlanningPreviewVersion
	database.Where("preview_id = ?", base.PreviewID).Order("version ASC").Find(&versions)
	if len(versions) != 3 || versions[1].Version == versions[2].Version {
		t.Fatalf("concurrent versions were not unique: %+v", versions)
	}
}
