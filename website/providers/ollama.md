---
title: Ollama Provider
description: Run local LLMs with Ollama
---

# Ollama

Run LLM models locally using Ollama. No API key required — the model runs on your own hardware.

## Setup

1. Install Ollama from [ollama.com](https://ollama.com).
2. Pull a model: `ollama pull llama3.3`.
3. Configure BlackCat:

```bash
blackcat configure --provider ollama --model llama3.3
```

## YAML Configuration

```yaml
llm:
  provider: "ollama"
  model: "llama3.3"
  baseURL: "http://localhost:11434/v1"
```

## Supported Models

Any model available in your Ollama installation. Run `ollama list` to see installed models.

## Tips

- Ollama is ideal for users who want to keep their data on-premises and have the hardware to support local inference.
- Ensure your machine has enough VRAM for the models you're running.
