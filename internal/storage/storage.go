package storage

import (
	"errors"

	"github.com/AbramovArseniy/SimpleVotes/internal/types"
)

var (
	ErrNotFound    = errors.New("data not found")
	ErrLoginExists = errors.New("login is already used")
	ErrInvalidData = errors.New("data is invalid")
)

type Storage interface {
	SaveQuestion(types.Question) error
	SaveAnswer(types.Answer) error
	GetPercentages(types.Question) ([]int, error)
	GetQuestion(types.Question) (types.Question, error)
	RegisterUser(types.User) error
	GetUserData(login string) (types.User, error)
	GetUserDataById(id int) (types.User, error)
	GetQuestionsByUser(id int) ([]types.Question, error)
	GetPopularQuestions(id int) ([]types.Question, error)
	Close()
}
