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
	saveQuestionStmt       = `INSERT INTO questions (text, type, options, user_id) VALUES($1, $2, $3, $4)`
	getQuestionsByUserStmt = `SELECT questions.id, questions.text as text, questions.options, questions.user_id, COALESCE(ans_cnt.cnt_usr,0)
								FROM questions
								LEFT JOIN  
								(SELECT answers.question_id as qid, COUNT(answers.user_id) as cnt_usr
								FROM answers 
								GROUP BY answers.question_id) AS ans_cnt
								ON ans_cnt.qid = questions.id
								WHERE user_id=$1
								ORDER BY ans_cnt.cnt_usr`
	getUserByLoginStmt       = `SELECT id, password FROM users WHERE login=$1`
	getQuestionByIdStmt      = `SELECT text, type, options, user_id FROM questions WHERE id=$1`
	getUserByIdStmt          = `SELECT login FROM users WHERE id=$1`
	getAnswersWithOptionStmt = `SELECT COUNT(*) FROM answers WHERE question_id=$1 AND option=$2`
	getPopularQuestionsStmt  = `SELECT questions.id, questions.text as text, questions.options, questions.user_id, COALESCE(ans_cnt.cnt_usr,0)
								FROM questions
								LEFT JOIN  
								(SELECT answers.question_id as qid, COUNT(answers.user_id) as cnt_usr
								FROM answers 
								GROUP BY answers.question_id) AS ans_cnt
								ON ans_cnt.qid = questions.id
								ORDER BY ans_cnt.cnt_usr`
	getAllAnswersStmt = `SELECT COUNT(*) FROM answers WHERE question_id=$1`
	saveAnswerStmt    = `INSERT INTO anwers (question_id, options, user_id) VALUES($1, $2, $3)`
	registerUserStmt  = `INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`
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
	_, err := db.DB.Query(saveQuestionStmt, q.Text, q.Type, q.Options, q.UserID)
	if err != nil {
		return fmt.Errorf("error while making sql query question: %w", err)
	}
	return nil
}

func (db *Database) SaveAnswer(a types.Answer) error {
	_, err := db.DB.Query(saveAnswerStmt, a.QuestionId, a.Options, a.UserID)
	if err != nil {
		return fmt.Errorf("error while making sql query question: %w", err)
	}
	return nil
}

func (db *Database) GetPercentages(q types.Question) ([]int, error) {
	var percentages []int
	var totalAns int
	err := db.DB.QueryRow(getAllAnswersStmt, q.Id).Scan(&totalAns)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("error while getting data from db: %w", err)
	}
	for i := range q.Options {
		var ans int
		err := db.DB.QueryRow(getAnswersWithOptionStmt, q.Id, i).Scan(&ans)
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
	rows, err := db.DB.Query(getQuestionsByUserStmt, user_id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while making query to database: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var q types.Question
		rows.Scan(&q.Text, &q.Type, &q.Options, &q.UserID)
		q.Percentages, err = db.GetPercentages(q)
		if err != nil {
			return questions, err
		}
		questions = append(questions, q)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return questions, nil
}

func (db *Database) GetPopularQuestions() ([]types.Question, error) {
	var questions []types.Question
	rows, err := db.DB.Query(getPopularQuestionsStmt)
	if err != nil {
		return questions, fmt.Errorf("error while doing query to database: %w", err)
	}
	for rows.Next() {
		var q types.Question
		var userCnt int
		err := rows.Scan(&q.Id, &userCnt, &q.Text, &q.Type, &q.Options, &q.UserID)
		if errors.Is(err, sql.ErrNoRows) {
			return questions, storage.ErrNotFound
		}
		if err != nil {
			return questions, err
		}
		q.Percentages, err = db.GetPercentages(q)
		if err != nil {
			return questions, err
		}
		questions = append(questions, q)

	}
	return questions, nil
}

func (db *Database) GetQuestion(id int) (types.Question, error) {
	var q types.Question
	err := db.DB.QueryRow(getQuestionByIdStmt, id).Scan(&q.Text, &q.Type, &q.Options)
	if err == sql.ErrNoRows {
		return q, storage.ErrNotFound
	}
	if err != nil {
		return q, fmt.Errorf("error while doing query to database: %w", err)
	}
	return q, nil
}

func (db *Database) RegisterUser(u *types.User) error {
	if err := u.GeneratePasswordHash(); err != nil {
		return fmt.Errorf("cannot generate password hash: %w", err)
	}
	err := db.DB.QueryRow(registerUserStmt, u.Login, u.Password).Scan(&u.Id)
	if err != nil {
		return fmt.Errorf("error while making sql query question: %w", err)
	}
	return nil
}

func (db *Database) GetUserByLogin(login string) (types.User, error) {
	var user types.User
	err := db.DB.QueryRow(getUserByLoginStmt, login).Scan(&user.Id, &user.Password)
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
	err := db.DB.QueryRow(getUserByLoginStmt, id).Scan(&user.Login)
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
