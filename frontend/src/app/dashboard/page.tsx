"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Plus, BookText, Calendar, FileText, Trash2, Loader2, Sparkles } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { formatDistanceToNow } from "date-fns";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { Notebook, fetchNotebooks, createNotebook, deleteNotebook } from "@/lib/api";

export default function Dashboard() {
  const [notebooks, setNotebooks] = useState<Notebook[]>([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [newDesc, setNewDesc] = useState("");
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    loadNotebooks(controller.signal);
    return () => controller.abort();
  }, []);

  const loadNotebooks = async (signal?: AbortSignal) => {
    try {
      const data = await fetchNotebooks(signal);
      setNotebooks(data);
    } catch (error: unknown) {
      if (error instanceof Error && error.name === "AbortError") return;
      toast.error("Failed to load notebooks");
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async () => {
    if (!newName.trim()) return;
    setCreating(true);
    try {
      const nb = await createNotebook(newName, newDesc);
      setNotebooks([...notebooks, nb]);
      setCreateOpen(false);
      setNewName("");
      setNewDesc("");
      toast.success("Notebook created successfully");
    } catch (error) {
      toast.error("Failed to create notebook");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (e: React.MouseEvent, id: string) => {
    e.preventDefault();
    if (!confirm("Are you sure you want to delete this notebook? All data will be lost.")) return;
    
    try {
      await deleteNotebook(id);
      setNotebooks(notebooks.filter((n) => n.id !== id));
      toast.success("Notebook deleted");
    } catch (error) {
      toast.error("Failed to delete notebook");
    }
  };

  return (
    <div className="min-h-screen bg-background relative overflow-hidden">
      {/* Decorative ambient background */}
      <div className="absolute top-0 -left-40 w-[600px] h-[600px] bg-primary/20 rounded-full blur-3xl opacity-30 pointer-events-none" />
      <div className="absolute bottom-0 -right-40 w-[600px] h-[600px] bg-purple-500/20 rounded-full blur-3xl opacity-30 pointer-events-none" />

      <main className="container mx-auto p-8 relative z-10 max-w-7xl">
        <div className="flex flex-col md:flex-row items-start md:items-center justify-between mb-12 gap-4">
          <div>
            <h1 className="text-4xl font-extrabold tracking-tight flex items-center gap-3">
              <span className="text-primary">
                <Sparkles className="w-8 h-8" />
              </span>
              GopherNotebook
            </h1>
            <p className="text-muted-foreground mt-2 text-lg">
              Source-grounded RAG workspaces powered by local AI.
            </p>
          </div>
          <Button size="lg" className="rounded-full shadow-lg hover:shadow-primary/20 transition-all font-semibold" onClick={() => setCreateOpen(true)}>
            <Plus className="w-5 h-5 mr-2" />
            New Notebook
          </Button>
        </div>

        {loading ? (
          <div className="flex justify-center items-center h-64">
            <Loader2 className="w-8 h-8 animate-spin text-primary" />
          </div>
        ) : notebooks.length === 0 ? (
          <motion.div 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="flex flex-col items-center justify-center p-16 text-center border border-dashed rounded-3xl bg-card/50 backdrop-blur-sm"
          >
            <div className="w-20 h-20 bg-primary/10 rounded-full flex items-center justify-center mb-6">
              <BookText className="w-10 h-10 text-primary" />
            </div>
            <h2 className="text-2xl font-bold mb-2">No notebooks yet</h2>
            <p className="text-muted-foreground mb-8 max-w-md text-lg">
              Create your first notebook to start uploading documents and chatting with your knowledge base.
            </p>
            <Button size="lg" className="rounded-full shadow-lg" onClick={() => setCreateOpen(true)}>
              <Plus className="w-5 h-5 mr-2" />
              Create Notebook
            </Button>
          </motion.div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            <AnimatePresence>
              {notebooks.map((nb, i) => (
                <motion.div
                  key={nb.id}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, scale: 0.95 }}
                  transition={{ delay: i * 0.05 }}
                  layout
                >
                  <Link href={`/notebook/${nb.id}`} className="block h-full group">
                    <Card className="h-full flex flex-col hover:border-primary/50 hover:shadow-xl hover:shadow-primary/10 transition-all duration-300 bg-card/60 backdrop-blur-md overflow-hidden relative">
                      <div className="absolute inset-0 bg-gradient-to-br from-primary/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
                      <CardHeader>
                        <CardTitle className="flex justify-between items-start">
                          <span className="text-xl inline-block truncate">{nb.name}</span>
                          <Button 
                            variant="ghost" 
                            size="icon" 
                            className="h-8 w-8 text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 transition-opacity z-20"
                            onClick={(e) => handleDelete(e, nb.id)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </CardTitle>
                        <CardDescription className="line-clamp-2 min-h-10 text-base">
                          {nb.description || "No description provided."}
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="flex-1" />
                      <CardFooter className="flex justify-between text-xs text-muted-foreground border-t pt-4">
                        <div className="flex items-center gap-1.5 font-medium">
                          <FileText className="w-4 h-4 text-primary" />
                          {nb.fileCount} sources
                        </div>
                        <div className="flex items-center gap-1.5">
                          <Calendar className="w-4 h-4" />
                          {formatDistanceToNow(new Date(nb.createdAt), { addSuffix: true })}
                        </div>
                      </CardFooter>
                    </Card>
                  </Link>
                </motion.div>
              ))}
            </AnimatePresence>
          </div>
        )}

        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogContent className="sm:max-w-[425px] glass border-primary/20">
            <DialogHeader>
              <DialogTitle className="text-2xl">Create Notebook</DialogTitle>
              <DialogDescription className="text-base pt-2">
                A notebook acts as an isolated workspace for a specific project or topic.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-6 py-4">
              <div className="grid gap-2">
                <Label htmlFor="name" className="text-base">
                  Name
                </Label>
                <Input
                  id="name"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="e.g. Q3 Financial Reports"
                  autoFocus
                  className="text-base py-6 bg-background/50"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleCreate();
                  }}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="description" className="text-base">
                  Description <span className="text-muted-foreground font-normal">(Optional)</span>
                </Label>
                <Input
                  id="description"
                  value={newDesc}
                  onChange={(e) => setNewDesc(e.target.value)}
                  placeholder="Analysis of revenue projections"
                  className="text-base py-6 bg-background/50"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleCreate();
                  }}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setCreateOpen(false)} className="rounded-full px-6">
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={!newName.trim() || creating} className="rounded-full px-6">
                {creating ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : null}
                Create Space
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </main>
    </div>
  );
}
