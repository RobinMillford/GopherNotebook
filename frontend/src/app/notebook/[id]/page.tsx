"use client";

import { useEffect, useRef, useState, use } from "react";
import Link from "next/link";
import { 
  ArrowLeft, Upload, FileText, Settings, Send, Bot, User, 
  Loader2, Trash2, CheckCircle2, AlertCircle, Key, Sparkles, Download, Copy, Check, Eye, EyeOff
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { formatDistanceToNow } from "date-fns";
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import { 
  NotebookDetail, IngestProgress, 
  getNotebook, uploadFiles, API_BASE 
} from "@/lib/api";

type Message = {
  id: string;
  role: "user" | "assistant";
  content: string;
  citations?: { fileName: string; pageNumber: number; snippet: string; index: number }[];
};

const AVAILABLE_MODELS: Record<string, {id: string; name: string}[]> = {
  openai: [
    { id: "gpt-4o", name: "GPT-4o (Default)" },
    { id: "gpt-4o-mini", name: "GPT-4o Mini" },
    { id: "gpt-4-turbo", name: "GPT-4 Turbo" },
    { id: "o1-preview", name: "o1 Preview" },
    { id: "o1-mini", name: "o1 Mini" },
  ],
  google: [
    { id: "gemini-2.5-flash", name: "Gemini 2.5 Flash" },
    { id: "gemini-2.0-pro-exp-02-05", name: "Gemini 2.0 Pro Experimental" },
    { id: "gemini-2.0-flash", name: "Gemini 2.0 Flash (Default)" },
    { id: "gemini-1.5-pro", name: "Gemini 1.5 Pro" },
    { id: "gemini-1.5-flash", name: "Gemini 1.5 Flash" },
  ],
  anthropic: [
    { id: "claude-3-7-sonnet-20250219", name: "Claude 3.7 Sonnet (Default)" },
    { id: "claude-3-5-sonnet-20241022", name: "Claude 3.5 Sonnet" },
    { id: "claude-3-5-haiku-20241022", name: "Claude 3.5 Haiku" },
    { id: "claude-3-opus-20240229", name: "Claude 3 Opus" },
  ],
  groq: [
    { id: "llama-3.3-70b-versatile", name: "Llama 3.3 70B Versatile" },
    { id: "llama-3.1-8b-instant", name: "Llama 3.1 8B Instant" },
    { id: "mixtral-8x7b-32768", name: "Mixtral 8x7B" },
    { id: "gemma2-9b-it", name: "Gemma 2 9B IT" },
  ],
  openrouter: [
    { id: "anthropic/claude-3.7-sonnet", name: "Claude 3.7 Sonnet (OpenRouter)" },
    { id: "anthropic/claude-3.5-sonnet", name: "Claude 3.5 Sonnet (OpenRouter)" },
    { id: "google/gemini-2.5-flash", name: "Gemini 2.5 Flash (OpenRouter)" },
    { id: "meta-llama/llama-3.3-70b-instruct", name: "Llama 3.3 70B (OpenRouter)" },
    { id: "openai/o3-mini", name: "o3 Mini (OpenRouter)" },
  ],
};

export default function NotebookWorkspace({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const notebookId = resolvedParams.id;
  
  const [notebook, setNotebook] = useState<NotebookDetail | null>(null);
  const [loading, setLoading] = useState(true);
  
  // Upload state
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState<IngestProgress | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Chat state
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [chatting, setChatting] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Settings state
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [provider, setProvider] = useState("openai");
  const [apiKey, setApiKey] = useState("");
  const [model, setModel] = useState("");
  
  // Dynamic Models
  const [dynamicModels, setDynamicModels] = useState<{id: string, name: string}[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);
  const [showApiKey, setShowApiKey] = useState(false);
  const [modelSearch, setModelSearch] = useState("");

  useEffect(() => {
    // Load from local storage
    const storedProvider = localStorage.getItem("gn_provider") || "openai";
    const storedKey = localStorage.getItem("gn_api_key") || "";
    const storedModel = localStorage.getItem("gn_model") || "";
    setProvider(storedProvider);
    setApiKey(storedKey);
    setModel(storedModel);
    
    if (storedKey) {
      fetchDynamicModels(storedProvider, storedKey);
    }

    loadData();
    setupSSE();
  }, [notebookId]);

  const fetchDynamicModels = async (currentProvider: string, currentApiKey: string) => {
    if (!currentApiKey.trim()) return;
    setLoadingModels(true);
    try {
      const res = await fetch("/api/models", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider: currentProvider, apiKey: currentApiKey })
      });
      if (res.ok) {
        const data = await res.json();
        if (data && data.models) {
          setDynamicModels(data.models);
        }
      }
    } catch (e) {
      console.error("Failed to load models list", e);
    } finally {
      setLoadingModels(false);
    }
  };

  useEffect(() => {
    // Auto-scroll chat
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const loadData = async () => {
    try {
      const data = await getNotebook(notebookId);
      setNotebook(data);
      if (data && data.messages) {
        setMessages(data.messages);
      }
    } catch (error) {
      toast.error("Failed to load notebook");
    } finally {
      setLoading(false);
    }
  };

  const setupSSE = () => {
    const eventSource = new EventSource(`${API_BASE}/notebooks/${notebookId}/ingest/progress`);
    
    eventSource.addEventListener("progress", (e) => {
      const data = JSON.parse(e.data) as IngestProgress;
      setProgress(data);
      if (data.status === "done") {
        setUploading(false);
        setTimeout(() => setProgress(null), 3000);
        loadData(); // reload sources
        toast.success("Documents ingested successfully");
      }
    });

    eventSource.addEventListener("ping", () => {});
    
    return () => eventSource.close();
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files || e.target.files.length === 0) return;
    
    const files = Array.from(e.target.files);
    setUploading(true);
    
    try {
      await uploadFiles(notebookId, files);
      toast.info(`Started processing ${files.length} files`);
      loadData(); // Update sources list to show "processing"
    } catch (error: any) {
      toast.error(error.message || "Upload failed");
      setUploading(false);
    }
    
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const saveSettings = () => {
    localStorage.setItem("gn_provider", provider);
    localStorage.setItem("gn_api_key", apiKey);
    localStorage.setItem("gn_model", model);
    setSettingsOpen(false);
    toast.success("Settings saved locally");
  };

  const submitChat = async () => {
    if (!input.trim() || chatting) return;
    if (!apiKey) {
      setSettingsOpen(true);
      toast.error("API Key is required to chat");
      return;
    }

    const query = input.trim();
    setInput("");
    
    const newMsgId = Date.now().toString();
    setMessages(prev => [...prev, { id: `u_${newMsgId}`, role: "user", content: query }]);
    setChatting(true);

    const asstMsgId = `a_${newMsgId}`;
    setMessages(prev => [...prev, { id: asstMsgId, role: "assistant", content: "" }]);

    try {
      const res = await fetch(`${API_BASE}/notebooks/${notebookId}/chat`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": apiKey,
          "X-LLM-Provider": provider,
          "X-LLM-Model": model
        },
        body: JSON.stringify({ query }),
      });

      if (!res.ok) throw new Error("Failed to start chat");
      if (!res.body) throw new Error("No response body");

      const reader = res.body.getReader();
      const decoder = new TextDecoder("utf-8");

      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        
        buffer += decoder.decode(value, { stream: true });
        
        const events = buffer.split("\n\n");
        buffer = events.pop() || ""; // Keep the incomplete part

        for (const event of events) {
          if (event.startsWith("event:message\ndata:")) {
            const dataStr = event.substring("event:message\ndata:".length);
            try {
              const data = JSON.parse(dataStr);
              setMessages(prev => prev.map(m => {
                if (m.id === asstMsgId) {
                  return {
                    ...m,
                    content: m.content + (data.content || ""),
                    citations: data.citations ? data.citations : m.citations
                  };
                }
                return m;
              }));
              if (data.done) {
                 // Finalize
              }
            } catch (e) {
              console.error("Parse error", e);
            }
          }
        }
      }
    } catch (error: any) {
      toast.error(error.message || "Chat stream failed");
    } finally {
      setChatting(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-background overflow-hidden font-sans">
      {/* Sidebar */}
      <aside className="w-80 border-r bg-card/30 flex flex-col backdrop-blur-xl shrink-0 transition-all z-10">
        <div className="p-4 border-b flex items-center gap-3">
          <Link href="/dashboard">
            <Button variant="ghost" size="icon" className="shrink-0 -ml-2 hover:bg-black/5 dark:hover:bg-white/10">
              <ArrowLeft className="w-5 h-5" />
            </Button>
          </Link>
          <div className="min-w-0">
            <h2 className="font-semibold truncate pr-2" title={notebook?.name}>{notebook?.name}</h2>
            <p className="text-xs text-muted-foreground truncate">{notebook?.fileCount} sources</p>
          </div>
        </div>

        <div className="p-4 flex-none">
          <Button 
            className="w-full justify-start rounded-xl font-medium shadow-sm transition-all" 
            variant="secondary"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploading}
          >
            <Upload className="w-4 h-4 mr-2 text-primary" />
            {uploading ? "Ingesting..." : "Upload Sources"}
          </Button>
          <input 
            type="file" 
            ref={fileInputRef} 
            className="hidden" 
            multiple 
            accept=".pdf,.docx,.txt,.odt,.xlsx,.pptx,.html,.epub" 
            onChange={handleFileUpload} 
          />
        </div>

        {progress && (
          <div className="px-4 pb-4">
            <div className="bg-primary/10 border border-primary/20 rounded-xl p-3 text-sm">
              <div className="flex justify-between mb-1.5 text-xs font-medium text-primary">
                <span>Ingesting {progress.totalFiles} files</span>
                <span>{progress.processedFiles}/{progress.totalFiles}</span>
              </div>
              <Progress value={(progress.processedFiles / progress.totalFiles) * 100} className="h-1.5 bg-primary/20" />
              <p className="text-[10px] text-muted-foreground mt-2 truncate font-mono">
                {progress.currentFile.split("/").pop()}
              </p>
            </div>
          </div>
        )}

        <ScrollArea className="flex-1 min-h-0 px-2 h-full">
          <div className="space-y-1 p-2 pb-4">
            {notebook?.sources.map((src) => (
              <div 
                key={src.fileName} 
                className="flex items-center gap-3 p-2.5 rounded-lg hover:bg-black/5 dark:hover:bg-white/5 group border border-transparent hover:border-border/50 transition-all text-sm"
              >
                <div className={`p-1.5 rounded-md flex-shrink-0 ${src.status === 'failed' ? 'bg-destructive/10 text-destructive' : 'bg-primary/10 text-primary'}`}>
                  {src.status === 'processing' ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : src.status === 'failed' ? (
                    <AlertCircle className="w-4 h-4" />
                  ) : (
                    <FileText className="w-4 h-4" />
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium">{src.fileName}</p>
                  <p className="text-[10px] text-muted-foreground uppercase tracking-wider font-semibold">
                    {(src.fileSize / 1024).toFixed(0)} KB • {formatDistanceToNow(new Date(src.ingestedAt))} ago
                  </p>
                </div>
              </div>
            ))}
            {notebook?.sources.length === 0 && !uploading && (
              <div className="text-center p-6 text-muted-foreground">
                <FileText className="w-8 h-8 mx-auto mb-2 opacity-20" />
                <p className="text-sm">No sources uploaded yet.</p>
              </div>
            )}
          </div>
        </ScrollArea>
        
        <div className="p-3 border-t">
          <Button variant="ghost" className="w-full justify-start text-muted-foreground hover:text-foreground rounded-lg" onClick={() => setSettingsOpen(true)}>
            <Settings className="w-4 h-4 mr-2" />
            LLM Settings
          </Button>
        </div>
      </aside>

      {/* Main Chat Area */}
      <main className="flex-1 flex flex-col relative bg-dot-pattern h-full min-w-0">
        {/* Dynamic ambient header glow */}
        <div className="absolute top-0 inset-x-0 h-32 bg-gradient-to-b from-primary/5 to-transparent pointer-events-none z-10" />

        <ScrollArea className="flex-1 min-h-0 h-full w-full px-4 lg:px-12 py-6">
          <div className="max-w-3xl mx-auto space-y-6 pb-24 pt-4">
            {messages.length === 0 ? (
              <div className="h-[60vh] flex flex-col items-center justify-center text-center">
                <div className="w-20 h-20 rounded-2xl bg-gradient-to-tr from-primary to-purple-600 shadow-xl shadow-primary/20 flex items-center justify-center mb-8 rotate-3">
                  <Sparkles className="w-10 h-10 text-white" />
                </div>
                <h1 className="text-3xl font-bold mb-3 tracking-tight">Ask your documents</h1>
                <p className="text-muted-foreground max-w-md text-lg">
                  Ground your questions in the uploaded sources. Responses are backed by embedded citations.
                </p>
              </div>
            ) : (
              messages.map((msg) => (
                <motion.div 
                  key={msg.id}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className={`flex gap-4 ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  {msg.role === 'assistant' && (
                    <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center shrink-0 shadow-sm">
                      <Sparkles className="w-4 h-4 text-primary-foreground" />
                    </div>
                  )}
                  
                  <div className={`max-w-[85%] ${msg.role === 'user' ? 'bg-secondary text-secondary-foreground px-5 py-3.5 rounded-2xl rounded-tr-sm' : 'pt-1'}`}>
                    {msg.role === 'user' ? (
                      <p className="whitespace-pre-wrap">{msg.content}</p>
                    ) : (
                      <div className="prose prose-sm dark:prose-invert max-w-none prose-p:leading-relaxed prose-pre:bg-black/5 dark:prose-pre:bg-white/5">
                        <ReactMarkdown
                          remarkPlugins={[remarkGfm]}
                          components={{
                            a: ({ node, href, children, ...props }) => {
                              const match = href?.match(/^#citation-(\d+)$/);
                              if (match && msg.citations) {
                                const citeIdx = parseInt(match[1]) - 1;
                                const cite = msg.citations.find(c => c.index === citeIdx + 1);
                                if (cite) {
                                  return (
                                    <Tooltip>
                                      <TooltipTrigger>
                                        <Badge variant="outline" className="mx-1 cursor-pointer hover:bg-primary/10 text-[10px] h-5 align-middle border-primary/30 text-primary shadow-sm hover:shadow transition-all">
                                          {match[1]}
                                        </Badge>
                                      </TooltipTrigger>
                                      <TooltipContent className="max-w-xs p-3 glass">
                                        <p className="font-semibold text-xs mb-1">{cite.fileName} (Page {cite.pageNumber})</p>
                                        <p className="text-xs text-muted-foreground line-clamp-4 italic">"{cite.snippet}"</p>
                                      </TooltipContent>
                                    </Tooltip>
                                  );
                                }
                              }
                              return (
                                <a
                                  {...props}
                                  href={href}
                                  className="text-primary hover:underline underline-offset-4"
                                  target="_blank"
                                  rel="noopener noreferrer"
                                >
                                  {children}
                                </a>
                              );
                            }
                          }}
                        >
                          {msg.content.replace(/\[Source (\d+)\]/g, '[$1](#citation-$1)')}
                        </ReactMarkdown>
                        
                        {msg.citations && msg.citations.length > 0 && (
                          <div className="mt-6 pt-4 border-t border-border/50">
                            <p className="text-xs font-semibold text-muted-foreground mb-3 tracking-wider uppercase">Sources Cited</p>
                            <div className="flex flex-wrap gap-2">
                              {msg.citations.map((c, i) => (
                                <div key={i} className="flex items-center gap-1.5 text-xs bg-black/5 dark:bg-white/5 py-1 px-2.5 rounded-md border border-border/50">
                                  <span className="text-primary font-bold">{c.index}</span>
                                  <FileText className="w-3 h-3 text-muted-foreground" />
                                  <span className="max-w-[150px] truncate">{c.fileName}</span>
                                  {c.pageNumber > 0 && <span className="text-muted-foreground opacity-70">p.{c.pageNumber}</span>}
                                </div>
                              ))}
                            </div>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                  
                  {msg.role === 'user' && (
                    <div className="w-8 h-8 rounded-full bg-muted flex items-center justify-center shrink-0">
                      <User className="w-4 h-4 text-muted-foreground" />
                    </div>
                  )}
                </motion.div>
              ))
            )}
            {chatting && (
              <div className="flex gap-4">
                <div className="w-8 h-8 rounded-full bg-primary/20 animate-pulse flex items-center justify-center shrink-0">
                  <div className="w-2 h-2 rounded-full bg-primary" />
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        </ScrollArea>

        {/* Input Area */}
        <div className="p-4 lg:p-6 bg-gradient-to-t from-background via-background to-transparent pt-10">
          <div className="max-w-3xl mx-auto relative group">
            <div className="absolute inset-0 bg-primary/5 rounded-2xl blur-xl transition-all group-hover:bg-primary/10" />
            <div className="relative flex items-center bg-card/80 backdrop-blur-xl border border-border/60 shadow-sm rounded-2xl p-2 pl-4 overflow-hidden focus-within:border-primary/50 focus-within:ring-1 focus-within:ring-primary/20 transition-all">
              <input
                className="flex-1 bg-transparent border-none outline-none resize-none px-2 py-3 text-base placeholder:text-muted-foreground/60"
                placeholder={notebook?.sources.length === 0 ? "Upload sources first to ask questions..." : "Ask a question about your documents..."}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    submitChat();
                  }
                }}
                disabled={chatting || (notebook?.sources.length === 0)}
              />
              <Button 
                size="icon" 
                className={`rounded-xl h-10 w-10 shrink-0 transition-all ${input.trim() ? 'bg-primary text-primary-foreground shadow-md' : 'bg-muted text-muted-foreground'}`}
                disabled={!input.trim() || chatting}
                onClick={submitChat}
              >
                <Send className="w-4 h-4 translate-y-[1px] -translate-x-[1px]" />
              </Button>
            </div>
          </div>
          <div className="max-w-3xl mx-auto text-center mt-3">
             <p className="text-[10px] text-muted-foreground font-medium uppercase tracking-widest">
               Powered by LocalAI Reranker + {provider}
             </p>
          </div>
        </div>
      </main>

      {/* Settings Dialog */}
      <Dialog open={settingsOpen} onOpenChange={(open) => {
        if (!open) {
          localStorage.setItem("gn_provider", provider);
          localStorage.setItem("gn_api_key", apiKey);
          localStorage.setItem("gn_model", model);
        }
        setSettingsOpen(open);
      }}>
        <DialogContent className="sm:max-w-[480px] glass max-h-[90vh] flex flex-col p-0 gap-0">
          <DialogHeader className="px-6 pt-6 pb-4 border-b shrink-0">
            <DialogTitle className="flex items-center gap-2">
              <Key className="w-5 h-5 text-primary" />
              LLM Configuration
            </DialogTitle>
            <DialogDescription className="text-xs">
              Keys stored in your browser only. Embedding &amp; reranking stay local.
            </DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-5 px-6 py-5 overflow-y-auto flex-1">
            {/* Provider pills */}
            <div className="grid gap-2">
              <Label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Provider</Label>
              <div className="flex flex-wrap gap-2">
                {[
                  { value: "openai", label: "OpenAI" },
                  { value: "google", label: "Gemini" },
                  { value: "anthropic", label: "Anthropic" },
                  { value: "groq", label: "Groq" },
                  { value: "openrouter", label: "OpenRouter" },
                ].map((p) => (
                  <button
                    key={p.value}
                    onClick={() => {
                      setProvider(p.value);
                      setDynamicModels([]);
                      setModel("");
                      if (apiKey) fetchDynamicModels(p.value, apiKey);
                    }}
                    className={`px-3 py-1.5 rounded-full text-sm font-medium transition-all border ${
                      provider === p.value
                        ? "bg-primary text-primary-foreground border-primary shadow-sm shadow-primary/30"
                        : "bg-background/50 text-muted-foreground border-border hover:border-primary/50 hover:text-foreground"
                    }`}
                  >
                    {p.label}
                  </button>
                ))}
              </div>
            </div>

            {/* API Key */}
            <div className="grid gap-2">
              <div className="flex items-center justify-between">
                <Label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">API Key</Label>
                {apiKey && (
                  <button
                    onClick={() => fetchDynamicModels(provider, apiKey)}
                    disabled={loadingModels}
                    className="text-xs text-primary hover:underline disabled:opacity-50 flex items-center gap-1"
                  >
                    {loadingModels ? (
                      <><Loader2 className="w-3 h-3 animate-spin" /> Loading models...</>
                    ) : (
                      "↻ Refresh models"
                    )}
                  </button>
                )}
              </div>
              <div className="relative">
                <Input
                  type={showApiKey ? "text" : "password"}
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  onBlur={() => { if (apiKey) fetchDynamicModels(provider, apiKey); }}
                  placeholder={provider === "openrouter" ? "sk-or-..." : provider === "groq" ? "gsk_..." : "sk-..."}
                  className="bg-background/50 pr-10 font-mono text-sm"
                />
                <Button
                  type="button" variant="ghost" size="icon"
                  className="absolute right-0 top-0 h-10 w-10 text-muted-foreground hover:text-foreground"
                  onClick={() => setShowApiKey(!showApiKey)}
                >
                  {showApiKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </Button>
              </div>
              {apiKey && (
                <p className="text-[10px] text-muted-foreground flex items-center gap-1">
                  <CheckCircle2 className="w-3 h-3 text-green-500" />
                  Key saved — ends in …{apiKey.slice(-4)}
                </p>
              )}
            </div>

            {/* Model Picker */}
            <div className="grid gap-2">
              <div className="flex items-center justify-between">
                <Label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Model</Label>
                <span className="text-[10px] text-muted-foreground">
                  {dynamicModels.length > 0 ? (
                    <span className="text-green-500 font-medium">● {dynamicModels.length} live models</span>
                  ) : (
                    <span className="text-yellow-500 font-medium">● static defaults</span>
                  )}
                </span>
              </div>

              {/* Search */}
              <div className="relative">
                <Input
                  type="text"
                  placeholder="Search models…"
                  value={modelSearch}
                  onChange={(e) => setModelSearch(e.target.value)}
                  className="bg-background/50 h-9 pl-8 text-sm"
                  disabled={loadingModels}
                />
                <svg className="absolute left-2.5 top-2.5 w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24"><circle cx="11" cy="11" r="8" /><path d="m21 21-4.35-4.35" /></svg>
              </div>

              {/* Model cards */}
              <div className="rounded-xl border border-border/60 overflow-hidden">
                {loadingModels ? (
                  <div className="flex items-center justify-center gap-2 py-8 text-sm text-muted-foreground">
                    <Loader2 className="w-4 h-4 animate-spin" /> Fetching latest models…
                  </div>
                ) : (
                  <div className="divide-y divide-border/40 max-h-[240px] overflow-y-auto">
                    {/* Backend default option */}
                    <button
                      onClick={() => setModel("")}
                      className={`w-full text-left px-4 py-3 transition-all flex items-center justify-between gap-3 group ${
                        model === ""
                          ? "bg-primary/10 text-primary"
                          : "hover:bg-muted/50 text-muted-foreground hover:text-foreground"
                      }`}
                    >
                      <div>
                        <p className="text-sm font-medium">Backend Default</p>
                        <p className="text-[10px]">Let the server decide</p>
                      </div>
                      {model === "" && <CheckCircle2 className="w-4 h-4 text-primary shrink-0" />}
                    </button>

                    {(dynamicModels.length > 0 ? dynamicModels : (AVAILABLE_MODELS[provider as keyof typeof AVAILABLE_MODELS] || []))
                      .filter((m) =>
                        modelSearch === "" ||
                        (m.name || m.id).toLowerCase().includes(modelSearch.toLowerCase()) ||
                        m.id.toLowerCase().includes(modelSearch.toLowerCase())
                      )
                      .map((m) => (
                        <button
                          key={m.id}
                          onClick={() => setModel(m.id)}
                          className={`w-full text-left px-4 py-3 transition-all flex items-center justify-between gap-3 ${
                            model === m.id
                              ? "bg-primary/10 text-primary"
                              : "hover:bg-muted/50 text-foreground/80 hover:text-foreground"
                          }`}
                        >
                          <div className="min-w-0">
                            <p className="text-sm font-medium truncate">{m.name || m.id}</p>
                            {m.name && m.name !== m.id && (
                              <p className="text-[10px] text-muted-foreground font-mono truncate">{m.id}</p>
                            )}
                          </div>
                          {model === m.id && <CheckCircle2 className="w-4 h-4 text-primary shrink-0" />}
                        </button>
                      ))
                    }
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Sticky footer with current selection summary */}
          <div className="px-6 py-4 border-t bg-muted/30 shrink-0">
            <div className="flex items-center gap-3 mb-3">
              <div className="flex-1 min-w-0">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold mb-0.5">Currently active</p>
                <p className="text-sm font-semibold truncate">
                  {provider.charAt(0).toUpperCase() + provider.slice(1)}
                  <span className="text-muted-foreground font-normal"> · </span>
                  <span className="text-primary">{model || "Backend Default"}</span>
                </p>
              </div>
            </div>
            <Button onClick={saveSettings} className="w-full rounded-full font-semibold">
              <CheckCircle2 className="w-4 h-4 mr-2" /> Save Configuration
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
