# Database Setup Guide

This guide covers setting up PostgreSQL for the Virtual FitCheck application, including installation, configuration, and troubleshooting.

## Overview

Virtual FitCheck uses PostgreSQL to store processing jobs persistently. This allows users to:

- View their job history across browser sessions
- Track multiple try-on requests
- Access results even after server restarts

## Prerequisites

- PostgreSQL 14 or higher
- Go 1.21 or higher (for running migrations)
- `golang-migrate` CLI tool (optional, for manual migrations)

## Installation Options

### Option 1: Docker (Recommended for Development)

Docker provides the easiest way to get PostgreSQL running without installing it system-wide.

#### Step 1: Install Docker

Download and install Docker Desktop from [docker.com](https://www.docker.com/products/docker-desktop/)

#### Step 2: Run PostgreSQL Container

```bash
docker run --name fitcheck-postgres \
  -e POSTGRES_PASSWORD=devpassword \
  -e POSTGRES_DB=virtualfitcheck \
  -p 5432:5432 \
  -d postgres:14
```

**Explanation:**

- `--name fitcheck-postgres`: Names the container for easy reference
- `-e POSTGRES_PASSWORD=devpassword`: Sets the postgres user password
- `-e POSTGRES_DB=virtualfitcheck`: Creates the database automatically
- `-p 5432:5432`: Maps PostgreSQL port to localhost
- `-d`: Runs container in background
- `postgres:14`: Uses PostgreSQL version 14

#### Step 3: Verify Container is Running

```bash
docker ps
```

You should see `fitcheck-postgres` in the list of running containers.

#### Managing the Docker Container

**Stop the container:**

```bash
docker stop fitcheck-postgres
```

**Start the container:**

```bash
docker start fitcheck-postgres
```

**Remove the container (deletes all data):**

```bash
docker rm -f fitcheck-postgres
```

**View container logs:**

```bash
docker logs fitcheck-postgres
```

**Access PostgreSQL shell:**

```bash
docker exec -it fitcheck-postgres psql -U postgres -d virtualfitcheck
```

### Option 2: Local Installation

#### Windows

1. Download PostgreSQL installer from [postgresql.org/download/windows](https://www.postgresql.org/download/windows/)
2. Run the installer and follow the setup wizard
3. Remember the password you set for the `postgres` user
4. Add PostgreSQL bin directory to PATH (usually `C:\Program Files\PostgreSQL\14\bin`)

**Create the database:**

```cmd
psql -U postgres
CREATE DATABASE virtualfitcheck;
\q
```

#### macOS

**Using Homebrew:**

```bash
brew install postgresql@14
brew services start postgresql@14
```

**Create the database:**

```bash
createdb virtualfitcheck
```

#### Linux (Ubuntu/Debian)

```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

**Create the database:**

```bash
sudo -u postgres createdb virtualfitcheck
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'yourpassword';"
```

## Configuration

### Step 1: Configure Environment Variables

Copy the example environment file:

```bash
cd backend
cp .env.example .env
```

### Step 2: Update Database Connection String

Edit `backend/.env` and update the `DATABASE_URL`:

**For Docker setup:**

```env
DATABASE_URL=postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable
```

**For local installation:**

```env
DATABASE_URL=postgresql://postgres:yourpassword@localhost:5432/virtualfitcheck?sslmode=disable
```

**Connection String Format:**

```
postgresql://[user]:[password]@[host]:[port]/[database]?[parameters]
```

### Step 3: Configure Connection Pool (Optional)

Adjust these settings based on your needs:

```env
DATABASE_MAX_CONNECTIONS=25  # Maximum open connections
DATABASE_MAX_IDLE=5          # Maximum idle connections
```

**Guidelines:**

- For development: 10-25 max connections is sufficient
- For production: Scale based on expected concurrent users
- Idle connections should be 20-40% of max connections

## Database Migrations

The application uses database migrations to manage schema changes. Migrations are located in `backend/migrations/`.

### Automatic Migrations (Recommended)

The backend server automatically runs migrations on startup. Simply start the server:

```bash
cd backend
go run main.go
```

You'll see migration logs in the console:

```
Running database migrations...
Applying migration: 000001_create_jobs_table.up.sql
Migrations completed successfully
```

### Manual Migrations

If you need to run migrations manually, you can use the `golang-migrate` CLI tool.

#### Install golang-migrate

**macOS:**

```bash
brew install golang-migrate
```

**Windows:**

```powershell
scoop install migrate
```

**Linux:**

```bash
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/
```

**Or install via Go:**

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

#### Run Migrations

**Apply all migrations (up):**

```bash
cd backend
migrate -path ./migrations -database "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable" up
```

**Rollback last migration (down):**

```bash
migrate -path ./migrations -database "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable" down 1
```

**Check migration version:**

```bash
migrate -path ./migrations -database "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable" version
```

**Force migration version (use with caution):**

```bash
migrate -path ./migrations -database "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable" force VERSION
```

## Database Schema

### Jobs Table

The main table storing processing job information:

| Column           | Type                     | Description                                      |
| ---------------- | ------------------------ | ------------------------------------------------ |
| `id`             | UUID                     | Primary key, auto-generated                      |
| `session_id`     | VARCHAR(255)             | User session identifier                          |
| `user_photo_id`  | VARCHAR(255)             | ID of uploaded user photo                        |
| `shirt_image_id` | VARCHAR(255)             | ID of uploaded shirt image                       |
| `status`         | VARCHAR(50)              | Job status (pending/processing/completed/failed) |
| `progress`       | INTEGER                  | Progress percentage (0-100)                      |
| `status_message` | TEXT                     | Current status message                           |
| `result_id`      | VARCHAR(255)             | ID of result image (when completed)              |
| `result_token`   | VARCHAR(255)             | Token for result access                          |
| `error`          | TEXT                     | Error message (when failed)                      |
| `email`          | VARCHAR(255)             | Optional email for notifications                 |
| `created_at`     | TIMESTAMP WITH TIME ZONE | Job creation timestamp                           |
| `updated_at`     | TIMESTAMP WITH TIME ZONE | Last update timestamp (auto-updated)             |
| `completed_at`   | TIMESTAMP WITH TIME ZONE | Job completion timestamp                         |

### Indexes

- `idx_jobs_session_id`: Fast lookup by session
- `idx_jobs_status`: Fast filtering by status
- `idx_jobs_created_at`: Fast sorting by date

### Triggers

- `update_jobs_updated_at`: Automatically updates `updated_at` on row changes

## Verification

### Verify Database Connection

Test the connection using `psql`:

```bash
psql "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable"
```

### Verify Tables Exist

```sql
\dt
```

You should see the `jobs` table listed.

### Verify Indexes

```sql
\di
```

You should see the three indexes: `idx_jobs_session_id`, `idx_jobs_status`, and `idx_jobs_created_at`.

### Query Sample Data

```sql
SELECT * FROM jobs LIMIT 10;
```

## Troubleshooting

### Connection Refused

**Symptom:** `connection refused` or `could not connect to server`

**Solutions:**

1. Verify PostgreSQL is running:

   ```bash
   # Docker
   docker ps | grep fitcheck-postgres

   # macOS
   brew services list | grep postgresql

   # Linux
   sudo systemctl status postgresql

   # Windows
   # Check Services app for "postgresql-x64-14"
   ```

2. Check if port 5432 is in use:

   ```bash
   # macOS/Linux
   lsof -i :5432

   # Windows
   netstat -ano | findstr :5432
   ```

3. Verify connection string in `.env` matches your setup

### Authentication Failed

**Symptom:** `password authentication failed for user "postgres"`

**Solutions:**

1. Verify password in `DATABASE_URL` matches your PostgreSQL password
2. For Docker, ensure you used the correct password when creating the container
3. Reset PostgreSQL password:

   ```bash
   # Docker
   docker exec -it fitcheck-postgres psql -U postgres -c "ALTER USER postgres PASSWORD 'newpassword';"

   # Local
   sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'newpassword';"
   ```

### Database Does Not Exist

**Symptom:** `database "virtualfitcheck" does not exist`

**Solutions:**

1. Create the database manually:

   ```bash
   # Docker
   docker exec -it fitcheck-postgres createdb -U postgres virtualfitcheck

   # Local
   createdb virtualfitcheck
   # or
   psql -U postgres -c "CREATE DATABASE virtualfitcheck;"
   ```

### Migration Errors

**Symptom:** `Dirty database version` or migration fails

**Solutions:**

1. Check current migration version:

   ```bash
   migrate -path ./migrations -database "YOUR_DATABASE_URL" version
   ```

2. If database is in dirty state, force to a clean version:

   ```bash
   # Force to version 1 (after first migration)
   migrate -path ./migrations -database "YOUR_DATABASE_URL" force 1
   ```

3. Manually verify schema in database:

   ```bash
   psql "YOUR_DATABASE_URL" -c "\d jobs"
   ```

4. If all else fails, drop and recreate:
   ```bash
   # WARNING: This deletes all data!
   psql "YOUR_DATABASE_URL" -c "DROP TABLE IF EXISTS jobs CASCADE;"
   migrate -path ./migrations -database "YOUR_DATABASE_URL" up
   ```

### Too Many Connections

**Symptom:** `sorry, too many clients already` or `remaining connection slots are reserved`

**Solutions:**

1. Reduce `DATABASE_MAX_CONNECTIONS` in `.env`
2. Check for connection leaks in application code
3. Increase PostgreSQL max_connections:

   ```bash
   # Docker
   docker exec -it fitcheck-postgres psql -U postgres -c "ALTER SYSTEM SET max_connections = 100;"
   docker restart fitcheck-postgres

   # Local - edit postgresql.conf
   # Find: max_connections = 100
   # Restart PostgreSQL after editing
   ```

### Slow Queries

**Symptom:** API responses are slow, database queries take too long

**Solutions:**

1. Verify indexes are being used:

   ```sql
   EXPLAIN ANALYZE SELECT * FROM jobs WHERE session_id = 'some-session-id';
   ```

   Look for "Index Scan" in the output.

2. Check database statistics:

   ```sql
   SELECT schemaname, tablename, n_live_tup, n_dead_tup
   FROM pg_stat_user_tables
   WHERE tablename = 'jobs';
   ```

3. Run VACUUM if needed:

   ```sql
   VACUUM ANALYZE jobs;
   ```

4. Add more indexes if needed (consult design document)

### Docker Container Won't Start

**Symptom:** Container exits immediately or won't start

**Solutions:**

1. Check container logs:

   ```bash
   docker logs fitcheck-postgres
   ```

2. Remove and recreate container:

   ```bash
   docker rm -f fitcheck-postgres
   docker run --name fitcheck-postgres \
     -e POSTGRES_PASSWORD=devpassword \
     -e POSTGRES_DB=virtualfitcheck \
     -p 5432:5432 \
     -d postgres:14
   ```

3. Check if port 5432 is already in use:

   ```bash
   # Use a different port
   docker run --name fitcheck-postgres \
     -e POSTGRES_PASSWORD=devpassword \
     -e POSTGRES_DB=virtualfitcheck \
     -p 5433:5432 \
     -d postgres:14

   # Update DATABASE_URL to use port 5433
   ```

### Permission Denied

**Symptom:** `permission denied for table jobs` or similar

**Solutions:**

1. Verify user has correct permissions:

   ```sql
   GRANT ALL PRIVILEGES ON DATABASE virtualfitcheck TO postgres;
   GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
   ```

2. Check if using correct user in connection string

## Production Considerations

### Security

1. **Use strong passwords:**

   ```env
   DATABASE_URL=postgresql://appuser:STRONG_RANDOM_PASSWORD@localhost:5432/virtualfitcheck?sslmode=require
   ```

2. **Enable SSL/TLS:**

   - Change `sslmode=disable` to `sslmode=require`
   - Configure PostgreSQL with SSL certificates

3. **Create dedicated application user:**

   ```sql
   CREATE USER fitcheck_app WITH PASSWORD 'strong_password';
   GRANT CONNECT ON DATABASE virtualfitcheck TO fitcheck_app;
   GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO fitcheck_app;
   ```

4. **Restrict network access:**
   - Configure `pg_hba.conf` to limit connections
   - Use firewall rules to restrict database port

### Performance

1. **Connection pooling:** Already configured in application
2. **Regular maintenance:**

   ```sql
   -- Run weekly
   VACUUM ANALYZE jobs;

   -- Monitor table bloat
   SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
   FROM pg_tables
   WHERE schemaname = 'public';
   ```

3. **Monitoring:**
   - Set up query logging
   - Monitor slow queries
   - Track connection count
   - Monitor disk usage

### Backup

1. **Automated backups:**

   ```bash
   # Daily backup script
   pg_dump "postgresql://postgres:password@localhost:5432/virtualfitcheck" > backup_$(date +%Y%m%d).sql
   ```

2. **Restore from backup:**

   ```bash
   psql "postgresql://postgres:password@localhost:5432/virtualfitcheck" < backup_20251206.sql
   ```

3. **Docker volume backup:**
   ```bash
   docker exec fitcheck-postgres pg_dump -U postgres virtualfitcheck > backup.sql
   ```

## Additional Resources

- [PostgreSQL Official Documentation](https://www.postgresql.org/docs/)
- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [PostgreSQL Connection Strings](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)
- [Docker PostgreSQL Image](https://hub.docker.com/_/postgres)

## Getting Help

If you encounter issues not covered in this guide:

1. Check application logs for detailed error messages
2. Verify your PostgreSQL version: `psql --version`
3. Test connection independently of the application
4. Consult the PostgreSQL logs (location varies by installation)
5. Open an issue on the project repository with:
   - Error message
   - PostgreSQL version
   - Operating system
   - Steps to reproduce
