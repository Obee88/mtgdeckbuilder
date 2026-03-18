# MTG Deck Builder

A full-stack web application for opening Magic: The Gathering booster packs, building decks, trading cards, and participating in a player-driven market.

## Features

- **Booster Packs** — Open packs to collect cards
- **Card Collection** — Browse and manage your owned cards, recycle duplicates
- **Deck Builder** — Create and edit decks from your collection
- **Market** — Bid on cards listed by other players
- **Trading** — Send and receive trade offers with other players
- **Admin Panel** — Manage card sets, banlists, and user roles (admin only)

## Tech Stack

**Frontend**
- React 19 + TypeScript
- Vite (dev server on port 5200)
- TailwindCSS
- React Router v6
- Axios

**Backend**
- Go + Gin
- MongoDB
- JWT authentication

## Development

### Prerequisites

- Node 20+
- Go 1.25+
- MongoDB running locally on `mongodb://localhost:27017`

### Backend

```bash
cd backend
go run .
# Runs on http://localhost:8090
```

### Frontend

```bash
cd frontend
npm install
npm run dev
# Runs on http://localhost:5200
```

### Environment variables

**Backend** (all optional in dev, required in prod):

| Variable | Default | Description |
|---|---|---|
| `ENV` | — | Set to `prod` to enable production mode |
| `PORT` | `8090` | Server port |
| `DB_URI` | `mongodb://localhost:27017` | MongoDB URI (used when `ENV=prod`) |
| `JWT_SECRET` | dev default | JWT signing secret |
| `ALLOWED_ORIGINS` | localhost ports | Comma-separated list of allowed CORS origins |

**Frontend** (build-time):

| Variable | Default | Description |
|---|---|---|
| `VITE_API_URL` | `http://localhost:8090/api` | Backend API base URL |

## Deployment

The project ships with a GitHub Actions workflow ([.github/workflows/deploy.yml](.github/workflows/deploy.yml)) that:

1. Builds and pushes Docker images for backend and frontend to Docker Hub on every push to `main`
2. SSHs into the server and restarts both containers

### Required GitHub Secrets

| Secret | Description |
|---|---|
| `DOCKERHUB_USERNAME` | Docker Hub username |
| `DOCKERHUB_TOKEN` | Docker Hub access token |
| `SSH_HOST` | Server IP or hostname |
| `SSH_USERNAME` | SSH user |
| `SSH_PRIVATE_KEY` | SSH private key |
| `DOCKER_NETWORK` | Docker network name shared with the reverse proxy and MongoDB |
| `JWT_SECRET` | JWT signing secret |
| `DB_URI` | MongoDB URI (e.g. `mongodb://mongo:27017/mtgdeckbuilder`) |
| `ALLOWED_ORIGINS` | Comma-separated frontend origins (e.g. `https://yourdomain.com`) |
| `VITE_API_URL` | Backend API URL baked into the frontend build (e.g. `https://api.yourdomain.com/api`) |

### Ports

| Container | Internal | External |
|---|---|---|
| `mtgdeckbuilder-backend` | 8090 | 8090 |
| `mtgdeckbuilder-frontend` | 80 | 3001 |
