"""Web server — serves the TUI to a browser via textual-serve."""

from textual_serve.server import Server

server = Server(
    command="python -m valtui",
    host="0.0.0.0",
    port=8000,
    title="valtui · Valorant esports in your terminal",
)

if __name__ == "__main__":
    server.serve()
