FROM python:3.12-slim

WORKDIR /app

# Install dependencies first for layer caching.
COPY pyproject.toml README.md ./
COPY valtui ./valtui
RUN pip install --no-cache-dir ".[serve]"

# Copy the worker and serve entrypoints.
COPY worker ./worker
COPY serve ./serve

# Shared cache volume populated by the worker, read by the TUI.
ENV VALTUI_DB=/var/lib/valtui/cache.db

CMD ["python", "-m", "valtui"]
