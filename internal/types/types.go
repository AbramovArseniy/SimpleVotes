package types

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
	UserID      int
}

type User struct {
	Id       int
	Login    string
	Password string
}

type Answer struct {
	QuestionId int
	UserID     int
	Options    []int
}
