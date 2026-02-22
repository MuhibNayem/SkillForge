# AI Service — SkillForge Phase 2

FastAPI service providing LLM-powered features using **Z.AI GLM-4.7-Flash** via LangChain.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/ai/recommendations` | Personalized course suggestions |
| POST | `/ai/content/tag` | Auto-tag content with categories & difficulty |
| POST | `/ai/content/summarize` | Generate course description & learning objectives |
| GET | `/health` | Health check |

## Setup

```bash
pip install -r requirements.txt
export ZAI_API_KEY=your_key_here
uvicorn app.main:app --reload --port 8080
```

## Testing

```bash
pytest
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ZAI_API_KEY` | *(required)* | Z.AI API key from https://z.ai/manage-apikey/apikey-list |
| `PORT` | `8080` | HTTP port |
