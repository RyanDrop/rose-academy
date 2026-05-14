package models_test

import (
	"testing"
	"time"

	"rose-academy/models"
)

func TestElapsedCalendarMonths(t *testing.T) {
	anchor := time.Date(2026, time.May, 10, 12, 0, 0, 0, time.UTC)
	if got := models.ElapsedCalendarMonths(anchor, time.Date(2026, time.May, 28, 0, 0, 0, 0, time.UTC)); got != 0 {
		t.Fatalf("same calendar month: got %d want 0", got)
	}
	if got := models.ElapsedCalendarMonths(anchor, time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)); got != 1 {
		t.Fatalf("next month: got %d want 1", got)
	}
	if got := models.ElapsedCalendarMonths(anchor, time.Date(2027, time.May, 1, 0, 0, 0, 0, time.UTC)); got != 12 {
		t.Fatalf("12 months later: got %d want 12", got)
	}
}

func TestTrainingMonthActiveForPlan(t *testing.T) {
	anchor := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	ref := time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC)
	if !models.TrainingMonthActiveForPlan(1, anchor, ref) {
		t.Fatal("month 1 should be active in first calendar month of plan")
	}
	if models.TrainingMonthActiveForPlan(2, anchor, ref) {
		t.Fatal("month 2 should not be active yet")
	}
	ref = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	if !models.TrainingMonthActiveForPlan(2, anchor, ref) {
		t.Fatal("month 2 should be active after one calendar month from plan anchor")
	}
	if models.TrainingMonthActiveForPlan(13, anchor, ref) {
		t.Fatal("month 13 must never be active (fora dos 12 meses do plano)")
	}
	ref = time.Date(2027, time.April, 1, 0, 0, 0, 0, time.UTC) // 11 meses após maio = abril seguinte
	if !models.TrainingMonthActiveForPlan(12, anchor, ref) {
		t.Fatal("month 12 should unlock after 11 calendar months from anchor")
	}
}
