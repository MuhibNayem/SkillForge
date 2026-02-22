"""
AI Service — SkillForge Phase 2
FastAPI application exposing three AI-powered endpoints via Z.AI GLM-4.7-Flash.

Endpoints:
  POST /ai/recommendations   — Personalized course recommendations
  POST /ai/content/tag       — Auto-tag content with categories & difficulty
  POST /ai/content/summarize — Generate course description & learning objectives
"""

from __future__ import annotations

import os

from dotenv import load_dotenv
from fastapi import FastAPI

# Load .env when running locally (no-op in Docker where env vars are injected)
load_dotenv()
from fastapi.middleware.cors import CORSMiddleware

from .routers import recommendations, tagging, summarize

app = FastAPI(
    title="SkillForge AI Service",
    description="LLM-powered features using Z.AI GLM-4.7-Flash via LangChain",
    version="1.0.0",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(recommendations.router, prefix="/ai", tags=["recommendations"])
app.include_router(tagging.router, prefix="/ai", tags=["tagging"])
app.include_router(summarize.router, prefix="/ai", tags=["summarize"])


@app.get("/health")
async def health() -> dict:
    return {"status": "ok", "service": "ai-service"}
