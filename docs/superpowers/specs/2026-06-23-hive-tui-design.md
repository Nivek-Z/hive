# Hive TUI Design

## Goal

Build a Go terminal front end for the existing Hive real-time chat system. The first version focuses on the core chat path: log in, choose a hive, choose a text channel, read message history, receive live WebSocket updates, and send text messages from the terminal.

The TUI should feel simple, keyboard-first, and terminal-native, similar in spirit to Claude Code: dense enough to be useful, restrained visually, and clear about current focus, selected channel, connection state, and errors.

## Scope

Version A includes:

- A new `tui/` directory at the repository root, parallel to the existing `hive/` Spring Boot app.
- A standalone Go module for the terminal client.
- A config file at `tui/config.toml`.
- Configurable remote server address, defaulting to `localhost:8080`.
- Login with username and password through `POST /api/auth/login`.
- JWT storage in process memory for the current run.
- Fetching current hives through `GET /api/hives`.
- Fetching hive details through `GET /api/hives/{id}`.
- Rendering the hive channel tree from the flat `channels` list.
- Loading message history through `GET /api/channels/{channelId}/messages?limit=50`.
- Connecting to `ws://<server>/ws?token=<JWT>`.
- Handling `READY`, `MSG_NEW`, `MSG_DELETED`, `ERROR`, `PONG`, and basic reconnect status.
- Sending text messages through WebSocket `MSG_SEND`.
- Sending `PING` heartbeats.
- Marking the current channel as read with `POST /api/channels/{channelId}/read` after loading or receiving messages.

## Non-Goals

Version A will not include private messages, friends, file upload, reactions, replies, message deletion commands, hive creation, invite joining, channel management, member management, role management, search, achievements, stats, or terminal image previews.

Slash commands are allowed because they are plain text sent through `MSG_SEND`, but the TUI will not provide command-specific UI.

## Approach

Use the Go Bubble Tea ecosystem:

- `github.com/charmbracelet/bubbletea` for the update loop.
- `github.com/charmbracelet/lipgloss` for restrained terminal styling.
- `github.com/gorilla/websocket` for WebSocket client support.
- `github.com/pelletier/go-toml/v2` for config parsing.

This gives enough control for a polished terminal chat layout without hand-writing raw terminal event handling. It also keeps the first version smaller than a full widget framework while still supporting keyboard navigation, focus, resizing, and testable state transitions.

## Directory Layout

```text
tui/
  go.mod
  go.sum
  config.toml
  cmd/hive-tui/main.go
  internal/app/
  internal/api/
  internal/config/
  internal/model/
  internal/ui/
  internal/ws/
```

Responsibilities:

- `config`: load and validate `config.toml`.
- `model`: DTO structs matching the existing REST and WebSocket payloads.
- `api`: REST client, API response envelope decoding, bearer auth, URL building.
- `ws`: WebSocket connect, heartbeat, read loop, write messages, reconnect notifications.
- `app`: Bubble Tea model, state transitions, commands, current hive/channel/message state.
- `ui`: rendering helpers and layout styles.
- `cmd/hive-tui`: executable entrypoint.

## Configuration

`tui/config.toml`:

```toml
server_url = "localhost:8080"
```

Rules:

- `server_url` may include or omit scheme.
- If no scheme is present, REST uses `http://` and WebSocket uses `ws://`.
- If `https://` is present, WebSocket uses `wss://`.
- The config loader trims trailing slashes.
- Missing config falls back to `localhost:8080`, but a sample config file remains in the repo for discoverability.

## User Experience

The default app surface has three regions:

```text
+--------------------+-- # channel ----------------------------+
| Hive               | afeng  10:24  hello                     |
|                    | xiaomi 10:25  received                  |
| > # general        |                                         |
|   # homework       |                                         |
|   # random         |                                         |
+--------------------+-----------------------------------------+
| > type message here                                          |
+--------------------------------------------------------------+
| connected | Up/Down move or scroll | Left/Right focus        |
+--------------------------------------------------------------+
```

Login appears first when no token is available:

- Username field.
- Password field.
- Server address shown from config for clarity.
- Login errors shown in the status line.

After login:

- Left pane shows hives and text channels. Categories are labels; text channels are selectable.
- Right pane shows current channel messages in chronological order.
- Bottom input is used for message composition when focus is in the composer.
- Status line shows connection state and the most important key hints.

## Keyboard Interaction

Core keys:

- `Up` and `Down`: move selection in the channel pane or scroll messages, depending on focus.
- `Left` and `Right`: switch focus between navigation, messages, and composer.
- `Enter`: select a highlighted channel, submit login, or send the current message.
- `Esc`: move focus back to navigation or clear transient status.
- `Ctrl+C`: quit.

Text input:

- Printable characters edit the active input.
- `Backspace` edits the active input.
- Empty messages are ignored.
- Multi-line compose is not part of Version A.

## Data Flow

Startup:

1. Load config.
2. Render login view.
3. Login through `POST /api/auth/login`.
4. Store token in memory.
5. Fetch `GET /api/hives`.
6. Select the first hive if available.
7. Fetch `GET /api/hives/{id}`.
8. Select the first `TEXT` channel if available.
9. Fetch recent messages.
10. Connect WebSocket.

Channel selection:

1. Update current channel.
2. Clear current messages.
3. Fetch recent history.
4. Mark read using the newest loaded message id.

Message sending:

1. Ignore empty input.
2. Generate a nonce.
3. Send WebSocket frame `{type:"MSG_SEND", data:{channelId, content, type:"TEXT", nonce}}`.
4. Keep the input responsive while waiting for the server echo.
5. When `MSG_NEW` arrives, append the message if it belongs to the current channel.

Incoming messages:

1. Parse WebSocket envelope.
2. For `MSG_NEW`, append if the message belongs to the current channel.
3. For messages in other channels, increment local unread state in the channel tree.
4. For `MSG_DELETED`, remove the message from the visible list if present.
5. For `ERROR`, display the server error in the status line.

## Error Handling

REST errors:

- Decode the existing `{code,msg,data}` envelope.
- Treat non-zero `code` as a user-visible error.
- Treat HTTP 401 as an expired or invalid login and return to the login view.

WebSocket errors:

- Show `connecting`, `connected`, `disconnected`, and `reconnecting` states.
- Keep the current message history visible during reconnect.
- Reconnect with bounded backoff.
- Keep heartbeats separate from user messages.

Rendering errors:

- Invalid or unknown WebSocket frames are ignored after updating a short status message.
- Unknown message types render as text with their raw content when possible.

## Testing

Use Go tests for logic that does not require a real terminal:

- Config parsing and default values.
- Server URL normalization and REST/WS scheme derivation.
- API envelope decoding for success, business error, and unauthorized cases.
- Channel tree filtering and display ordering for `CATEGORY` and `TEXT`.
- WebSocket frame encode/decode for `MSG_SEND`, `READY`, `MSG_NEW`, `MSG_DELETED`, `ERROR`, and `PONG`.
- App state reducers for login success, channel selection, incoming message append, unread increment, and message deletion.

Manual verification:

- Start the existing Hive backend at `localhost:8080`.
- Run the TUI.
- Log in with demo account `afeng` / `123456`.
- Select a text channel.
- Confirm history loads.
- Send a message.
- Confirm the same message appears in the browser client and the TUI receives browser-sent messages in real time.

## Acceptance Criteria

- `go test ./...` passes under `tui/`.
- `go run ./cmd/hive-tui` starts the terminal app.
- The client works against the existing backend without server changes.
- A user can complete the core chat flow using mostly arrow keys and Enter.
- The UI remains usable in a normal terminal size and shows a clear status for server errors and WebSocket reconnects.
- The first version's scope remains limited to the core chat path described above.
