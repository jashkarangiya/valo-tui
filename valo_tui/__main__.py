"""Entrypoint: ``python -m valo_tui``."""

from .app import ValoTUI


def main() -> None:
    ValoTUI().run()


if __name__ == "__main__":
    main()
