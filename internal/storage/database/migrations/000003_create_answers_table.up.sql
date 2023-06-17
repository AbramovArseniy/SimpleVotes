CREATE TABLE answers(
    question_id INT
    options INT[]
    user_id INT
    FOREIGN KEY(user_id) REFERENCES(users(id)) ON DELETE NO ACTION
    FOREIGN KEY(question_id) REFERENCES(questions(id)) ON DELETE CASCADE
)