# Voting App

Voting App is a full-stack polling application with a Go REST API and a React frontend. It started as a movie voting project, but the app now supports generic poll options so a poll can be about movies, books, games, restaurants, activities, vacation destinations, or custom choices.

The app uses:

- Go's standard `net/http` package for routing and handlers.
- PostgreSQL for users, polls, options, votes, and vote/option relationships.
- React, TypeScript, and Vite for the frontend.
- TMDB as the first external option search provider for movie polls.
- `github.com/joho/godotenv` to load local `.env` values.
- `github.com/DATA-DOG/go-sqlmock` in backend tests.

## Project Structure

- `main.go`: root entry point kept for platforms that build the repository root.
- `cmd/server/main.go`: idiomatic Go server entry point for local development and Render.
- `internal/app`: wires config, database, repositories, services, handlers, and routes.
- `internal/api`: HTTP request/response handlers.
- `internal/config`: environment variable loading and app configuration.
- `internal/database`: PostgreSQL connection setup.
- `internal/domain`: core models and validation rules for polls, options, votes, and users.
- `internal/repository`: PostgreSQL queries and persistence logic.
- `internal/service`: business rules between handlers and repositories.
- `frontend`: React, TypeScript, and Vite client application.

## Environment

Create `.env` locally:

```env
DATABASE_URL=postgres://user:password@localhost:5432/voting_app?sslmode=disable
TMDB_API_KEY=your_tmdb_api_key
PORT=8080
```

The `.env` file is ignored by git so secrets stay local.

For the frontend, set the backend URL when needed:

```env
VITE_API_BASE_URL=http://localhost:8080
```

## Database

Current tables use `options` instead of `movies` so polls can contain any kind of item.

```sql
CREATE TABLE users (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL
);

CREATE TABLE polls (
  id TEXT PRIMARY KEY,
  poll_code TEXT NOT NULL,
  name TEXT NOT NULL,
  poll_type TEXT NOT NULL DEFAULT 'movie',
  is_closed BOOLEAN NOT NULL,
  is_voting_active BOOLEAN NOT NULL DEFAULT FALSE,
  max_votes_per_person INTEGER NOT NULL,
  deadline TIMESTAMP NOT NULL
);

CREATE TABLE options (
  id TEXT PRIMARY KEY,
  poll_id TEXT NOT NULL REFERENCES polls(id),
  title TEXT NOT NULL,
  description TEXT,
  image_url TEXT,
  release_year INTEGER,
  metadata JSONB DEFAULT '{}'::jsonb
);

CREATE TABLE votes (
  id TEXT PRIMARY KEY,
  poll_id TEXT NOT NULL REFERENCES polls(id),
  user_id TEXT NOT NULL REFERENCES users(id)
);

CREATE TABLE vote_options (
  vote_id TEXT NOT NULL REFERENCES votes(id),
  option_id TEXT NOT NULL REFERENCES options(id),
  PRIMARY KEY (vote_id, option_id)
);
```

## Migration From Existing Movie Tables

Existing movie polls continue to work because the backend still has compatibility fallbacks for `movies` and `vote_movies`. To migrate old data to the generic schema, run:

```sql
ALTER TABLE polls
ADD COLUMN IF NOT EXISTS poll_type TEXT NOT NULL DEFAULT 'movie';

CREATE TABLE IF NOT EXISTS options (
  id TEXT PRIMARY KEY,
  poll_id TEXT NOT NULL REFERENCES polls(id),
  title TEXT NOT NULL,
  description TEXT,
  image_url TEXT,
  release_year INTEGER,
  metadata JSONB DEFAULT '{}'::jsonb
);

INSERT INTO options (id, poll_id, title, description, image_url, release_year)
SELECT id, poll_id, title, description, poster_url, release_year
FROM movies
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS vote_options (
  vote_id TEXT NOT NULL REFERENCES votes(id),
  option_id TEXT NOT NULL REFERENCES options(id),
  PRIMARY KEY (vote_id, option_id)
);

INSERT INTO vote_options (vote_id, option_id)
SELECT vote_id, movie_id
FROM vote_movies
ON CONFLICT (vote_id, option_id) DO NOTHING;
```

Keep the old `movies` and `vote_movies` tables until you confirm old clients and deployments no longer need them.

## Run

From the repository root:

```bash
go run ./cmd/server
```

The API listens on `http://localhost:8080` by default.

Run the frontend from the `frontend` directory:

```bash
cd frontend
npm install
npm run dev
```

The Vite development server normally runs at `http://localhost:5173`.

## Deploy Backend on Render

Use these settings for the backend service:

```text
Build Command: go build -o app ./cmd/server
Start Command: ./app
```

The root `main.go` also allows Render's older `go build -o app .` setting to keep working.

## API Routes

### Health

```http
GET /
```

Returns a simple text message confirming the API is running.

### Polls

```http
POST /polls
```

Creates a poll. `pollType` can be `movie`, `book`, `game`, `restaurant`, `activity`, or `custom`.

```json
{
  "name": "Friday Night Picks",
  "pollType": "movie",
  "maxVotesPerPerson": 2,
  "deadline": "2026-07-01T20:00:00Z"
}
```

```http
GET /polls
GET /polls/{pollCode}
PATCH /polls/{pollCode}/activate-voting
GET /results?pollCode={pollCode}
```

Poll responses include `options`, `votes`, and a compatibility `movies` field for older frontend code.

### Options

```http
POST /options
GET /options
GET /options/search?type=movie&q=dune
```

Creates, lists, or searches options. Movie search uses TMDB. Unsupported poll types currently return an empty suggestion list until a provider is added.

Example option body:

```json
{
  "title": "Dune",
  "pollId": "poll-id",
  "releaseYear": 2021,
  "description": "Desert politics",
  "imageUrl": "https://image.example/dune.jpg"
}
```

Legacy `/movies` and `/movies/search` routes still work while old clients migrate.

### Users

```http
POST /users
PATCH /users/{userId}
GET /users
```

Users keep the same ID when their display name changes, so renaming a user does not allow duplicate voting.

### Votes

```http
POST /votes
```

Creates a vote after validating poll rules:

- poll must exist
- voting must be active
- poll must not be closed
- poll deadline must not be expired
- user can vote only once per poll
- selected option count must not exceed the poll maximum
- selected option IDs must belong to the poll
- duplicate option selections are rejected

Example body:

```json
{
  "pollCode": "12345678",
  "userId": "user-id",
  "optionIds": ["option-id-1", "option-id-2"]
}
```

Votes are saved transactionally: the vote row and all selected option rows either save together or roll back together.

## Tests

Run all backend tests from the repository root:

```bash
go test ./...
```

Run the frontend build from `frontend`:

```bash
npm run build
```
