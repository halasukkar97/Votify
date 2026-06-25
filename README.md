# Votify

Votify is a full-stack movie polling application with a Go API and a React frontend for creating polls, adding movie options, voting, viewing results, and searching TMDB.

The app uses:

- Go's standard `net/http` package for routing and handlers.
- PostgreSQL for persistent users, polls, movies, votes, and vote/movie relationships.
- TMDB for external movie search.
- `github.com/joho/godotenv` to load local `.env` values.
- `github.com/DATA-DOG/go-sqlmock` in tests to mock PostgreSQL without requiring a running database.

## Project Structure

- `votify/main.go`: starts the server, loads `.env`, connects to PostgreSQL, and registers routes.
- `votify/database`: owns the shared PostgreSQL connection.
- `votify/api`: HTTP handlers, database helper functions, and TMDB search.
- `votify/poll`: poll domain model and voting rules.
- `votify/movie`: movie domain model.
- `votify/user`: user domain model.
- `votify/vote`: vote domain model.
- `frontend`: React, TypeScript, and Vite client application.

## Environment

Create `votify/.env` locally:

```env
TMDB_API_KEY=your_tmdb_api_key
```

The `.env` file is ignored by git so secrets stay local.

## Database

The app currently connects to PostgreSQL with this connection string in `database.Connect`:

```text
host=localhost port=5432 user=hela-sukkar dbname=movie_vote sslmode=disable
```

Expected tables:

```sql
CREATE TABLE users (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL
);

CREATE TABLE polls (
  id TEXT PRIMARY KEY,
  poll_code  TEXT NOT NULL,
  name TEXT NOT NULL,
  is_closed BOOLEAN NOT NULL,
  is_voting_active BOOLEAN NOT NULL DEFAULT FALSE,
  max_votes_per_person INTEGER NOT NULL,
  deadline TIMESTAMP NOT NULL
);

CREATE TABLE movies (
  id TEXT PRIMARY KEY,
  poll_id TEXT NOT NULL REFERENCES polls(id),
  title TEXT NOT NULL,
  release_year INTEGER NOT NULL,
  description TEXT,
  poster_url TEXT
);

CREATE TABLE votes (
  id TEXT PRIMARY KEY,
  poll_id TEXT NOT NULL REFERENCES polls(id),
  user_id TEXT NOT NULL REFERENCES users(id)
);

CREATE TABLE vote_movies (
  vote_id TEXT NOT NULL REFERENCES votes(id),
  movie_id TEXT NOT NULL REFERENCES movies(id),
  PRIMARY KEY (vote_id, movie_id)
);
```

For an existing database created before voting activation was added, run:

```sql
ALTER TABLE polls
ADD COLUMN IF NOT EXISTS is_voting_active BOOLEAN NOT NULL DEFAULT FALSE;
```

For an existing database created before movie posters were added, run:

```sql
ALTER TABLE movies ADD COLUMN IF NOT EXISTS poster_url TEXT;
```

## Run

From the repository root:

```bash
go run .
```

The API listens on:

```text
http://localhost:8080
```

Run the frontend from the `frontend` directory:

```bash
cd frontend
npm install
npm run dev
```

The Vite development server normally runs at `http://localhost:5173`. The selected language and display name are stored in the browser's local storage.

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

Creates a poll.

Example body:

```json
{
  "name": "Friday Movie Night",
  "maxVotesPerPerson": 2,
  "deadline": "2026-07-01T20:00:00Z"
}
```

```http
GET /polls
```

Lists polls from PostgreSQL. Each poll includes its movies and votes.

```http
GET /polls/{pollCode}
```

Loads one poll by public poll code, including related movies and votes. The backend can still fall back to the internal ID.

```http
PATCH /polls/{pollCode}/activate-voting
```

Moves a poll from setup into voting. After voting starts, movies can no longer be added.

```http
GET /results?pollId={id}
```

Returns vote totals by movie ID.

### Users

```http
POST /users
```

Creates a user.

Example body:

```json
{
  "name": "Hela"
}
```

```http
GET /users
```

Lists users from PostgreSQL.

### Movies

```http
POST /movies
```

Creates a movie option for an existing poll.

Example body:

```json
{
  "title": "Dune",
  "pollId": "poll-id",
  "releaseYear": 2021,
  "description": "Desert politics"
}
```

```http
GET /movies
```

Lists movies from PostgreSQL.

```http
GET /movies/search?q=dune
```

Searches TMDB. The search query is URL-escaped before the external request is sent.

### Votes

```http
POST /votes
```

Creates a vote after validating poll rules:

- poll must exist
- poll must not be closed
- poll deadline must not be expired
- user can vote only once per poll
- selected movie count must not exceed the poll maximum
- selected movie IDs must belong to the poll
- duplicate movie selections are rejected

Example body:

```json
{
  "pollId": "poll-id",
  "userId": "user-id",
  "movieIds": ["movie-id-1", "movie-id-2"]
}
```

Votes are saved transactionally: the vote row and all selected movie rows either save together or roll back together.

## Tests

Run all backend tests from the repository root:

```bash
go test ./...
```

Test coverage includes:

- constructors for users, movies, votes, and polls
- poll mutation helpers
- poll validation rules
- vote counting
- PostgreSQL save/read helper functions using `sqlmock`
- HTTP handlers using `httptest`
- TMDB search behavior using a mocked HTTP transport

