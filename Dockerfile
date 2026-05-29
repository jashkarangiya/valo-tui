FROM python:3.12-slim

WORKDIR /app

# Install dependencies first for layer caching.
COPY pyproject.toml README.md ./
COPY valo_tui ./valo_tui
RUN pip install --no-cache-dir ".[serve]"

# Copy the worker and serve entrypoints.
COPY worker ./worker
COPY serve ./serve

# Shared cache volume populated by the worker, read by the TUI.
ENV VALO_TUI_DB=/var/lib/valo-tui/cache.db

CMD ["python", "-m", "valo_tui"]
