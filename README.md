
# Golang User Management API

A RESTful User Management API built with Go, Fiber, MongoDB, and JWT authentication. The project demonstrates authentication, user CRUD operations, repository abstraction, request logging, background processing, and graceful shutdown.

## Features

- User registration and login
- Password hashing using bcrypt
- JWT authentication using HS256
- Protected API endpoints
- Create, retrieve, list, update, and delete users
- MongoDB integration using the official Go MongoDB driver
- Unique email index
- Input validation
- HTTP request logging middleware
- Background goroutine that logs the total user count every 10 seconds
- Graceful shutdown with context cancellation
- Unit tests using Go's standard `testing` package and a fake repository

## Tech Stack

- **Language:** Go
- **HTTP Framework:** Fiber v2
- **Database:** MongoDB
- **Database Driver:** Official MongoDB Go Driver
- **Authentication:** JWT (HS256)
- **Password Hashing:** bcrypt
- **Configuration:** Environment variables and `.env`

## Project Structure

```text
.
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── mongodb.go
│   ├── domain/
│   │   └── user.go
│   ├── handler/
│   │   ├── health_handler.go
│   │   └── user_handler.go
│   ├── middleware/
│   │   ├── auth.go
│   │   └── logger.go
│   ├── repository/
│   │   ├── user_repository.go
│   │   └── mongo_user_repository.go
│   ├── routes/
│   │   └── routes.go
│   └── service/
│       └── user_service.go
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

The test files are located alongside the packages they test.

## Prerequisites

Install the following before running the application:

- Go 1.22 or later
- MongoDB running locally or a reachable MongoDB instance
- Git

## Installation

Clone the repository:

```bash
git clone https://github.com/poom7087/golang-user-management-api.git
cd golang-user-management-api
```

Download dependencies:

```bash
go mod download
```

Create a `.env` file in the project root:

```env
PORT=3000
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=user_management
JWT_SECRET=replace-with-a-strong-random-secret
JWT_EXPIRES_IN=24
```

Configuration:

| Variable | Description |
|---|---|
| `PORT` | HTTP server port |
| `MONGODB_URI` | MongoDB connection URI |
| `MONGODB_DATABASE` | Database name |
| `JWT_SECRET` | Secret key used to sign JWTs |
| `JWT_EXPIRES_IN` | Token lifetime in hours |

**Security:** Use a strong, randomly generated JWT secret in production. Never commit `.env` or real credentials to Git.

## Run the Application

Make sure MongoDB is running, then execute:

```bash
go run ./cmd/api
```

The application connects to MongoDB, initializes the unique email index, and starts the HTTP server.

Example startup logs:

```text
MongoDB connected and user indexes initialized
API starting on port 3000
```

The background task logs the total number of users every 10 seconds.

Health check:

```http
GET http://localhost:3000/health
```

Example response:

```json
{
  "status": "ok"
}
```

## API Endpoints

Base URL: `http://localhost:3000`

| Method | Endpoint | Authentication | Description |
|---|---|---|---|
| GET | `/health` | Public | Health check |
| POST | `/api/v1/auth/register` | Public | Register a user |
| POST | `/api/v1/auth/login` | Public | Authenticate a user |
| GET | `/api/v1/me` | JWT required | Validate token and retrieve token subject |
| POST | `/api/v1/users/` | JWT required | Create a user |
| GET | `/api/v1/users/` | JWT required | List all users |
| GET | `/api/v1/users/:id` | JWT required | Retrieve a user by ID |
| PATCH | `/api/v1/users/:id` | JWT required | Update a user's name or email |
| DELETE | `/api/v1/users/:id` | JWT required | Delete a user |

Protected endpoints require a valid JWT in the `Authorization` header.

## Authentication Guide

### 1. Register a user

Request:

```http
POST /api/v1/auth/register
Content-Type: application/json
```

```json
{
  "name": "Test User",
  "email": "test@example.com",
  "password": "Password123!"
}
```

The password is hashed using bcrypt before being stored in MongoDB.

Use a unique email address for each registration.

### 2. Log in

Request:

```http
POST /api/v1/auth/login
Content-Type: application/json
```

```json
{
  "email": "test@example.com",
  "password": "Password123!"
}
```

A successful login returns a JWT token along with the authenticated user's information.

Copy the token from the actual response.

### 3. Use the JWT token

For protected endpoints, add the following header:

```http
Authorization: Bearer <your-jwt-token>
```

In Postman:

1. Open the request.
2. Select **Authorization**.
3. Choose **Bearer Token**.
4. Paste the token returned by Login.
5. Send the request.

The authentication middleware validates the token's signature, signing algorithm, expiration, and subject before allowing access.

A missing, invalid, or expired token should be rejected.

## User Operations

### Create a user

```http
POST /api/v1/users/
Authorization: Bearer <your-jwt-token>
Content-Type: application/json
```

```json
{
  "name": "Another User",
  "email": "another@example.com",
  "password": "Password123!"
}
```

### List users

```http
GET /api/v1/users/
Authorization: Bearer <your-jwt-token>
```

Returns the users stored in MongoDB.

### Retrieve a user by ID

```http
GET /api/v1/users/<user-id>
Authorization: Bearer <your-jwt-token>
```

Replace `<user-id>` with the user's MongoDB ObjectID.

### Update a user's name

```http
PATCH /api/v1/users/<user-id>
Authorization: Bearer <your-jwt-token>
Content-Type: application/json
```

```json
{
  "name": "Updated User"
}
```

To update the email instead:

```json
{
  "email": "updated@example.com"
}
```

Only the fields supplied in the request should be updated.

### Delete a user

```http
DELETE /api/v1/users/<user-id>
Authorization: Bearer <your-jwt-token>
```

The API deletes the specified user from MongoDB.

> Refer to the handler implementation for the exact response envelope and error-message format returned by each endpoint.

## Error Handling

The API handles common error conditions, including:

- Missing or invalid JWT
- Invalid input
- Invalid user ID
- User not found
- Duplicate email
- Database errors

Common HTTP status codes include:

| Status | Meaning |
|---|---|
| `200 OK` | Request completed successfully |
| `201 Created` | Resource created successfully, where applicable |
| `400 Bad Request` | Invalid request or input |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | User does not exist |
| `409 Conflict` | Duplicate email |
| `500 Internal Server Error` | Unexpected server error |

## Logging Middleware

The logging middleware records information about each HTTP request, including:

- HTTP method
- Request path
- Response status code
- Execution duration

This helps with debugging and observing API behavior.

## Background User Count

A background goroutine uses `time.Ticker` to query the total number of users every 10 seconds.

The task:

1. Calls the repository's `Count` method.
2. Logs the total user count.
3. Logs database errors without crashing the application.
4. Stops when its context is cancelled during shutdown.

## Testing

Run all tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

The service unit tests use a fake implementation of the `UserRepository` interface, allowing business logic to be tested without a live MongoDB connection.

Test cases include registration, password hashing, duplicate email, invalid input, login, incorrect passwords, user retrieval, update, delete, and user counting.

## Design Decisions

### Repository abstraction

The `UserRepository` interface separates business logic from MongoDB implementation details. This makes the service easier to test and allows the storage implementation to be replaced without changing the business logic.

### Separation of responsibilities

- **Domain:** User entity and input models
- **Service:** Business logic, validation, password hashing, and token generation
- **Repository:** Data access and MongoDB operations
- **Handler:** HTTP request and response handling
- **Middleware:** Authentication and request logging
- **Routes:** Endpoint registration
- **Database:** MongoDB connection setup

### Password security

Passwords are stored as bcrypt hashes rather than plaintext. The password field is excluded from JSON responses.

### Email uniqueness

A unique MongoDB index prevents duplicate email addresses at the database level, providing protection against concurrent registration attempts.

### JWT authentication

JWTs are signed with HS256 using a secret key supplied through configuration. Tokens include a subject identifying the user and an expiration time.

### Graceful shutdown

The application listens for operating-system shutdown signals, stops the background task, shuts down the HTTP server, and disconnects MongoDB using a bounded context.

### Background processing

A separate goroutine periodically counts users without blocking HTTP request handling.

## Assumptions and Limitations

- User IDs are MongoDB ObjectIDs.
- Email addresses are normalized by the service before registration.
- JWT expiration is configured in hours.
- Protected routes require a valid Bearer token.
- The current authentication design validates the JWT and its subject; it does not imply role-based authorization.
- The `/api/v1/me` endpoint returns the user ID associated with the token rather than necessarily fetching the complete user record.

## License

This project was created as a user management API coding assessment.
