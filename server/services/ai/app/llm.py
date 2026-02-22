"""Shared LLM client — Z.AI GLM-4.7-Flash via langchain-openai."""

from __future__ import annotations

import os

from langchain_openai import ChatOpenAI


def get_llm(streaming: bool = False) -> ChatOpenAI:
    """Return a ChatOpenAI client pointed at the Z.AI endpoint.

    The API key is read from the ZAI_API_KEY environment variable.
    Falls back gracefully if the key is missing (useful for unit tests
    that mock the LLM calls).
    """
    return ChatOpenAI(
        model="glm-4.7-flash",
        temperature=0.7,
        openai_api_key=os.getenv("ZAI_API_KEY", "dummy-key-for-tests"),
        openai_api_base="https://api.z.ai/api/paas/v4/",
        streaming=streaming,
        # Reasonable limits to avoid runaway token usage.
        max_tokens=1024,
        request_timeout=30,
    )
