CREATE TABLE submissions (
             username VARCHAR(255) NOT NULL,
             timestamp TIMESTAMP NOT NULL,
             submission_count INT NOT NULL
);
ALTER TABLE submissions ADD CONSTRAINT unique_user_time UNIQUE (username, timestamp);
CREATE INDEX idx_user_time ON submissions (username, timestamp);