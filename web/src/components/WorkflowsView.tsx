import React, { useState } from "react";
import { GitBranch, Plus, Trash2, Edit3, Check, Code, Copy, X } from "lucide-react";
import type { Workflow } from "../types";
import { createWorkflow, deleteWorkflow } from "../api";

interface WorkflowsViewProps {
  workflows: Workflow[];
  onRefresh: () => void;
  onEditTemplates: (workflowId: string) => void;
}

const AVAILABLE_CHANNELS = [
  { id: "in_app", label: "In-App Bell" },
  { id: "email", label: "Email (SMTP / Resend)" },
  { id: "web_push", label: "Web Push (RFC 8292)" },
  { id: "telegram", label: "Telegram Bot" },
  { id: "slack", label: "Slack Webhook" },
  { id: "discord", label: "Discord Webhook" },
  { id: "webhook", label: "Signed HTTPS Webhook" },
];

export const WorkflowsView: React.FC<WorkflowsViewProps> = ({
  workflows,
  onRefresh,
  onEditTemplates,
}) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [id, setId] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedChannels, setSelectedChannels] = useState<string[]>(["in_app", "email"]);
  const [error, setError] = useState<string | null>(null);

  // Code snippets modal state
  const [snippetWf, setSnippetWf] = useState<Workflow | null>(null);
  const [snippetLang, setSnippetLang] = useState<"curl" | "node" | "python" | "go" | "php">("curl");
  const [copiedSnippet, setCopiedSnippet] = useState(false);

  const toggleChannel = (ch: string) => {
    setSelectedChannels((prev) =>
      prev.includes(ch) ? prev.filter((c) => c !== ch) : [...prev, ch]
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      await createWorkflow({
        id,
        name,
        description,
        channels: selectedChannels,
      });
      setIsModalOpen(false);
      setId("");
      setName("");
      setDescription("");
      onRefresh();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleDelete = async (wfId: string) => {
    if (confirm(`Are you sure you want to delete workflow "${wfId}"?`)) {
      try {
        await deleteWorkflow(wfId);
        onRefresh();
      } catch (err: any) {
        alert(err.message);
      }
    }
  };

  const getSnippetCode = (wfId: string, lang: string) => {
    switch (lang) {
      case "curl":
        return `curl -X POST http://localhost:8080/v1/trigger \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{
    "workflow": "${wfId}",
    "subscriber": {
      "id": "usr_123",
      "email": "user@example.com"
    },
    "payload": {
      "title": "Order Confirmed",
      "body": "Your order #1042 has shipped.",
      "action_url": "https://example.com/orders/1042"
    }
  }'`;
      case "node":
        return `const response = await fetch("http://localhost:8080/v1/trigger", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer YOUR_API_KEY",
  },
  body: JSON.stringify({
    workflow: "${wfId}",
    subscriber: {
      id: "usr_123",
      email: "user@example.com",
    },
    payload: {
      title: "Order Confirmed",
      body: "Your order #1042 has shipped.",
      action_url: "https://example.com/orders/1042",
    },
  }),
});

const data = await response.json();
console.log("Notiflux dispatch:", data);`;
      case "python":
        return `import requests

resp = requests.post(
    "http://localhost:8080/v1/trigger",
    headers={"Authorization": "Bearer YOUR_API_KEY"},
    json={
        "workflow": "${wfId}",
        "subscriber": {
            "id": "usr_123",
            "email": "user@example.com",
        },
        "payload": {
            "title": "Order Confirmed",
            "body": "Your order #1042 has shipped.",
            "action_url": "https://example.com/orders/1042",
        },
    },
)

print(resp.json())`;
      case "go":
        return `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	payload := map[string]any{
		"workflow": "${wfId}",
		"subscriber": map[string]string{
			"id":    "usr_123",
			"email": "user@example.com",
		},
		"payload": map[string]string{
			"title":      "Order Confirmed",
			"body":       "Your order #1042 has shipped.",
			"action_url": "https://example.com/orders/1042",
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "http://localhost:8080/v1/trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer YOUR_API_KEY")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println("Status:", resp.Status)
}`;
      case "php":
        return `<?php
$ch = curl_init("http://localhost:8080/v1/trigger");
curl_setopt($ch, CURLOPT_POST, 1);
curl_setopt($ch, CURLOPT_HTTPHEADER, [
    "Content-Type: application/json",
    "Authorization: Bearer YOUR_API_KEY"
]);
curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode([
    "workflow" => "${wfId}",
    "subscriber" => [
        "id" => "usr_123",
        "email" => "user@example.com"
    ],
    "payload" => [
        "title" => "Order Confirmed",
        "body" => "Your order #1042 has shipped.",
        "action_url" => "https://example.com/orders/1042"
    ]
]));
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
$response = curl_exec($ch);
curl_close($ch);
echo $response;
?>`;
      default:
        return "";
    }
  };

  const copyCode = (code: string) => {
    navigator.clipboard.writeText(code);
    setCopiedSnippet(true);
    setTimeout(() => setCopiedSnippet(false), 2000);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-white tracking-tight">Workflows</h1>
          <p className="text-sm text-gray-400">Define multi-channel notification pipelines and routing targets</p>
        </div>
        <button
          onClick={() => setIsModalOpen(true)}
          className="bg-indigo-600 hover:bg-indigo-500 text-white font-medium py-2 px-4 rounded-lg text-sm flex items-center gap-2 transition-colors"
        >
          <Plus className="w-4 h-4" />
          Create Workflow
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {workflows.map((wf) => (
          <div
            key={wf.id}
            className="bg-[#111827] border border-[#243247] hover:border-indigo-500/40 rounded-xl p-5 shadow-sm transition-all flex flex-col justify-between"
          >
            <div>
              <div className="flex items-start justify-between mb-2">
                <div className="flex items-center gap-2">
                  <div className="w-7 h-7 rounded-lg bg-indigo-500/10 border border-indigo-500/30 flex items-center justify-center text-indigo-400">
                    <GitBranch className="w-4 h-4" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-white text-sm">{wf.name}</h3>
                    <span className="text-xs font-mono text-gray-400">{wf.id}</span>
                  </div>
                </div>
                <button
                  onClick={() => handleDelete(wf.id)}
                  className="text-gray-400 hover:text-rose-400 p-1 rounded"
                  title="Delete workflow"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>

              {wf.description && (
                <p className="text-xs text-gray-400 mb-4 line-clamp-2">{wf.description}</p>
              )}

              <div className="space-y-1 mb-4">
                <span className="text-[11px] font-medium uppercase tracking-wider text-gray-400">
                  Active Channels
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {wf.channels.map((ch) => (
                    <span
                      key={ch}
                      className="px-2 py-0.5 rounded text-[11px] font-medium bg-[#182234] text-indigo-300 border border-[#243247]"
                    >
                      {ch}
                    </span>
                  ))}
                </div>
              </div>
            </div>

            <div className="pt-3 border-t border-[#243247] flex items-center justify-between">
              <button
                onClick={() => setSnippetWf(wf)}
                className="text-xs text-gray-400 hover:text-indigo-400 font-medium flex items-center gap-1.5 transition-colors"
                title="View Integration Code Snippets"
              >
                <Code className="w-3.5 h-3.5" />
                Code Snippets
              </button>
              <button
                onClick={() => onEditTemplates(wf.id)}
                className="text-xs text-indigo-400 hover:text-indigo-300 font-medium flex items-center gap-1.5"
              >
                <Edit3 className="w-3.5 h-3.5" />
                Edit Templates
              </button>
            </div>
          </div>
        ))}
      </div>

      {/* Code Snippets Modal */}
      {snippetWf && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-[#111827] border border-[#243247] rounded-xl max-w-2xl w-full p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-[#243247]">
              <div className="flex items-center gap-2">
                <Code className="w-5 h-5 text-indigo-400" />
                <h2 className="text-base font-bold text-white">
                  Integration Snippet: <span className="font-mono text-indigo-300">{snippetWf.id}</span>
                </h2>
              </div>
              <button
                onClick={() => setSnippetWf(null)}
                className="text-gray-400 hover:text-gray-200"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Language Tabs */}
            <div className="flex items-center gap-2 border-b border-[#243247] pb-2">
              {[
                { id: "curl", label: "cURL" },
                { id: "node", label: "Node.js" },
                { id: "python", label: "Python" },
                { id: "go", label: "Go" },
                { id: "php", label: "PHP" },
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setSnippetLang(tab.id as any)}
                  className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
                    snippetLang === tab.id
                      ? "bg-indigo-600 text-white"
                      : "text-gray-400 hover:text-gray-200 bg-[#182234]"
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>

            {/* Code Block */}
            <div className="relative">
              <pre className="bg-[#0b0f17] border border-[#243247] rounded-lg p-4 font-mono text-xs text-gray-300 overflow-x-auto max-h-80 select-all leading-relaxed">
                {getSnippetCode(snippetWf.id, snippetLang)}
              </pre>
              <button
                onClick={() => copyCode(getSnippetCode(snippetWf.id, snippetLang))}
                className="absolute top-3 right-3 bg-[#182234] hover:bg-[#243247] border border-[#243247] text-gray-300 px-2.5 py-1 rounded text-xs flex items-center gap-1.5 transition-colors"
              >
                {copiedSnippet ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                {copiedSnippet ? "Copied" : "Copy"}
              </button>
            </div>

            <div className="text-xs text-gray-400 flex items-center justify-between pt-2">
              <span>POST /v1/trigger payload contract</span>
              <button
                onClick={() => setSnippetWf(null)}
                className="px-4 py-1.5 bg-[#182234] hover:bg-[#243247] text-gray-300 rounded-lg text-xs"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-[#111827] border border-[#243247] rounded-xl max-w-md w-full p-6 shadow-2xl">
            <h2 className="text-lg font-bold text-white mb-4">Create Notification Workflow</h2>

            {error && (
              <div className="p-3 mb-4 rounded-lg bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs">
                {error}
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">
                  Workflow Identifier (Slug)
                </label>
                <input
                  type="text"
                  placeholder="e.g. order.shipped"
                  value={id}
                  onChange={(e) => setId(e.target.value)}
                  className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">
                  Display Name
                </label>
                <input
                  type="text"
                  placeholder="e.g. Order Shipped Notification"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">
                  Description
                </label>
                <textarea
                  placeholder="Dispatched when carrier takes custody of package"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={2}
                  className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500 resize-none"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-400 mb-2">
                  Channels
                </label>
                <div className="space-y-2">
                  {AVAILABLE_CHANNELS.map((ch) => {
                    const isSelected = selectedChannels.includes(ch.id);
                    return (
                      <div
                        key={ch.id}
                        onClick={() => toggleChannel(ch.id)}
                        className={`flex items-center justify-between p-2.5 rounded-lg border cursor-pointer transition-colors text-xs ${
                          isSelected
                            ? "bg-indigo-600/10 border-indigo-500/40 text-gray-200"
                            : "bg-[#182234] border-[#243247] text-gray-400 hover:text-gray-200"
                        }`}
                      >
                        <span className="font-medium">{ch.label}</span>
                        <div
                          className={`w-4 h-4 rounded flex items-center justify-center border ${
                            isSelected
                              ? "bg-indigo-600 border-indigo-500 text-white"
                              : "border-[#374151]"
                          }`}
                        >
                          {isSelected && <Check className="w-3 h-3" />}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-[#243247]">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 text-sm text-gray-400 hover:text-gray-200 font-medium"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="bg-indigo-600 hover:bg-indigo-500 text-white font-medium py-2 px-4 rounded-lg text-sm transition-colors"
                >
                  Save Workflow
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
