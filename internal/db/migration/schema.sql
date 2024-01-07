DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS folded_public_keys;
DROP TABLE IF EXISTS constituencies;
DROP TABLE IF EXISTS first_names;
DROP TABLE IF EXISTS last_names;
DROP TABLE IF EXISTS is_end_of_election;

/*
PRIMARY KEYS must also be declared NOT NULL:
https://www.sqlite.org/lang_createtable.html#primkeyconst
*/

CREATE TABLE users (
uuid                 TEXT    PRIMARY KEY NOT NULL,
email                TEXT    UNIQUE      NOT NULL,
password             TEXT                NOT NULL DEFAULT 'password',
public_key           TEXT                NOT NULL DEFAULT '',
has_voted            BOOLEAN             NOT NULL DEFAULT FALSE,
has_default_password BOOLEAN             NOT NULL DEFAULT TRUE,
constituency         TEXT                NOT NULL DEFAULT 'N/A',
first_name           TEXT                NOT NULL DEFAULT 'N/A',
last_name            TEXT                NOT NULL DEFAULT 'N/A',
is_central_authority BOOLEAN             NOT NULL DEFAULT FALSE,
private_key          TEXT                NOT NULL DEFAULT ''
);

CREATE TABLE folded_public_keys (
singleton          INT2 PRIMARY KEY NOT NULL CHECK ( singleton = 1 ),
folded_public_keys TEXT             NOT NULL
);

CREATE TABLE is_end_of_election (
singleton INT2 PRIMARY KEY NOT NULL CHECK ( singleton = 1 ),
is_end_of_election BOOLEAN NOT NULL
);