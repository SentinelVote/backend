/*
argon2id()             is a user-defined function from appfunc.go.
generate_private_key() is a user-defined function from appfunc.go.
uuidv7()               is a user-defined function from appfunc.go.
derive_public_key()    is a user-defined function from appfunc.go.

WHERE x != 0 is a hack to make each row different.
*/

-- noinspection SqlResolveForFile
INSERT INTO users (uuid, email, password, has_default_password, is_central_authority) VALUES
(uuidv7(), 'admin@sentinelvote.tech', argon2id('password'), FALSE, TRUE);

CREATE TRIGGER IF NOT EXISTS derive_public_key AFTER INSERT ON users
BEGIN
UPDATE users SET public_key = derive_public_key(private_key) WHERE email = NEW.email;
END;

PRAGMA recursive_triggers = ON;

WITH RECURSIVE seq(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM seq LIMIT 2)
INSERT INTO users (uuid, email, password, has_default_password, constituency, first_name, last_name, private_key)
SELECT
	uuidv7(),
	'user' || x || '@sentinelvote.tech',
	argon2id('password'),
	FALSE,
	(SELECT constituency FROM constituencies WHERE x != 0 ORDER BY RANDOM() LIMIT 1),
	(SELECT first_name FROM first_names      WHERE x != 0 ORDER BY RANDOM() LIMIT 1),
	(SELECT last_name FROM last_names        WHERE x != 0 ORDER BY RANDOM() LIMIT 1),
	generate_private_key()
FROM seq;
