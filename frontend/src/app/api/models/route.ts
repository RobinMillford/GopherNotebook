import { NextResponse } from 'next/server';

export async function POST(req: Request) {
  try {
    const { provider, apiKey } = await req.json();

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
      const data = await res.json();
      
      const models = data.data
        .filter((m: any) => m.id.startsWith('gpt-') || m.id.startsWith('o1-'))
        .sort((a: any, b: any) => b.created - a.created) // Newest first
        .map((m: any) => ({ id: m.id, name: m.id }));
        
      return NextResponse.json({ models });
    }

    if (provider === 'google') {
      const res = await fetch(`https://generativelanguage.googleapis.com/v1beta/models?key=${apiKey}`);
      if (!res.ok) throw new Error('Invalid Google API Key');
      const data = await res.json();
      
      const models = data.models
        .filter((m: any) => m.name.includes('gemini'))
        .map((m: any) => ({ 
          id: m.name.replace('models/', ''), 
          name: m.displayName || m.name.replace('models/', '') 
        }));
        
      return NextResponse.json({ models });
    }

    if (provider === 'anthropic') {
      // Anthropic does have a v1/models endpoint, but we need to pass strict headers
      const res = await fetch('https://api.anthropic.com/v1/models', {
        headers: {
          'x-api-key': apiKey,
          'anthropic-version': '2023-06-01',
        },
      });
      
      if (!res.ok) {
        // Fallback to hardcoded list if their endpoint is unavailable or errors out
        return NextResponse.json({
          models: [
            { id: "claude-3-7-sonnet-latest", name: "Claude 3.7 Sonnet" },
            { id: "claude-3-5-sonnet-latest", name: "Claude 3.5 Sonnet" },
            { id: "claude-3-5-haiku-latest", name: "Claude 3.5 Haiku" },
            { id: "claude-3-opus-latest", name: "Claude 3 Opus" },
          ]
        });
      }
      
      const data = await res.json();
      const models = data.data
        .filter((m: any) => m.type === 'model')
        .map((m: any) => ({ id: m.id, name: m.display_name || m.id }));
        
      return NextResponse.json({ models });
    }

    if (provider === 'groq') {
      const res = await fetch('https://api.groq.com/openai/v1/models', {
        headers: { Authorization: `Bearer ${apiKey}` },
      });
      if (!res.ok) throw new Error('Invalid Groq API Key');
      const data = await res.json();
      const models = data.data
        .filter((m: any) => !m.id.includes('whisper'))
        .sort((a: any, b: any) => (b.created || 0) - (a.created || 0))
        .map((m: any) => ({ id: m.id, name: m.id }));
      return NextResponse.json({ models });
    }

    if (provider === 'openrouter') {
      const res = await fetch('https://openrouter.ai/api/v1/models', {
        headers: { Authorization: `Bearer ${apiKey}` },
      });
      if (!res.ok) throw new Error('Invalid OpenRouter API Key');
      const data = await res.json();
      const models = data.data
        .sort((a: any, b: any) => (b.created || 0) - (a.created || 0))
        .map((m: any) => ({ id: m.id, name: m.name || m.id }));
      return NextResponse.json({ models });
    }

    if (provider === 'ollama') {
      try {
        const res = await fetch('http://localhost:11434/api/tags');
        if (!res.ok) throw new Error('not_running');
        const data = await res.json();
        if (!data.models || data.models.length === 0) {
          return NextResponse.json({ models: [], hint: 'no_models' });
        }
        const models = data.models.map((m: any) => ({
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
        const data = await res.json();
        const models = (data.data || []).map((m: any) => ({ id: m.id, name: m.id }));
        return NextResponse.json({ models });
      } catch {
        return NextResponse.json({ models: [], hint: 'not_running' });
      }
    }

    return NextResponse.json({ error: 'Unknown provider' }, { status: 400 });


  } catch (error: any) {
    return NextResponse.json({ error: error.message }, { status: 500 });
  }
}
