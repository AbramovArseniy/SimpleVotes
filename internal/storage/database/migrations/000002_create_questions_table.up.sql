CREATE TABLE questions(
    id SERIAL UNIQUE,
    type VARCHAR(32),
    text VARCHAR(512),
    options VARCHAR(512)[],
    user_id INT,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE NO ACTION
)