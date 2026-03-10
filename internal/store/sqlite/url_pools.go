package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aven/ngoogle/internal/model"
)

type urlPoolStore struct{ db *sql.DB }

func (s *urlPoolStore) Create(ctx context.Context, p *model.URLPool) error {
	p.Normalize()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO url_pools(id,name,type,description,urls_json,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.Type, p.Description, p.URLsJSON, p.CreatedAt.UTC(), p.UpdatedAt.UTC())
	return err
}

func (s *urlPoolStore) Get(ctx context.Context, id string) (*model.URLPool, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id,name,type,description,urls_json,created_at,updated_at FROM url_pools WHERE id=?`, id)
	return scanURLPool(row)
}

func (s *urlPoolStore) List(ctx context.Context) ([]*model.URLPool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id,name,type,description,urls_json,created_at,updated_at FROM url_pools ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.URLPool
	for rows.Next() {
		p, err := scanURLPool(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (s *urlPoolStore) Update(ctx context.Context, p *model.URLPool) error {
	p.Normalize()
	_, err := s.db.ExecContext(ctx, `
		UPDATE url_pools
		SET name=?, type=?, description=?, urls_json=?, updated_at=?
		WHERE id=?`,
		p.Name, p.Type, p.Description, p.URLsJSON, p.UpdatedAt.UTC(), p.ID)
	return err
}

func (s *urlPoolStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM url_pools WHERE id=?`, id)
	return err
}

func scanURLPool(row scanner) (*model.URLPool, error) {
	p := &model.URLPool{}
	err := row.Scan(&p.ID, &p.Name, &p.Type, &p.Description, &p.URLsJSON, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("url pool not found")
	}
	if err != nil {
		return nil, err
	}
	p.Normalize()
	return p, nil
}
