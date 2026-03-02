package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type Gallery struct {
	ID     int
	UserId int
	Title  string
}

type GalleryService struct {
	DB *sql.DB
}

func (gserv *GalleryService) Create(title string, userID int) (*Gallery, error) {
	gallery := Gallery{
		Title:  title,
		UserId: userID,
	}
	row := gserv.DB.QueryRow(`
	INSERT INTO galleries (title, user_id)
	VALUES ($1, $2) RETURNING id;`, gallery.Title, gallery.UserId)
	err := row.Scan(&gallery.ID)
	if err != nil {
		return nil, fmt.Errorf("create gallery failed: %w", err)
	}
	return &gallery, nil
}

func (gserv *GalleryService) ByID(id int) (*Gallery, error) {
	gallery := Gallery{
		ID: id,
	}

	row := gserv.DB.QueryRow(`
	SELECT title, user_id
	FROM galleries
	WHERE id=$1;`, gallery.ID)
	err := row.Scan(&gallery.Title, &gallery.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("quering gallery by id failed: %w", err)
	}
	return &gallery, nil
}

func (gserv *GalleryService) ByUserID(userID int) ([]Gallery, error) {
	rows, err := gserv.DB.Query(`SELECT id, title
	FROM galleries
	WHERE user_id=$1;`, userID)
	if err != nil {
		return nil, fmt.Errorf("quering gallery by userID failed: %w", err)
	}
	var galleries []Gallery
	for rows.Next() {
		gallery := Gallery{
			UserId: userID,
		}
		err = rows.Scan(&gallery.ID, &gallery.Title)
		if err != nil {
			return nil, fmt.Errorf("quering gallery by userID failed: %w", err)
		}
		galleries = append(galleries, gallery)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("quering gallery by userID failed: %w", err)
	}
	return galleries, nil
}

func (gserv *GalleryService) Update(gallery *Gallery) error {
	_, err := gserv.DB.Exec(`
	UPDATE galleries
	SET title=$2
	WHERE id=$1;`, gallery.ID, gallery.Title)
	if err != nil {
		return fmt.Errorf("update gallery failed: %w", err)
	}
	return nil
}

func (gserv *GalleryService) Delete(id int) error {
	_, err := gserv.DB.Exec(`
	DELETE FROM galleries
	WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete gallery failed: %w", err)
	}
	return nil
}
