package postgres

import (
	"context"
	"database/sql"
	"fmt"
	pb "podcast_service/genproto/episodes"
	"time"

	"github.com/google/uuid"
)

type EpisodeRepo struct {
	Db *sql.DB
}

func NewEpisodeRepo(db *sql.DB) *EpisodeRepo {
	return &EpisodeRepo{Db: db}
}

func (e *EpisodeRepo) CreatePodcastEpisode(episode *pb.EpisodeCreate) (string, error) {
	query := `
	insert into
		episodes(
		episode_id,
		podcast_id,
   	 	user_id,
      	title,
      	file_audio,
      	description,
      	duration,
		updated_at
      	)
	values ($1, $2, $3, $4, $5, $6. $7, $8)	
`
	id := uuid.NewString()
	tx, err := e.Db.Begin()
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(query, id, episode.PodcastId, episode.UserId, episode.Title, episode.FileAudio,
		episode.Description, episode.Duration, time.Now())
	if err != nil {
		tx.Rollback()
		return "", err
	}
	err = tx.Commit()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (p *PodcastRepo) GetEpisodesByPodcastId(podcastId *pb.ID) (*pb.Episodes, error) {
	query := `select id, podcast_id, user_id, title, file_audio,
	description, duration, created_at, updated_at from episodes 
	where podcast_id = $1`

	rows, err := p.Db.Query(query, podcastId)
	if err != nil {
		return nil, err
	}

	episodes := pb.Episodes{}
	for rows.Next() {
		episode := pb.Episode{}
		err := rows.Scan(&episode.Id, &episode.PodcastId, &episode.UserId,
			&episode.Title, &episode.FileAudio, &episode.Description,
			&episode.Duration, &episode.CreatedAt, &episode.UpdatedAt)
		if err != nil {
			return nil, err
		}
		episodes.Episodes = append(episodes.Episodes, &episode)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &episodes, nil
}

func (e *EpisodeRepo) UpdateEpisode(ctx context.Context, in *pb.IDs) (*pb.Void, error) {
	query := `update episodes set `
	params := []interface{}{}

	if in.Episode.UserId != "" {
		query += fmt.Sprintf("user_id = $%d, ", len(params)+1)
		params = append(params, in.Episode.UserId)
	}
	if in.Episode.Title != "" {
		query += fmt.Sprintf("title = $%d, ", len(params)+1)
		params = append(params, in.Episode.Title)
	}
	if in.Episode.FileAudio != nil {
		query += fmt.Sprintf("file_audio = $%d, ", len(params)+1)
		params = append(params, in.Episode.FileAudio)
	}
	if in.Episode.Description != "" {
		query += fmt.Sprintf("description = $%d, ", len(params)+1)
		params = append(params, in.Episode.Description)
	}
	if in.Episode.Duration > 0 {
		query += fmt.Sprintf("duration = $%d, ", len(params)+1)
		params = append(params, in.Episode.Duration)
	}
	query += fmt.Sprintf("updated_at = $%d ", len(params)+1)
	params = append(params, time.Now())
	query += fmt.Sprintf("where podcast_id = $%d ", len(params)+1)
	params = append(params, in.PodcastId)
	query += fmt.Sprintf(" and id = $%d and deleted_at = null", len(params)+1)
	params = append(params, in.EpisodeId)

	tr, err := e.Db.Begin()

	defer func() {
		if err != nil {
			tr.Rollback()
		} else {
			tr.Commit()
		}
	}()

	_, err = e.Db.Exec(query, params...)
	if err != nil {
		return nil, err
	}

	return &pb.Void{}, nil
}

func (e *EpisodeRepo) DeletePodcastEpisode(ids pb.IDsForDelete) error {
	query := `
	update 
	    episodes 
	set 
	    deleted_at = now() 
	where 
	    episode_id = $1,
		podcast_id = $2
`
	tx, err := e.Db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(query, ids.EpisodeId, ids.PodcastId)
	if err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (p *PodcastRepo) PublishPodcast(podcastId *pb.ID) (*pb.Success, error) {
	tx, err := p.Db.Begin()
	if err != nil {
		return &pb.Success{Success: false}, err
	}
	defer tx.Commit()

	query := `update podcasts set status = 'published' where status != 
	'published' and id = $1`

	res, err := tx.Exec(query, podcastId)
	if err != nil {
		return &pb.Success{Success: false}, err
	}

	if err, _ := res.RowsAffected(); err <= 0 {
		return &pb.Success{Success: false}, fmt.Errorf("no rows affected")
	}
	return &pb.Success{Success: true}, nil
}