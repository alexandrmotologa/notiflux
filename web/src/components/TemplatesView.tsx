import React, { useState, useEffect } from "react";
import {
  FileCode,
  Save,
  Eye,
  Code,
  Sparkles,
  Smartphone,
  Mail,
  Send,
  MessageSquare,
  Globe,
  Bell,
} from "lucide-react";
import type { Workflow, Template } from "../types";
import { fetchTemplates, updateTemplates, previewTemplate } from "../api";

interface TemplatesViewProps {
  workflows: Workflow[];
  initialWorkflowId?: string;
}

const CHANNELS = [
  { id: "in_app", label: "In-App Bell", icon: Bell },
  { id: "email", label: "Email", icon: Mail },
  { id: "web_push", label: "Web Push", icon: Smartphone },
  { id: "telegram", label: "Telegram", icon: Send },
  { id: "slack", label: "Slack", icon: MessageSquare },
  { id: "webhook", label: "Webhook", icon: Globe },
];

export const TemplatesView: React.FC<TemplatesViewProps> = ({
  workflows,
  initialWorkflowId,
}) => {
  const [selectedWf, setSelectedWf] = useState(initialWorkflowId || "");
  const [templates, setTemplates] = useState<Record<string, { subject: string; body: string }>>({});
  const [activeChannel, setActiveChannel] = useState("in_app");
  const [activeMode, setActiveMode] = useState<"edit" | "preview">("edit");
  const [previewOutput, setPreviewOutput] = useState<{ subject: string; body: string; html?: string } | null>(null);
  const [sampleData, setSampleData] = useState(`{\n  "order_id": "#8942",\n  "total": "$149.00",\n  "customer_name": "Alexandr",\n  "action_url": "https://example.com/orders/8942"\n}`);
  const [saveStatus, setSaveStatus] = useState<string | null>(null);

  useEffect(() => {
    if (!selectedWf && workflows.length > 0) {
      setSelectedWf(workflows[0].id);
    }
  }, [workflows, selectedWf]);

  useEffect(() => {
    if (selectedWf) {
      loadTemplates(selectedWf);
    }
  }, [selectedWf]);

  const loadTemplates = async (wfId: string) => {
    try {
      const list: Template[] = await fetchTemplates(wfId);
      const map: Record<string, { subject: string; body: string }> = {};
      list.forEach((t) => {
        map[t.channel] = { subject: t.subject || "", body: t.body || "" };
      });

      // Populate defaults if missing
      CHANNELS.forEach((ch) => {
        if (!map[ch.id]) {
          map[ch.id] = {
            subject: ch.id === "email" ? "Notification: {{order_id}}" : "Update",
            body: ch.id === "email"
              ? "## Hello {{customer_name}}\n\nYour order **{{order_id}}** for {{total}} is confirmed.\n\n[View Details]({{action_url}})"
              : "Hello {{customer_name}}, your order {{order_id}} for {{total}} is ready.",
          };
        }
      });
      setTemplates(map);
    } catch {
      // Handle error
    }
  };

  const handleSave = async () => {
    if (!selectedWf) return;
    setSaveStatus("Saving...");
    try {
      const list = Object.entries(templates).map(([channel, data]) => ({
        channel,
        subject: data.subject,
        body: data.body,
      }));
      await updateTemplates(selectedWf, list);
      setSaveStatus("Saved successfully!");
      setTimeout(() => setSaveStatus(null), 2500);
    } catch (err: any) {
      setSaveStatus(`Error: ${err.message}`);
    }
  };

  const handlePreview = async () => {
    let parsedData = {};
    try {
      parsedData = JSON.parse(sampleData);
    } catch {}

    const currentTpl = templates[activeChannel] || { subject: "", body: "" };
    try {
      const res = await previewTemplate(activeChannel, currentTpl.subject, currentTpl.body, parsedData);
      setPreviewOutput(res);
      setActiveMode("preview");
    } catch {
      // Handle error
    }
  };

  const insertVariable = (varName: string) => {
    const current = templates[activeChannel] || { subject: "", body: "" };
    setTemplates({
      ...templates,
      [activeChannel]: {
        ...current,
        body: current.body + ` {{${varName}}}`,
      },
    });
  };

  const currentTemplate = templates[activeChannel] || { subject: "", body: "" };

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-white tracking-tight">Template Designer</h1>
          <p className="text-sm text-gray-400">Design dynamic multi-channel message templates with live variable simulation</p>
        </div>

        <div className="flex items-center gap-3">
          <select
            value={selectedWf}
            onChange={(e) => setSelectedWf(e.target.value)}
            className="bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500"
          >
            {workflows.map((wf) => (
              <option key={wf.id} value={wf.id}>
                {wf.name} ({wf.id})
              </option>
            ))}
          </select>

          <button
            onClick={handleSave}
            className="bg-indigo-600 hover:bg-indigo-500 text-white font-medium py-2 px-4 rounded-lg text-sm flex items-center gap-2 transition-colors"
          >
            <Save className="w-4 h-4" />
            Save Changes
          </button>
        </div>
      </div>

      {saveStatus && (
        <div
          className={`p-3 rounded-lg text-xs font-mono ${
            saveStatus.startsWith("Error")
              ? "bg-rose-500/10 text-rose-400 border border-rose-500/20"
              : "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
          }`}
        >
          {saveStatus}
        </div>
      )}

      {/* Channel Tabs */}
      <div className="flex items-center gap-2 border-b border-[#243247] pb-px overflow-x-auto">
        {CHANNELS.map((ch) => {
          const Icon = ch.icon;
          const isActive = activeChannel === ch.id;
          return (
            <button
              key={ch.id}
              onClick={() => {
                setActiveChannel(ch.id);
                setActiveMode("edit");
              }}
              className={`flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors whitespace-nowrap ${
                isActive
                  ? "border-indigo-500 text-indigo-400 bg-indigo-500/5"
                  : "border-transparent text-gray-400 hover:text-gray-200 hover:bg-[#182234]"
              }`}
            >
              <Icon className="w-4 h-4" />
              {ch.label}
            </button>
          );
        })}
      </div>

      {/* Editor & Preview Container */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column: Template Editor */}
        <div className="lg:col-span-8 bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm space-y-4">
          <div className="flex items-center justify-between pb-3 border-b border-[#243247]">
            <div className="flex items-center gap-2">
              <FileCode className="w-4 h-4 text-indigo-400" />
              <span className="text-sm font-semibold text-white">
                {CHANNELS.find((c) => c.id === activeChannel)?.label} Template
              </span>
            </div>

            <div className="flex items-center gap-1 bg-[#182234] border border-[#243247] p-1 rounded-lg">
              <button
                onClick={() => setActiveMode("edit")}
                className={`px-3 py-1 rounded text-xs font-medium flex items-center gap-1.5 transition-colors ${
                  activeMode === "edit" ? "bg-indigo-600 text-white" : "text-gray-400 hover:text-gray-200"
                }`}
              >
                <Code className="w-3.5 h-3.5" /> Edit
              </button>
              <button
                onClick={handlePreview}
                className={`px-3 py-1 rounded text-xs font-medium flex items-center gap-1.5 transition-colors ${
                  activeMode === "preview" ? "bg-indigo-600 text-white" : "text-gray-400 hover:text-gray-200"
                }`}
              >
                <Eye className="w-3.5 h-3.5" /> Preview
              </button>
            </div>
          </div>

          {activeMode === "edit" ? (
            <div className="space-y-4">
              {/* Subject */}
              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">
                  Subject / Title
                </label>
                <input
                  type="text"
                  value={currentTemplate.subject}
                  onChange={(e) =>
                    setTemplates({
                      ...templates,
                      [activeChannel]: { ...currentTemplate, subject: e.target.value },
                    })
                  }
                  className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 font-mono focus:outline-none focus:border-indigo-500"
                />
              </div>

              {/* Body */}
              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="text-xs font-medium text-gray-400">
                    Message Body (Mustache &amp; Markdown supported)
                  </label>
                  <span className="text-[11px] text-gray-500">
                    {activeChannel === "email" ? "Renders to styled HTML email" : "Text payload"}
                  </span>
                </div>
                <textarea
                  value={currentTemplate.body}
                  onChange={(e) =>
                    setTemplates({
                      ...templates,
                      [activeChannel]: { ...currentTemplate, body: e.target.value },
                    })
                  }
                  rows={10}
                  className="w-full bg-[#182234] border border-[#243247] rounded-lg p-3 text-sm text-gray-200 font-mono focus:outline-none focus:border-indigo-500 resize-none leading-relaxed"
                />
              </div>

              {/* Variable Insertion Pills */}
              <div>
                <span className="text-xs font-medium text-gray-400 block mb-2 flex items-center gap-1.5">
                  <Sparkles className="w-3.5 h-3.5 text-amber-400" /> Quick Variables
                </span>
                <div className="flex flex-wrap gap-2">
                  {["order_id", "total", "customer_name", "action_url", "subscriber.email"].map(
                    (v) => (
                      <button
                        key={v}
                        type="button"
                        onClick={() => insertVariable(v)}
                        className="px-2 py-1 rounded bg-[#182234] border border-[#243247] hover:border-indigo-500/50 text-[11px] font-mono text-indigo-300 transition-colors"
                      >
                        + {`{{${v}}}`}
                      </button>
                    )
                  )}
                </div>
              </div>
            </div>
          ) : (
            /* Preview Rendering */
            <div className="space-y-4">
              <div className="bg-[#182234] border border-[#243247] rounded-lg p-3 text-xs">
                <span className="text-gray-400">Rendered Subject: </span>
                <span className="font-semibold text-white">{previewOutput?.subject || "(No subject)"}</span>
              </div>

              {activeChannel === "email" && previewOutput?.html ? (
                <div className="border border-[#243247] rounded-lg overflow-hidden bg-white">
                  <iframe
                    title="Email Preview"
                    srcDoc={previewOutput.html}
                    className="w-full h-80 border-0"
                  />
                </div>
              ) : (
                <div className="bg-[#182234] border border-[#243247] rounded-lg p-4 font-mono text-sm text-gray-200 whitespace-pre-wrap">
                  {previewOutput?.body}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Right Column: Sample JSON Variables */}
        <div className="lg:col-span-4 bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm space-y-3">
          <div className="flex items-center justify-between pb-3 border-b border-[#243247]">
            <span className="text-sm font-semibold text-white">Sample Preview Variables</span>
            <span className="text-xs text-gray-500">JSON</span>
          </div>

          <p className="text-xs text-gray-400">
            Edit test variables to simulate how Mustache variables resolve in live dispatches.
          </p>

          <textarea
            value={sampleData}
            onChange={(e) => setSampleData(e.target.value)}
            rows={10}
            className="w-full bg-[#182234] border border-[#243247] rounded-lg p-3 text-xs text-gray-200 font-mono focus:outline-none focus:border-indigo-500 resize-none leading-relaxed"
          />

          <button
            onClick={handlePreview}
            className="w-full bg-[#182234] hover:bg-indigo-600/20 text-indigo-400 border border-indigo-500/30 font-medium py-2 px-3 rounded-lg text-xs flex items-center justify-center gap-2 transition-colors"
          >
            <Eye className="w-3.5 h-3.5" /> Re-render Preview
          </button>
        </div>
      </div>
    </div>
  );
};
