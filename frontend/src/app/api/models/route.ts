import { NextResponse } from 'next/server';

interface ModelItem {
  id: string;
  name: string;
}

// Per-provider response shapes — only the fields we actually use.
interface OpenAIModel {
  id: string;
  created: number;
}

interface GoogleModel {
  name: string;
  displayName?: string;
}

interface AnthropicModel {
  id: string;
  type: string;
  display_name?: string;
}

interface GroqModel {
  id: string;
  created?: number;
}

interface OpenRouterModel {
  id: string;
  name?: string;
  created?: number;
}

interface OllamaModel {
  name: string;
  size: number;
}

interface LMStudioModel {
  id: string;
}

export async function POST(req: Request) {
  try {
    const { provider, apiKey } = await req.json() as { provider?: string; apiKey?: string };

    if (!provider) {
      return NextResponse.json({ error: 'Missing provider' }, { status: 400 });
    }

    if (!apiKey) {
      return NextResponse.json({ error: 'Missing apiKey' }, { status: 400 });
    }

    if (provider === 'openai') {
      const res = await fetch('https://api.openai.com/v1/models', {
        headers: { Authorization: `Bearer ${apiKey}` },
      });
      if (!res.ok) throw new Error('Invalid OpenAI API Key');
      const data = await res.json() as { data: OpenAIModel[] };

      const models: ModelItem[] = data.data
        .filter((m) => m.id.startsWith('gpt-') || m.id.startsWith('o1-'))
        .sort((a, b) => b.created - a.created)
        .map((m) => ({ id: m.id, name: m.id }));

      return NextResponse.json({ models });
    }

    if (provider === 'google') {
      const res = await fetch(`https://generativelanguage.googleapis.com/v1beta/models?key=${apiKey}`);
      if (!res.ok) throw new Error('Invalid Google API Key');
      const data = await res.json() as { models: GoogleModel[] };

      const models: ModelItem[] = data.models
        .filter((m) => m.name.includes('gemini'))
        .map((m) => ({
          id: m.name.replace('models/', ''),
          name: m.displayName ?? m.name.replace('models/', ''),
        }));

      return NextResponse.json({ models });
    }

    if (provider === 'anthropic') {
      const res = await fetch('https://api.anthropic.com/v1/models', {
        headers: {
          'x-api-key': apiKey,
          'anthropic-version': '2023-06-01',
        },
      });

      if (!res.ok) {
        return NextResponse.json({
          models: [
            { id: "claude-3-7-sonnet-latest", name: "Claude 3.7 Sonnet" },
            { id: "claude-3-5-sonnet-latest", name: "Claude 3.5 Sonnet" },
            { id: "claude-3-5-haiku-latest", name: "Claude 3.5 Haiku" },
            { id: "claude-3-opus-latest", name: "Claude 3 Opus" },
          ] satisfies ModelItem[],
        });
      }

      const data = await res.json() as { data: AnthropicModel[] };
      const models: ModelItem[] = data.data
        .filter((m) => m.type === 'model')
        .map((m) => ({ id: m.id, name: m.display_name ?? m.id }));

      return NextResponse.json({ models });
    }

    if (provider === 'groq') {
      const res = await fetch('https://api.groq.com/openai/v1/models', {
        headers: { Authorization: `Bearer ${apiKey}` },
      });
      if (!res.ok) throw new Error('Invalid Groq API Key');
      const data = await res.json() as { data: GroqModel[] };

      const models: ModelItem[] = data.data
        .filter((m) => !m.id.includes('whisper'))
        .sort((a, b) => (b.created ?? 0) - (a.created ?? 0))
        .map((m) => ({ id: m.id, name: m.id }));

      return NextResponse.json({ models });
    }

    if (provider === 'openrouter') {
      const res = await fetch('https://openrouter.ai/api/v1/models', {
        headers: { Authorization: `Bearer ${apiKey}` },
      });
      if (!res.ok) throw new Error('Invalid OpenRouter API Key');
      const data = await res.json() as { data: OpenRouterModel[] };

      const models: ModelItem[] = data.data
        .sort((a, b) => (b.created ?? 0) - (a.created ?? 0))
        .map((m) => ({ id: m.id, name: m.name ?? m.id }));

      return NextResponse.json({ models });
    }

    if (provider === 'ollama') {
      try {
        const res = await fetch('http://localhost:11434/api/tags');
        if (!res.ok) throw new Error('not_running');
        const data = await res.json() as { models?: OllamaModel[] };
        if (!data.models || data.models.length === 0) {
          return NextResponse.json({ models: [], hint: 'no_models' });
        }
        const models: ModelItem[] = data.models.map((m) => ({
          id: m.name,
          name: `${m.name} (${(m.size / 1e9).toFixed(1)} GB)`,
        }));
        return NextResponse.json({ models });
      } catch {
        return NextResponse.json({ models: [], hint: 'not_running' });
      }
    }

    if (provider === 'lmstudio') {
      try {
        const res = await fetch('http://localhost:1234/v1/models');
        if (!res.ok) throw new Error('not_running');
        const data = await res.json() as { data?: LMStudioModel[] };
        const models: ModelItem[] = (data.data ?? []).map((m) => ({ id: m.id, name: m.id }));
        return NextResponse.json({ models });
      } catch {
        return NextResponse.json({ models: [], hint: 'not_running' });
      }
    }

    return NextResponse.json({ error: 'Unknown provider' }, { status: 400 });

  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : 'Internal server error';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
