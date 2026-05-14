package models

import "strings"

// Textos canónicos devolvidos ao front (PT-BR com acentuação); o banco pode guardar sem acentos.
const (
	levelDisplayIniciante     = "Iniciante"
	levelDisplayIntermediario = "Intermediário"
	levelDisplayAvancado      = "Avançado"
)

// formatTrainingLevelDisplay mapeia o valor gravado no banco para a forma correta no JSON (acentos e ç).
// Valores reconhecidos como iniciante/intermediário/avançado (com ou sem acento no banco) viram as constantes acima;
// níveis desconhecidos são retornados como estão (após trim).
func formatTrainingLevelDisplay(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	switch trainingLevelRank(&raw) {
	case 1:
		return levelDisplayIniciante
	case 2:
		return levelDisplayIntermediario
	case 3:
		return levelDisplayAvancado
	default:
		return raw
	}
}

// normalizeLevelForOrder remove acentos e cedilha para comparar níveis (PT-BR).
func normalizeLevelForOrder(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	r := strings.NewReplacer(
		"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a",
		"è", "e", "é", "e", "ê", "e", "ë", "e",
		"ì", "i", "í", "i", "î", "i", "ï", "i",
		"ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o",
		"ù", "u", "ú", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
	)
	return r.Replace(s)
}

// trainingLevelRank ordem fixa: 1 iniciante, 2 intermediário, 3 avançado; desconhecidos por último (99).
func trainingLevelRank(level *string) int {
	var raw string
	if level != nil {
		raw = *level
	}
	n := normalizeLevelForOrder(raw)
	if n == "" {
		return 99
	}
	switch n {
	case "1", "01":
		return 1
	case "2", "02":
		return 2
	case "3", "03":
		return 3
	}
	if strings.Contains(n, "iniciante") {
		return 1
	}
	if strings.Contains(n, "intermediario") {
		return 2
	}
	if strings.Contains(n, "avancado") {
		return 3
	}
	if strings.Contains(n, "advanced") {
		return 3
	}
	return 99
}

// ToTrainingLevelDBEnum converte o nível enviado pelo cliente para o literal do enum PostgreSQL training_level_type.
func ToTrainingLevelDBEnum(raw string) (enum string, ok bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}
	switch trainingLevelRank(&s) {
	case 1:
		return "iniciante", true
	case 2:
		return "intermediario", true
	case 3:
		return "avancado", true
	default:
		return "", false
	}
}
