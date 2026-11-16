REVOKE CREATE ON SCHEMA public FROM PUBLIC;
SELECT format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database())
\gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO moetopia_app', current_database())
\gexec
GRANT USAGE ON SCHEMA identity, sessions, "authorization", developer, administration, notifications, audit TO moetopia_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA identity, sessions, "authorization", developer, administration, notifications, audit TO moetopia_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA identity, sessions, "authorization", developer, administration, notifications, audit TO moetopia_app;
ALTER DEFAULT PRIVILEGES FOR ROLE moetopia_migrator IN SCHEMA identity, sessions, "authorization", developer, administration, notifications, audit GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO moetopia_app;
ALTER DEFAULT PRIVILEGES FOR ROLE moetopia_migrator IN SCHEMA identity, sessions, "authorization", developer, administration, notifications, audit GRANT USAGE, SELECT ON SEQUENCES TO moetopia_app;
