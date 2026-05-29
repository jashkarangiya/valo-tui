"""Entrypoint: ``python -m valtui``."""

from .app import ValTUI


def main() -> None:
    ValTUI().run()


if __name__ == "__main__":
    main()
