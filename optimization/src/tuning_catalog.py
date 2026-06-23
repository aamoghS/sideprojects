"""Catalog of every tuning approach in this project."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Callable

from src.metrics import ModelResult


@dataclass(frozen=True)
class TuningMethod:
    id: str
    name: str
    category: str
    model: str
    description: str
    why_it_helps: str
    runner: Callable[..., tuple[object, ModelResult, dict]]


# Populated at import time by tune.py to avoid circular imports.
REGISTRY: dict[str, TuningMethod] = {}


def register(method: TuningMethod) -> TuningMethod:
    REGISTRY[method.id] = method
    return method


def list_methods() -> list[TuningMethod]:
    return list(REGISTRY.values())


def print_catalog() -> None:
    methods = list_methods()
    categories: dict[str, list[TuningMethod]] = {}
    for m in methods:
        categories.setdefault(m.category, []).append(m)

    print("\nTuning methods available in this project\n" + "=" * 72)
    for category, items in categories.items():
        print(f"\n{category}")
        print("-" * 72)
        for m in items:
            print(f"  [{m.id}] {m.name} ({m.model})")
            print(f"       {m.description}")
            print(f"       Why it helps: {m.why_it_helps}")
    print("\n" + "=" * 72)
    print("Run one:  python -m src.main tune --method <id>")
    print("Run all:  python -m src.main tune --all-methods")
    print("Default:  python -m src.main tune   (runs the 4 core methods)\n")
