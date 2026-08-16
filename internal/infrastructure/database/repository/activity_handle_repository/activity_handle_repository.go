package activity_handle_repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) ports.ActivityHandleRepository { return &Repository{db: db} }

func (r *Repository) Save(ctx context.Context, handle domain.ActivityHandle) error {
	endpoints, _ := json.Marshal(handle.Endpoints)
	metadata, _ := json.Marshal(handle.Metadata)
	_, err := r.db.ExecContext(ctx, `INSERT INTO activity_handles (
		id, execution_run_id, activity_id, resource_id, runtime_id, external_id,
		status, endpoints, started_at, finished_at, exit_code, failure, metadata
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET external_id=excluded.external_id,
		status=excluded.status, endpoints=excluded.endpoints,
		started_at=excluded.started_at, finished_at=excluded.finished_at,
		exit_code=excluded.exit_code, failure=excluded.failure,
		metadata=excluded.metadata, updated_at=CURRENT_TIMESTAMP`,
		handle.ID, handle.RunID, handle.ActivityID, handle.ResourceID, handle.RuntimeID,
		handle.ExternalID, handle.Status, string(endpoints), handle.StartedAt,
		handle.FinishedAt, handle.ExitCode, handle.Failure, string(metadata))
	return err
}

func (r *Repository) Find(ctx context.Context, id string) (*domain.ActivityHandle, error) {
	var handle domain.ActivityHandle
	var endpoints, metadata string
	err := r.db.QueryRowContext(ctx, `SELECT id, execution_run_id, activity_id,
		resource_id, runtime_id, external_id, status, endpoints, started_at,
		finished_at, exit_code, failure, metadata FROM activity_handles WHERE id=?`, id).
		Scan(&handle.ID, &handle.RunID, &handle.ActivityID, &handle.ResourceID,
			&handle.RuntimeID, &handle.ExternalID, &handle.Status, &endpoints,
			&handle.StartedAt, &handle.FinishedAt, &handle.ExitCode, &handle.Failure,
			&metadata)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(endpoints), &handle.Endpoints)
	_ = json.Unmarshal([]byte(metadata), &handle.Metadata)
	return &handle, nil
}
