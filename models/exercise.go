package models

import "github.com/google/uuid"

// ExercisePublic item da biblioteca de exercícios (tabela exercises, is_active).
type ExercisePublic struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Equipment      *string   `json:"equipment,omitempty"`
	TargetMuscle   *string   `json:"target_muscle,omitempty"`
	VideoURL       *string   `json:"video_url"`
	VideoEmbedURL  *string   `json:"video_embed_url"`
	Instructions   *string   `json:"instructions,omitempty"`
	ThumbnailURL   *string   `json:"thumbnail_url,omitempty"`
}

// ExerciseRepository leitura da biblioteca de exercícios.
type ExerciseRepository interface {
	ListActive(search string) ([]*ExercisePublic, error)
}
