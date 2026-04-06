"use client";

import Link from "next/link";
import { motion, Variants } from "framer-motion";
import { ArrowRight, Box, Cpu, FileText, Lock, Sparkles, Zap, Server } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function LandingPage() {
  const containerVariants = {
    hidden: { opacity: 0 },
    visible: { opacity: 1, transition: { staggerChildren: 0.2, delayChildren: 0.1 } },
  };

  const itemVariants: Variants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { type: "spring", stiffness: 100 } },
  };

  return (
    <div className="min-h-screen bg-background text-foreground overflow-hidden selection:bg-primary/30">
      {/* Decorative Orbs */}
      <div className="absolute top-[-10%] left-[-10%] w-[50vw] h-[50vw] bg-primary/20 rounded-full blur-[120px] opacity-40 pointer-events-none mix-blend-screen" />
      <div className="absolute bottom-[-10%] right-[-10%] w-[50vw] h-[50vw] bg-purple-500/20 rounded-full blur-[120px] opacity-40 pointer-events-none mix-blend-screen" />
      <div className="absolute top-[40%] left-[50%] w-[30vw] h-[30vw] -translate-x-1/2 -translate-y-1/2 bg-blue-500/10 rounded-full blur-[100px] pointer-events-none" />

      {/* Navigation */}
      <nav className="relative z-50 flex items-center justify-between px-6 py-6 max-w-7xl mx-auto border-b border-border/10">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-primary to-purple-600 flex items-center justify-center shadow-lg shadow-primary/20">
            <Sparkles className="w-4 h-4 text-white" />
          </div>
          <span className="text-xl font-bold tracking-tight">GopherNotebook</span>
        </div>
        <div className="flex items-center gap-4">
          <Link href="/dashboard">
            <Button variant="ghost" className="hidden sm:flex text-muted-foreground hover:text-primary">
              Enter Workspace
            </Button>
          </Link>
          <Link href="/dashboard">
            <Button className="rounded-full px-6 shadow-xl shadow-primary/20 group">
              Start Free
              <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
            </Button>
          </Link>
        </div>
      </nav>

      <main className="relative z-10">
        {/* Hero Section */}
        <section className="pt-24 pb-32 px-6">
          <motion.div 
            className="max-w-4xl mx-auto text-center"
            variants={containerVariants}
            initial="hidden"
            animate="visible"
          >
            <motion.div variants={itemVariants} className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-primary/10 border border-primary/20 text-primary text-sm font-medium mb-8">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-primary"></span>
              </span>
              LocalAI RAG Engine Active
            </motion.div>
            
            <motion.h1 variants={itemVariants} className="text-5xl md:text-7xl font-extrabold tracking-tight mb-8 leading-[1.1]">
              Your intelligence, <br />
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-primary via-purple-400 to-primary/40 animate-gradient-x">
                securely grounded.
              </span>
            </motion.h1>
            
            <motion.p variants={itemVariants} className="text-xl text-muted-foreground max-w-2xl mx-auto mb-12 leading-relaxed">
              Upload hundreds of documents. Keep your embeddings 100% local. Generate responses with zero hallucinations using an enterprise-grade Golang RAG pipeline.
            </motion.p>
            
            <motion.div variants={itemVariants} className="flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link href="/dashboard">
                <Button size="lg" className="rounded-full px-8 h-14 text-lg w-full sm:w-auto shadow-xl shadow-primary/25 hover:shadow-primary/40 transition-all font-semibold">
                  Launch Dashboard
                </Button>
              </Link>
              <Link href="https://github.com/RobinMillford/GopherNotebook" target="_blank">
                <Button size="lg" variant="outline" className="rounded-full px-8 h-14 text-lg w-full sm:w-auto glass hover:bg-white/5 transition-all">
                  View Architecture
                </Button>
              </Link>
            </motion.div>
          </motion.div>
          
          {/* Abstract Dashboard Preview window */}
          <motion.div 
            initial={{ opacity: 0, y: 40 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.6, duration: 1, type: "spring", bounce: 0.4 }}
            className="mt-20 max-w-5xl mx-auto hidden md:block"
          >
            <div className="rounded-2xl border border-white/10 bg-black/40 backdrop-blur-3xl shadow-2xl overflow-hidden glass relative group">
              <div className="absolute inset-0 bg-gradient-to-t from-background via-transparent to-transparent z-20 pointer-events-none" />
              
              {/* Fake Window Header */}
              <div className="h-12 border-b border-white/5 flex items-center px-4 gap-2 bg-white/5">
                <div className="w-3 h-3 rounded-full bg-red-500/50" />
                <div className="w-3 h-3 rounded-full bg-yellow-500/50" />
                <div className="w-3 h-3 rounded-full bg-green-500/50" />
              </div>
              
              {/* Fake Window Content */}
              <div className="p-8 grid grid-cols-3 gap-6 opacity-60 group-hover:opacity-100 transition-opacity duration-1000">
                <div className="col-span-1 border-r border-white/5 pr-6 space-y-4">
                  <div className="h-10 rounded-lg bg-white/5 w-full mb-8"></div>
                  <div className="flex gap-3 items-center"><FileText className="w-4 h-4 text-primary"/><div className="h-4 rounded bg-white/10 flex-1"></div></div>
                  <div className="flex gap-3 items-center"><FileText className="w-4 h-4 text-purple-400"/><div className="h-4 rounded bg-white/5 w-2/3"></div></div>
                  <div className="flex gap-3 items-center"><FileText className="w-4 h-4 text-blue-400"/><div className="h-4 rounded bg-white/5 w-4/5"></div></div>
                </div>
                <div className="col-span-2 space-y-6">
                  <div className="w-3/4 p-4 rounded-xl rounded-tl-sm bg-white/5 ml-4">
                     <div className="h-3 rounded bg-white/20 w-1/2 mb-2"></div>
                     <div className="h-3 rounded bg-white/10 w-full mb-2"></div>
                     <div className="h-3 rounded bg-white/10 w-2/3"></div>
                  </div>
                  <div className="w-5/6 p-4 rounded-xl rounded-tr-sm bg-primary/20 ml-auto border border-primary/20 text-right">
                     <div className="h-3 rounded bg-primary/50 w-full mb-2 ml-auto"></div>
                     <div className="h-3 rounded bg-primary/30 w-3/4 ml-auto"></div>
                  </div>
                </div>
              </div>
            </div>
          </motion.div>
        </section>

        {/* Features Section */}
        <section className="py-24 px-6 relative border-t border-white/5 bg-black/20">
          <div className="max-w-6xl mx-auto">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-5xl font-bold tracking-tight mb-4">Uncompromising Architecture.</h2>
              <p className="text-muted-foreground text-lg max-w-2xl mx-auto">Engineered for maximum speed, privacy, and contextual accuracy.</p>
            </div>
            
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
              {[
                {
                  icon: Lock,
                  title: "100% Local RAG",
                  desc: "Your files never leave your machine during ingestion. Embeddings and reranking run entirely via local Weaviate and LocalAI models."
                },
                {
                  icon: Zap,
                  title: "Blazing Fast Golang",
                  desc: "Built on Go, the backend handles large document chunking and concurrent hybrid searches at sub-millisecond latencies."
                },
                {
                  icon: Cpu,
                  title: "Hybrid Search & Reranking",
                  desc: "Combines dense vector matching with BM25 keyword search, then refines results using a powerful cross-encoder reranker."
                },
                {
                  icon: Server,
                  title: "Bring Your Own LLM",
                  desc: "Plug in your API keys for OpenAI, Anthropic, or Google Gemini. Swap seamlessly with full streaming support."
                },
                {
                  icon: FileText,
                  title: "Granular Citations",
                  desc: "Never question the AI. Every claim includes embedded citations linking directly to the source document and page number."
                },
                {
                  icon: Box,
                  title: "Isolated Workspaces",
                  desc: "Group related documents into 'Notebooks'. Create distinct mental contexts for distinct projects without cross-contamination."
                }
              ].map((feature, i) => (
                <div key={i} className="p-8 rounded-3xl bg-card border border-border/50 hover:border-primary/50 transition-colors group relative overflow-hidden">
                   <div className="absolute inset-0 bg-gradient-to-br from-primary/5 to-transparent opacity-0 group-hover:opacity-100 transition-duration-500" />
                   <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mb-6 text-primary group-hover:scale-110 group-hover:bg-primary group-hover:text-white transition-all duration-300">
                     <feature.icon className="w-6 h-6" />
                   </div>
                   <h3 className="text-xl font-bold mb-3">{feature.title}</h3>
                   <p className="text-muted-foreground leading-relaxed">
                     {feature.desc}
                   </p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* CTA Section */}
        <section className="py-32 px-6 relative overflow-hidden">
           <div className="absolute inset-0 bg-primary/5" />
           <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-full max-w-lg h-full max-h-lg bg-primary/20 blur-[100px] rounded-full pointer-events-none" />
           
           <div className="max-w-4xl mx-auto text-center relative z-10 glass p-12 rounded-3xl border border-white/10">
             <h2 className="text-4xl md:text-5xl font-bold tracking-tight mb-6">Ready to expand your local brain?</h2>
             <p className="text-xl text-muted-foreground mb-10">Stop searching, start knowing. Access your first notebook free.</p>
             <Link href="/dashboard">
                <Button size="lg" className="rounded-full px-10 h-16 text-xl shadow-2xl shadow-primary/30 hover:scale-105 transition-transform duration-300 font-bold">
                  Get Started Now
                </Button>
             </Link>
           </div>
        </section>
      </main>
      
      <footer className="py-8 text-center text-muted-foreground text-sm border-t border-border/10 relative z-10">
        <p>© 2026 GopherNotebook. Open-source local AI infrastructure.</p>
      </footer>
    </div>
  );
}
