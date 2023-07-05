package types

import "golang.org/x/crypto/bcrypt"

const (
	OneOptionType   QuestionType = "one-option"
	MultipleOptions QuestionType = "multiple-options"
)

type QuestionType string

type Question struct {
	Id          int
	Type        QuestionType
	Text        string
	Options     []string
	Percentages []int
	Answered    []int
	IsAnswered  bool
	UserID      int
}

type User struct {
	Id       int
	Login    string
	Password string
}

func (u *User) GeneratePasswordHash() error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	u.Password = string(bytes)
	return err
}
func (u *User) CheckPasswordHash(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(password), []byte(u.Password))
	return err == nil
}

type Answer struct {
	QuestionId int
	UserID     int
	Options    []int
}

type PopularQuestionsPageData struct {
	Questions    []Question
	LoggedInUser User
}

type ProfilePageData struct {
	Questions    []Question
	LoggedInUser User
	User         User
}

type AddQuestionPageData struct {
	LoggedInUser User
}
