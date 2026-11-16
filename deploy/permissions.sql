\set ON_ERROR_STOP on
SELECT format('CREATE ROLE moetopia_app LOGIN PASSWORD %L', rtrim(pg_read_file('/run/secrets/app_db_password'), E'\r\n'))
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'moetopia_app')
\gexec
SELECT format('ALTER ROLE moetopia_app PASSWORD %L', rtrim(pg_read_file('/run/secrets/app_db_password'), E'\r\n'))
\gexec
\ir grants.sql
