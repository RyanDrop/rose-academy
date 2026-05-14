package models

import (
	"context"
	"database/sql"
	"strings"
)

// PostgresExerciseRepository lista exercícios ativos.
type PostgresExerciseRepository struct {
	db *sql.DB
}

// NewPostgresExerciseRepository cria o repositório.
func NewPostgresExerciseRepository(db *sql.DB) *PostgresExerciseRepository {
	return &PostgresExerciseRepository{db: db}
}

// ListActive retorna exercícios is_active; search opcional (ILIKE em nome, músculo, equipamento).
func (r *PostgresExerciseRepository) ListActive(search string) ([]*ExercisePublic, error) {
	search = strings.TrimSpace(search)
	var rows *sql.Rows
	var err error
	if search == "" {
		rows, err = r.db.QueryContext(context.Background(), `
			SELECT id, name, equipment, target_muscle, video_url, video_embed_url, instructions, thumbnail_url
			FROM exercises WHERE is_active = true ORDER BY name ASC`)
	} else {
		pattern := "%" + search + "%"
		rows, err = r.db.QueryContext(context.Background(), `
			SELECT id, name, equipment, target_muscle, video_url, video_embed_url, instructions, thumbnail_url
			FROM exercises WHERE is_active = true
			  AND (name ILIKE $1 OR COALESCE(target_muscle,'') ILIKE $1 OR COALESCE(equipment,'') ILIKE $1)
			ORDER BY name ASC`, pattern)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ExercisePublic
	for rows.Next() {
		ex, err := scanExerciseRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ex)
	}
	return out, rows.Err()
}

func scanExerciseRow(scanner interface {
	Scan(dest ...any) error
}) (*ExercisePublic, error) {
	var ex ExercisePublic
	var eq, mus, vid, emb, ins, thumb sql.NullString
	if err := scanner.Scan(&ex.ID, &ex.Name, &eq, &mus, &vid, &emb, &ins, &thumb); err != nil {
		return nil, err
	}
	if eq.Valid {
		s := eq.String
		ex.Equipment = &s
	}
	if mus.Valid {
		s := mus.String
		ex.TargetMuscle = &s
	}
	if vid.Valid {
		s := vid.String
		ex.VideoURL = &s
	}
	if emb.Valid {
		s := emb.String
		ex.VideoEmbedURL = &s
	}
	if ins.Valid {
		s := ins.String
		ex.Instructions = &s
	}
	if thumb.Valid {
		s := thumb.String
		ex.ThumbnailURL = &s
	}
	return &ex, nil
}
