"""POST /ai/content/tag — Auto-tag content with categories & difficulty."""

from __future__ import annotations

import json
from typing import List

from fastapi import APIRouter
from langchain.prompts import ChatPromptTemplate
from pydantic import BaseModel

from ..llm import get_llm

router = APIRouter()


class TagContentRequest(BaseModel):
    content_text: str
    content_title: str


class TagContentResponse(BaseModel):
    categories: List[str]
    difficulty_level: str
    keywords: List[str]


_PROMPT = ChatPromptTemplate.from_messages(
    [
        (
            "system",
            (
                "You are a content classification expert for an e-learning platform. "
                "Analyse the provided content and return ONLY valid JSON with keys: "
                '"categories" (array of strings), '
                '"difficulty_level" (one of: beginner, intermediate, advanced), '
                '"keywords" (array of up to 10 strings). '
                "No extra text, no markdown fences."
            ),
        ),
        (
            "human",
            "Title: {title}\n\nContent excerpt:\n{text}",
        ),
    ]
)


@router.post("/content/tag", response_model=TagContentResponse)
async def tag_content(req: TagContentRequest) -> TagContentResponse:
    llm = get_llm()
    chain = _PROMPT | llm

    raw = await chain.ainvoke(
        {
            "title": req.content_title,
            "text": req.content_text[:2000],  # cap to avoid excessive tokens
        }
    )

    try:
        data = json.loads(raw.content)
        return TagContentResponse(
            categories=data.get("categories", []),
            difficulty_level=data.get("difficulty_level", "beginner"),
            keywords=data.get("keywords", []),
        )
    except (json.JSONDecodeError, TypeError):
        return TagContentResponse(
            categories=["general"],
            difficulty_level="beginner",
            keywords=[],
        )
