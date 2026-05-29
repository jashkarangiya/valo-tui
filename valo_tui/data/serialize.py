"""Turn ``vlrdevapi`` dataclass objects into JSON-safe primitives.

The library returns nested dataclasses containing ``date``/``time`` objects,
tuples, and enums. ``dataclasses.asdict`` flattens the nesting; the JSON
encoder default below handles the leaf types that ``json`` cannot.
"""

from __future__ import annotations

import dataclasses
import datetime as dt
import enum
import json
from typing import Any


def _default(obj: Any) -> Any:
    if isinstance(obj, (dt.date, dt.time, dt.datetime)):
        return obj.isoformat()
    if isinstance(obj, enum.Enum):
        return obj.value
    if dataclasses.is_dataclass(obj):
        return dataclasses.asdict(obj)
    if isinstance(obj, (set, frozenset)):
        return list(obj)
    raise TypeError(f"cannot serialize {type(obj).__name__}")


def to_jsonable(obj: Any) -> Any:
    """Recursively convert a (possibly nested) dataclass into primitives."""
    if dataclasses.is_dataclass(obj) and not isinstance(obj, type):
        obj = dataclasses.asdict(obj)
    return json.loads(json.dumps(obj, default=_default))


def dumps(obj: Any) -> str:
    """Serialize a list/dict of dataclasses to a JSON string."""
    return json.dumps(obj, default=_default)
