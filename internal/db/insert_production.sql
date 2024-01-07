/*
LIMIT 8 is modified by schema.go to a value set by command-line flag.
generate_private_key() is a user-defined function from appfunc.go.
uuidv7() is a user-defined function from appfunc.go.
`WHERE x != 0` is a hack to make each row different.
*/

DROP TRIGGER IF EXISTS derive_public_key;
WITH RECURSIVE seq(x) AS (SELECT 3 UNION ALL SELECT x + 1 FROM seq LIMIT 8)
INSERT INTO users (uuid, email, password, constituency, first_name, last_name, private_key)
SELECT
	uuidv7(),
	'user' || x || '@sentinelvote.tech',
	'password',
	(SELECT constituency FROM constituencies WHERE x != 0 ORDER BY RANDOM() LIMIT 1),
	(SELECT first_name FROM first_names      WHERE x != 0 ORDER BY RANDOM() LIMIT 1),
	(SELECT last_name FROM last_names        WHERE x != 0 ORDER BY RANDOM() LIMIT 1),
	''
FROM seq;

DROP TRIGGER IF EXISTS derive_public_key;
DROP TABLE IF EXISTS constituencies;
DROP TABLE IF EXISTS first_names;
DROP TABLE IF EXISTS last_names;