package models

import "testing"

func ptr(s string) *string { return &s }

func TestTrainingLevelRank(t *testing.T) {
	cases := []struct {
		level *string
		want  int
	}{
		{ptr("INICIANTE"), 1},
		{ptr("iniciante"), 1},
		{ptr("  Iniciante  "), 1},
		{ptr("INTERMEDIÁRIO"), 2},
		{ptr("intermediário"), 2},
		{ptr("Intermediario"), 2},
		{ptr("AVANÇADO"), 3},
		{ptr("avançado"), 3},
		{ptr("Avancado"), 3},
		{ptr("advanced"), 3},
		{ptr("1"), 1},
		{ptr("2"), 2},
		{ptr("3"), 3},
		{ptr("outro"), 99},
		{nil, 99},
		{ptr(""), 99},
	}
	for _, tc := range cases {
		if got := trainingLevelRank(tc.level); got != tc.want {
			t.Errorf("trainingLevelRank(%v) = %d, want %d", tc.level, got, tc.want)
		}
	}
}

func TestFormatTrainingLevelDisplay_plainDB(t *testing.T) {
	// Banco sem acentos → JSON com acentuação correta
	cases := []struct {
		db   string
		want string
	}{
		{"iniciante", levelDisplayIniciante},
		{"INICIANTE", levelDisplayIniciante},
		{"intermediario", levelDisplayIntermediario},
		{"intermediário", levelDisplayIntermediario},
		{"avancado", levelDisplayAvancado},
		{"avançado", levelDisplayAvancado},
		{"1", levelDisplayIniciante},
		{"2", levelDisplayIntermediario},
		{"3", levelDisplayAvancado},
		{"outro_nivel", "outro_nivel"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := formatTrainingLevelDisplay(tc.db); got != tc.want {
			t.Errorf("formatTrainingLevelDisplay(%q) = %q, want %q", tc.db, got, tc.want)
		}
	}
}
