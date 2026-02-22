"""Tests for the AI Service — all LLM calls are mocked."""

from __future__ import annotations

import json
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)


def _make_llm_response(content: str) -> MagicMock:
    """Build a mock AIMessage returned by ChatOpenAI."""
    msg = MagicMock()
    msg.content = content
    return msg


def _make_chain_mock(content: str) -> MagicMock:
    """Return a mock chain whose ainvoke resolves to a message with `content`."""
    chain = MagicMock()
    chain.ainvoke = AsyncMock(return_value=_make_llm_response(content))
    return chain


# ─── Health ────────────────────────────────────────────────────────────────────

def test_health():
    resp = client.get("/health")
    assert resp.status_code == 200
    assert resp.json()["status"] == "ok"


# ─── Recommendations ───────────────────────────────────────────────────────────

RECO_PAYLOAD = json.dumps([
    {"title": "Advanced Python", "match_reason": "You completed basics", "confidence_score": 0.9},
    {"title": "Data Science", "match_reason": "Matches skills", "confidence_score": 0.8},
])


def test_recommendations_success():
    """Verify recommendations endpoint returns 200 with mocked LLM chain."""
    chain = _make_chain_mock(RECO_PAYLOAD)
    # Patch _PROMPT.__or__ so that `_PROMPT | llm` returns our mock chain.
    with patch("app.routers.recommendations._PROMPT") as mock_prompt:
        mock_prompt.__or__ = MagicMock(return_value=chain)
        resp = client.post("/ai/recommendations", json={
            "user_id": "u1",
            "tenant_id": "t1",
            "completed_course_ids": ["intro-python"],
            "skills_of_interest": ["data science"],
            "limit": 2,
        })
    assert resp.status_code == 200
    data = resp.json()
    assert data["user_id"] == "u1"
    assert len(data["recommendations"]) == 2
    assert data["recommendations"][0]["title"] == "Advanced Python"


def test_recommendations_endpoint_exists():
    """Verify the route is registered (no 404) with mocked LLM."""
    chain = _make_chain_mock(RECO_PAYLOAD)
    with patch("app.routers.recommendations._PROMPT") as mock_prompt:
        mock_prompt.__or__ = MagicMock(return_value=chain)
        resp = client.post("/ai/recommendations", json={
            "user_id": "u1", "tenant_id": "t1"
        })
    assert resp.status_code == 200


# ─── Tagging ───────────────────────────────────────────────────────────────────

TAGGING_PAYLOAD = json.dumps({
    "categories": ["programming", "python"],
    "difficulty_level": "beginner",
    "keywords": ["python", "variables", "loops"],
})


def test_tagging_endpoint_exists():
    chain = _make_chain_mock(TAGGING_PAYLOAD)
    with patch("app.routers.tagging._PROMPT") as mock_prompt:
        mock_prompt.__or__ = MagicMock(return_value=chain)
        resp = client.post("/ai/content/tag", json={
            "content_text": "This course covers Python fundamentals.",
            "content_title": "Intro to Python",
        })
    assert resp.status_code == 200
    data = resp.json()
    assert data["difficulty_level"] == "beginner"
    assert "python" in data["categories"]


# ─── Summarize ─────────────────────────────────────────────────────────────────

SUMMARIZE_PAYLOAD = json.dumps({
    "summary": "Learn ML from scratch with practical examples.",
    "learning_objectives": "Understand supervised learning\nApply model evaluation techniques",
})


def test_summarize_endpoint_exists():
    chain = _make_chain_mock(SUMMARIZE_PAYLOAD)
    with patch("app.routers.summarize._PROMPT") as mock_prompt:
        mock_prompt.__or__ = MagicMock(return_value=chain)
        resp = client.post("/ai/content/summarize", json={
            "course_title": "Machine Learning Fundamentals",
            "lesson_titles": ["Intro to ML", "Supervised Learning", "Model Evaluation"],
            "instructor_notes": "Focus on practical examples.",
        })
    assert resp.status_code == 200
    data = resp.json()
    assert "ML" in data["summary"]


# ─── Input validation ──────────────────────────────────────────────────────────

def test_recommendations_missing_required_fields():
    resp = client.post("/ai/recommendations", json={})
    assert resp.status_code == 422  # Pydantic validation error


def test_tagging_missing_fields():
    resp = client.post("/ai/content/tag", json={})
    assert resp.status_code == 422


def test_summarize_missing_fields():
    resp = client.post("/ai/content/summarize", json={})
    assert resp.status_code == 422
