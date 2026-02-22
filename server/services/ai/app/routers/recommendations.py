"""POST /ai/recommendations — Personalized course suggestions."""

from __future__ import annotations

import json
from typing import List

from fastapi import APIRouter
from langchain.prompts import ChatPromptTemplate
from pydantic import BaseModel

from ..llm import get_llm

router = APIRouter()


class RecommendationRequest(BaseModel):
    user_id: str
    tenant_id: str
    completed_course_ids: List[str] = []
    skills_of_interest: List[str] = []
    limit: int = 5


class RecommendedCourse(BaseModel):
    title: str
    match_reason: str
    confidence_score: float


class RecommendationResponse(BaseModel):
    user_id: str
    recommendations: List[RecommendedCourse]


_PROMPT = ChatPromptTemplate.from_messages(
    [
        (
            "system",
            (
                "You are a personalized learning assistant for an online education platform. "
                "Given a learner's history and interests, suggest relevant next courses. "
                "Return ONLY a JSON array of objects with keys: "
                '"title", "match_reason", "confidence_score" (0.0–1.0). '
                "No extra text, no markdown fences."
            ),
        ),
        (
            "human",
            (
                "Completed courses: {completed}\n"
                "Skills of interest: {skills}\n"
                "Please suggest {limit} courses."
            ),
        ),
    ]
)


@router.post("/recommendations", response_model=RecommendationResponse)
async def recommend_courses(req: RecommendationRequest) -> RecommendationResponse:
    llm = get_llm()
    chain = _PROMPT | llm

    raw = await chain.ainvoke(
        {
            "completed": ", ".join(req.completed_course_ids) or "none",
            "skills": ", ".join(req.skills_of_interest) or "general",
            "limit": req.limit,
        }
    )

    try:
        items = json.loads(raw.content)
        recommendations = [
            RecommendedCourse(
                title=item.get("title", ""),
                match_reason=item.get("match_reason", ""),
                confidence_score=float(item.get("confidence_score", 0.5)),
            )
            for item in items
        ]
    except (json.JSONDecodeError, TypeError):
        # Graceful degradation — return the raw text as a single entry.
        recommendations = [
            RecommendedCourse(
                title="See AI suggestion",
                match_reason=raw.content,
                confidence_score=0.5,
            )
        ]

    return RecommendationResponse(user_id=req.user_id, recommendations=recommendations)
