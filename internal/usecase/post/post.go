package post

import "arktie.org/internal/data/client"

type Usecase struct {
	db *client.Database
}

func NewUsecase(db *client.Database) *Usecase {
	return &Usecase{
		db: db,
	}
}
