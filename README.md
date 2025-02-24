# GoBid - Real-time Auction System

GoBid is a real-time auction system built with Go, featuring WebSocket support for live bidding and PostgreSQL for data persistence.

## Features

- Real-time bidding using WebSocket connections
- User authentication and session management
- Product management
- Auction room system
- PostgreSQL database integration

## Prerequisites

- Go 1.20 or higher
- PostgreSQL
- Environment variables setup (see Configuration section)

## Configuration

Create a `.env` file in the root directory with the following variables:

```env
GOBID_DATABASE_USER=your_db_user
GOBID_DATABASE_PASSWORD=your_db_password
GOBID_DATABASE_HOST=localhost
GOBID_DATABASE_PORT=5432
GOBID_DATABASE_NAME=gobid
```

## Setup

1. Clone the repository
2. Set up the environment variables
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Run the application:
   ```bash
   go run cmd/api/main.go
   ```

The server will start on `localhost:3080`.

## Project Structure

- `cmd/api/` - Application entry point
- `internal/api/` - API handlers and routes
- `internal/services/` - Business logic and services
  - User Service
  - Product Service
  - Bids Service
  - Auction Lobby

## Technologies Used

- Chi router for HTTP routing
- pgx for PostgreSQL connection
- gorilla/websocket for WebSocket support
- alexedwards/scs for session management

