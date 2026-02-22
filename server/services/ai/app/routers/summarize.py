"""POST /ai/content/summarize — Generate course description & learning objectives."""

from __future__ import annotations

import json
from typing import List

from fastapi import APIRouter
from langchain.prompts import ChatPromptTemplate
from pydantic import BaseModel

from ..llm import get_llm

router = APIRouter()


class SummarizeCourseRequest(BaseModel):
    course_title: str
    lesson_titles: List[str] = []
    instructor_notes: str = ""


class SummarizeCourseResponse(BaseModel):
    summary: str
    learning_objectives: str


_PROMPT = ChatPromptTemplate.from_messages(
    [
        (
            "system",
            (
                "You are a curriculum designer helping instructors write compelling course descriptions. "
                "Given a course title, list of lessons, and instructor notes, produce: "
                "1) A 2–3 sentence course description for prospective students. "
                "2) 3–5 measurable learning objectives (start each with an action verb). "
                "Return ONLY valid JSON with keys: "
                '"summary" (string), "learning_objectives" (string with newline-separated objectives). '
                "No extra text, no markdown fences."
            ),
        ),
        (
            "human",
            (
                "Course title: {title}\n"
                "Lessons: {lessons}\n"
                "Instructor notes: {notes}"
            ),
        ),
    ]
)


@router.post("/content/summarize", response_model=SummarizeCourseResponse)
async def summarize_course(req: SummarizeCourseRequest) -> SummarizeCourseResponse:
    llm = get_llm()
    chain = _PROMPT | llm

    raw = await chain.ainvoke(
        {
            "title": req.course_title,
            "lessons": ", ".join(req.lesson_titles) or "not specified",
            "notes": req.instructor_notes or "none",
        }
    )

    try:
        data = json.loads(raw.content)
        return SummarizeCourseResponse(
            summary=data.get("summary", ""),
            learning_objectives=data.get("learning_objectives", ""),
        )
    except (json.JSONDecodeError, TypeError):
        # If the model didn't return JSON, use the raw text as the summary.
        return SummarizeCourseResponse(
            summary=raw.content,
            learning_objectives="",
        )
