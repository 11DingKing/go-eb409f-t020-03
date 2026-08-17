package dispatch

import (
	"testing"
	"time"

	"microgrid-ops/internal/workorder"
)

// TestAnomalyDeadlineMonitorTiming covers the observable behaviour of the
// background deadline monitor: an inspection with a detected anomaly must only
// be reported as overdue once the reporting window has actually elapsed.
func TestAnomalyDeadlineMonitorTiming(t *testing.T) {
	orch := newTestOrchestrator() // 10-minute anomaly reporting window
	registerCabin(orch, "C-30")

	wo, err := orch.CreateInspectionOrder("C-30", "insp-1", workorder.PriorityNormal)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	wo, err = orch.DetectAnomaly(wo.ID, 50.0, false, true)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	detectedAt := wo.Anomaly.DetectedAt

	// Still well inside the reporting window: nothing is overdue yet.
	for _, elapsed := range []time.Duration{0, time.Second, 2 * time.Minute, 9 * time.Minute} {
		if expired := orch.CheckAnomalyDeadlines(detectedAt.Add(elapsed)); len(expired) != 0 {
			t.Fatalf("orders %v reported overdue only %s after detection, still inside the reporting window", expired, elapsed)
		}
	}
	for _, n := range orch.Notifications() {
		if n.Type == NotifAnomalyDeadline {
			t.Fatalf("deadline-expired notification raised inside the reporting window: %s", n.Message)
		}
	}

	// Past the window with no report submitted: the order must be reported.
	expired := orch.CheckAnomalyDeadlines(detectedAt.Add(11 * time.Minute))
	if len(expired) != 1 || expired[0] != wo.ID {
		t.Fatalf("expired = %v, want [%s]", expired, wo.ID)
	}
	found := false
	for _, n := range orch.Notifications() {
		if n.Type == NotifAnomalyDeadline && n.WorkOrderID == wo.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a deadline-expired notification after the reporting window elapsed")
	}
}
