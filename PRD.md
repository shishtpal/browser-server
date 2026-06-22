# Browser Server

This is a Go-based REST API server for managing personal data like todos, bookmarks, browsing history, and a password wallet. API routes are served under `/api/` and require the operator API token, except `/health`.

## Prerequisites

- Go installed
- Python's `httpie` command-line tool (`pip install httpie`)

## Setup & Running

1.  **Set Environment Variable (for Windows Powershell):**
    ```powershell
    # User-level
    [System.Environment]::SetEnvironmentVariable("CGO_ENABLED", "1", "User")
    ```

2.  **Build and Run:**
    ```bash
    go build
    ./browser-server
    ```
    The server will start on `http://localhost:8080`.

3.  **Static Frontend:**
    The `GET /` endpoint serves static files from the `frontend/dist/` directory relative to the executable.

## API Endpoints

### Routes

-   **List all available routes**
    ```bash
    http POST http://localhost:8080/api/routes "Authorization: Bearer <token>"
    ```

### Search

-   **Search Chrome omnibox suggestions**

    Returns a mixed list of bookmark and URL-grouped history suggestions. Each item includes `source` (`history` or `bookmark`) so clients can label results. History results include `visit_count`, allowing the extension to show how many times a URL was opened. When both sources match, results are balanced so bookmarks are not hidden behind a full page of history matches.

    ```bash
    http GET "http://localhost:8080/api/search/omnibox?user_id=1&q=golang&limit=6" "Authorization: Bearer <token>"
    ```

### Users

-   **Create a new user**
    ```bash
    http POST http://localhost:8080/users username=john email=john@example.com
    ```

-   **Get all users**
    ```bash
    http GET http://localhost:8080/users
    ```

-   **Get user by ID**
    ```bash
    http GET http://localhost:8080/users/1
    ```

### Todos

-   **Create a new todo**
    ```bash
    http POST http://localhost:8080/todos user_id:=1 title="Learn Go" description="Build a REST API" completed:=false
    ```

-   **Get all todos**
    ```bash
    http GET http://localhost:8080/todos
    ```

-   **Get todos by user**
    ```bash
    http GET http://localhost:8080/todos user_id==1
    ```

-   **Get completed todos by user**
    ```bash
    http GET "http://localhost:8080/todos?user_id=1&completed=true"
    ```

-   **Get todo by ID**
    ```bash
    http GET http://localhost:8080/todos/1
    ```

-   **Update a todo**
    ```bash
    http PUT http://localhost:8080/todos/1 user_id:=1 title="Master Go" description="Build an amazing REST API" completed:=true
    ```

-   **Delete a todo**
    ```bash
    http DELETE http://localhost:8080/todos/1
    ```

### Bookmarks

-   **Create a new bookmark**
    ```bash
    http POST http://localhost:8080/bookmarks user_id:=1 title="Go Documentation" url="https://golang.org" tags:='["programming", "go"]'
    ```

-   **Get all bookmarks**
    ```bash
    http GET http://localhost:8080/bookmarks
    ```

-   **Get bookmarks by user**
    ```bash
    http GET http://localhost:8080/bookmarks user_id==1
    ```

-   **Get bookmarks by tags**
    ```bash
    http GET http://localhost:8080/bookmarks tags==go,programming
    ```

-   **Get bookmark by ID**
    ```bash
    http GET http://localhost:8080/bookmarks/1
    ```

-   **Update a bookmark**
    ```bash
    http PUT http://localhost:8080/bookmarks/1 user_id:=1 title="Official Go Docs" url="https://go.dev/doc/" tags:='["go", "official"]'
    ```

-   **Delete a bookmark**
    ```bash
    http DELETE http://localhost:8080/bookmarks/1
    ```

### History

-   **Add a history entry**
    ```bash
    http POST http://localhost:8080/history user_id:=1 url="https://google.com" title="Google" duration:=30
    ```

-   **Get all history**
    ```bash
    http GET http://localhost:8080/history
    ```

-   **Get history by user**
    ```bash
    http GET http://localhost:8080/history user_id==1
    ```

-   **Filter history by URL**
    ```bash
    http GET http://localhost:8080/history url==google
    ```

-   **Get URL-grouped, searched & paginated history** (server-side; used by the popup so large histories don't load all at once)
    ```bash
    http GET http://localhost:8080/api/history/grouped user_id==1 q==google column==all limit==100 offset==0 "Authorization: Bearer <token>"
    ```

-   **Get history entry by ID**
    ```bash
    http GET http://localhost:8080/history/1
    ```

-   **Delete a history entry**
    ```bash
    http DELETE http://localhost:8080/history/1
    ```

### Wallet

-   **Create a wallet entry**
    ```bash
    http POST http://localhost:8080/wallet user_id:=1 website="github.com" username="testuser" password="secretpassword"
    ```

-   **Get all wallet entries**
    ```bash
    http GET http://localhost:8080/wallet
    ```

-   **Get wallet entries by user**
    ```bash
    http GET http://localhost:8080/wallet user_id==1
    ```

-   **Filter wallet entries by website**
    ```bash
    http GET http://localhost:8080/wallet website==github
    ```

-   **Get wallet entry by ID**
    ```bash
    http GET http://localhost:8080/wallet/1
    ```

-   **Update a wallet entry**
    ```bash
    http PUT http://localhost:8080/wallet/1 user_id:=1 website="github.com" username="newuser" password="newpassword"
    ```

-   **Delete a wallet entry**
    ```bash
    http DELETE http://localhost:8080/wallet/1
    ```
