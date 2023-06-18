package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/AbramovArseniy/SimpleVotes/internal/storage"
	"github.com/AbramovArseniy/SimpleVotes/internal/types"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	SaveQuestionStmt         = `INSERT INTO questions (text, options, user_id) VALUES($1, $2, $3)`
	GetQuestionsByUserStmt   = `SELECT text, options, user_id FROM questions WHERE user_id=$1`
	GetUserByLoginStmt       = `SELECT id, password FROM users WHERE login=$1`
	GetUserByIdStmt          = `SELECT login FROM users WHERE id=$1`
	GetAnswersWithOptionStmt = `SELECT COUNT(*) FROM answers WHERE question_id=$1 AND option=$2`
	GetPopularQuestionsStmt  = `SELECT question_id, COUNT(user_id) as cnt_usr, question.text as text FROM answers LEFT JOIN questions ON questions.id = answers.question_id ORDERED BY cnt_usr DESC`
	GetAllAnswersStmt        = `SELECT COUNT(*) FROM answers WHERE question_id=$1`
	SaveAnswerStmt           = `INSERT INTO anwers (question_id, options, user_id) VALUES($1, $2, $3)`
	RegisterUserStmt         = `INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`
)

type Database struct {
	DB   *sql.DB
	Addr string
}

func NewDatabase(address string) (*Database, error) {
	db, err := sql.Open("pgx", address)
	if err != nil {
		return nil, fmt.Errorf("error while opening db: %w", err)
	}
	database := &Database{
		DB:   db,
		Addr: address,
	}
	err = database.Migrate()
	if err != nil {
		return nil, fmt.Errorf("migration error: %w", err)
	}
	return database, nil
}

func (db *Database) Migrate() error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://../internal/storage/database/migrations",
		db.Addr, driver)
	if err != nil {
		return fmt.Errorf("could not create migration: %w", err)
	}
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func (db *Database) SaveQuestion(q types.Question) error {
	_, err := db.DB.Query(SaveQuestionStmt, q.Text, q.Options, q.UserID)
	if err != nil {
		return fmt.Errorf("error while making sql query question: %w", err)
	}
	return nil
}

func (db *Database) SaveAnswer(a types.Answer) error {
	_, err := db.DB.Query(SaveAnswerStmt, a.QuestionId, a.Options, a.UserID)
	if err != nil {
		return fmt.Errorf("error while making sql query question: %w", err)
	}
	return nil
}

func (db *Database) GetPercentages(q types.Question) ([]int, error) {
	var percentages []int
	var totalAns int
	err := db.DB.QueryRow(GetAllAnswersStmt, q.Id).Scan(&totalAns)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("error while getting data from db: %w", err)
	}
	for i := range q.Options {
		var ans int
		err := db.DB.QueryRow(GetAnswersWithOptionStmt, q.Id, i).Scan(&ans)
		if err == sql.ErrNoRows {
			percentages = append(percentages, 0)
		} else if err != nil {
			return nil, fmt.Errorf("error while getting data from db: %w", err)
		} else {
			percentages = append(percentages, (100*ans)/totalAns)
		}
	}
	return percentages, nil
}

func (db *Database) GetQuestionsByUser(user_id int) ([]types.Question, error) {
	var questions = make([]types.Question, 100)
	rows, err := db.DB.Query(GetQuestionsByUserStmt, user_id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while making query to database: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var q types.Question
		rows.Scan(&q.Text, &q.Options, &q.UserID)
		questions = append(questions, q)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return questions, nil
}

func (db *Database) GetPopularQuestions() ([]types.Question, error) {
	var questions []types.Question
	return questions, nil
}

func (db *Database) GetQuestion(id int) (types.Question, error) {
	var question types.Question
	return question, nil
}

func (db *Database) RegisterUser(u *types.User) error {
	if err := u.GeneratePasswordHash(); err != nil {
		return fmt.Errorf("cannot generate password hash: %w", err)
	}
	err := db.DB.QueryRow(RegisterUserStmt, u.Login, u.Password).Scan(&u.Id)
	if err != nil {
		return fmt.Errorf("error while making sql query question: %w", err)
	}
	return nil
}

func (db *Database) GetUserByLogin(login string) (types.User, error) {
	var user types.User
	err := db.DB.QueryRow(GetUserByLoginStmt, login).Scan(&user.Id, &user.Password)
	if err == sql.ErrNoRows {
		return user, storage.ErrNotFound
	}
	if err != nil {
		return user, fmt.Errorf("error while doing query to database: %w", err)
	}
	return user, nil
}

func (db *Database) GetUserById(id int) (types.User, error) {
	var user types.User
	err := db.DB.QueryRow(GetUserByLoginStmt, id).Scan(&user.Login)
	if err == sql.ErrNoRows {
		return user, storage.ErrNotFound
	}
	if err != nil {
		return user, fmt.Errorf("error while doing query to database: %w", err)
	}
	return user, nil
}

func (db *Database) Close() {
	db.DB.Close()
}
