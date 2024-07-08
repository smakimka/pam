CREATE USER server with password 'server_password';
CREATE DATABASE server_db;
GRANT ALL PRIVILEGES ON DATABASE server_db TO server;
ALTER DATABASE server_db OWNER TO server;
