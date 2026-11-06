import React, { useState, useEffect } from "react";
import {
  Send,
  Radio,
  CheckCircle2,
  XCircle,
  Clock,
  Cpu,
  ArrowRight,
  Eye,
  MousePointerClick,
} from "lucide-react";
import type { SystemHealth, DeliveryLog, Workflow } from "../types";
import { triggerNotification } from "../api";

interface DashboardProps {
  health: SystemHealth | null;
  logs: DeliveryLog[];
  workflows: Workflow[];
  onTriggerSuccess: () => void;
  onNavigate: (tab: string) => void;
}

export const Dashboard: React.FC<DashboardProps> = ({
  health,
  logs,
  workflows,
  onTriggerSuccess,
  onNavigate,
}) => {
  const [selectedWf, setSelectedWf] = useState("");
  const [subscriberId, setSubscriberId] = useState("usr_demo");
  const [email, setEmail] = useState("alexandr@example.com");
  const [title, setTitle] = useState("Security Alert");
  const [body, setBody] = useState("A new login was detected from Frankfurt, Germany.");
  const [actionUrl, setActionUrl] = useState("https://example.com/security");
  const [isSending, setIsSending] = useState(false);
  const [triggerStatus, setTriggerStatus] = useState<string | null>(null);

  useEffect(() => {
    if (workflows.length > 0 && !selectedWf) {
      setSelectedWf(workflows[0].id);
    }
  }, [workflows, selectedWf]);

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedWf) return;
    setIsSending(true);
    setTriggerStatus(null);

    try {
      const resp = await triggerNotification({
        workflow: selectedWf,
        subscriber: {
          id: subscriberId,
          email: email || undefined,
        },
        payload: {
          title,
          body,
          action_url: actionUrl,
        },
      });
      setTriggerStatus(`Notification dispatched to: ${resp.dispatched_channels?.join(", ") || "in_app"}`);
      onTriggerSuccess();
    } catch (err: any) {
      setTriggerStatus(`Error: ${err.message}`);
    } finally {
      setIsSending(false);
    }
  };

  const sentCount = logs.filter((l) => l.status === "sent").length;
  const failedCount = logs.filter((l) => l.status === "failed").length;
  const openedCount = logs.filter((l) => !!l.opened_at).length;
  const clickedCount = logs.filter((l) => !!l.clicked_at).length;
  const openRate = sentCount > 0 ? ((openedCount / sentCount) * 100).toFixed(1) : "0.0";
  const ctrRate = sentCount > 0 ? ((clickedCount / sentCount) * 100).toFixed(1) : "0.0";

  return (
    <div className="space-y-6">
      {/* Metrics Row */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        <div className="bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm">
          <div className="flex items-center justify-between text-gray-400 mb-2">
            <span className="text-sm font-medium">Memory Allocation</span>
            <Cpu className="w-4 h-4 text-emerald-400" />
          </div>
          <div className="text-2xl font-bold text-white tracking-tight">
            {health?.memory_alloc_mb || 14.8} <span className="text-sm font-normal text-gray-400">MB</span>
          </div>
          <p className="text-xs text-gray-400 mt-1">Total System: {health?.memory_sys_mb || 26.2} MB (&lt; 45MB)</p>
        </div>

        <div className="bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm">
          <div className="flex items-center justify-between text-gray-400 mb-2">
            <span className="text-sm font-medium">Live SSE Listeners</span>
            <Radio className="w-4 h-4 text-indigo-400 animate-pulse" />
          </div>
          <div className="text-2xl font-bold text-white tracking-tight">
            {health?.active_sse_connections || 0}
          </div>
          <p className="text-xs text-gray-400 mt-1">Active client widgets receiving streams</p>
        </div>

        <div className="bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm">
          <div className="flex items-center justify-between text-gray-400 mb-2">
            <span className="text-sm font-medium">Dispatches</span>
            <CheckCircle2 className="w-4 h-4 text-emerald-400" />
          </div>
          <div className="text-2xl font-bold text-white tracking-tight">
            {sentCount} <span className="text-xs font-normal text-rose-400 ml-1">({failedCount} failed)</span>
          </div>
          <p className="text-xs text-gray-400 mt-1">Logged delivery dispatches</p>
        </div>

        <div className="bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm">
          <div className="flex items-center justify-between text-gray-400 mb-2">
            <span className="text-sm font-medium">Open Rate</span>
            <Eye className="w-4 h-4 text-sky-400" />
          </div>
          <div className="text-2xl font-bold text-white tracking-tight">
            {openRate}% <span className="text-xs font-normal text-gray-400 ml-1">({openedCount})</span>
          </div>
          <p className="text-xs text-gray-400 mt-1">Tracking pixel confirmations</p>
        </div>

        <div className="bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm">
          <div className="flex items-center justify-between text-gray-400 mb-2">
            <span className="text-sm font-medium">CTR (Click-Through)</span>
            <MousePointerClick className="w-4 h-4 text-amber-400" />
          </div>
          <div className="text-2xl font-bold text-white tracking-tight">
            {ctrRate}% <span className="text-xs font-normal text-gray-400 ml-1">({clickedCount})</span>
          </div>
          <p className="text-xs text-gray-400 mt-1">Verified redirect clicks</p>
        </div>
      </div>

      {/* Main Grid: Live Trigger Simulator & Recent Logs */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Trigger Simulator */}
        <div className="lg:col-span-5 bg-[#111827] border border-[#243247] rounded-xl p-6 shadow-sm">
          <div className="flex items-center gap-2 mb-4 pb-3 border-b border-[#243247]">
            <Send className="w-4 h-4 text-indigo-400" />
            <h2 className="text-base font-semibold text-white">Trigger Simulator</h2>
          </div>

          <form onSubmit={handleSend} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Workflow</label>
              <select
                value={selectedWf}
                onChange={(e) => setSelectedWf(e.target.value)}
                className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500"
              >
                {workflows.map((wf) => (
                  <option key={wf.id} value={wf.id}>
                    {wf.name} ({wf.id})
                  </option>
                ))}
              </select>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">Subscriber ID</label>
                <input
                  type="text"
                  value={subscriberId}
                  onChange={(e) => setSubscriberId(e.target.value)}
                  className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500"
                  required
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">Email</label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Subject / Title</label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500"
                required
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Body</label>
              <textarea
                value={body}
                onChange={(e) => setBody(e.target.value)}
                rows={3}
                className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500 resize-none"
                required
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Action URL</label>
              <input
                type="url"
                value={actionUrl}
                onChange={(e) => setActionUrl(e.target.value)}
                className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <button
              type="submit"
              disabled={isSending || !selectedWf}
              className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-medium py-2.5 px-4 rounded-lg text-sm flex items-center justify-center gap-2 transition-colors"
            >
              <Send className="w-4 h-4" />
              {isSending ? "Dispatching..." : "Fire Trigger (POST /v1/trigger)"}
            </button>

            {triggerStatus && (
              <div
                className={`p-3 rounded-lg text-xs font-mono ${
                  triggerStatus.startsWith("Error")
                    ? "bg-rose-500/10 text-rose-400 border border-rose-500/20"
                    : "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
                }`}
              >
                {triggerStatus}
              </div>
            )}
          </form>
        </div>

        {/* Live Delivery Activity */}
        <div className="lg:col-span-7 bg-[#111827] border border-[#243247] rounded-xl p-6 shadow-sm flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between mb-4 pb-3 border-b border-[#243247]">
              <div className="flex items-center gap-2">
                <Clock className="w-4 h-4 text-indigo-400" />
                <h2 className="text-base font-semibold text-white">Recent Delivery Activity</h2>
              </div>
              <button
                onClick={() => onNavigate("logs")}
                className="text-xs text-indigo-400 hover:text-indigo-300 font-medium flex items-center gap-1"
              >
                View all logs <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>

            {logs.length === 0 ? (
              <div className="text-center py-16 text-gray-500 text-sm">
                No notifications dispatched yet. Use the simulator to fire your first trigger!
              </div>
            ) : (
              <div className="divide-y divide-[#243247] overflow-hidden">
                {logs.slice(0, 7).map((log) => (
                  <div key={log.id} className="py-2.5 flex items-center justify-between text-xs">
                    <div className="flex items-center gap-3">
                      {log.status === "sent" ? (
                        <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
                      ) : (
                        <XCircle className="w-4 h-4 text-rose-400 shrink-0" />
                      )}
                      <div>
                        <div className="font-medium text-gray-200">
                          {log.workflow_id} <span className="text-gray-500">→</span> {log.subscriber_id}
                        </div>
                        <div className="text-gray-500 text-[11px] flex items-center gap-2">
                          <span>Channel: <span className="text-indigo-400 font-mono">{log.channel}</span></span>
                          {log.opened_at && (
                            <span className="text-sky-400 font-medium flex items-center gap-0.5">
                              <Eye className="w-3 h-3" /> Opened
                            </span>
                          )}
                          {log.clicked_at && (
                            <span className="text-amber-400 font-medium flex items-center gap-0.5">
                              <MousePointerClick className="w-3 h-3" /> Clicked
                            </span>
                          )}
                          {log.error_message && (
                            <span className="text-rose-400 ml-1">({log.error_message})</span>
                          )}
                        </div>
                      </div>
                    </div>
                    <span className="text-gray-500 text-[11px] font-mono shrink-0">
                      {new Date(log.dispatched_at).toLocaleTimeString()}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="pt-4 border-t border-[#243247] text-xs text-gray-400 flex items-center justify-between">
            <span>Server-Sent Events active and broadcasting</span>
            <span className="font-mono text-emerald-400">WAL Mode ON</span>
          </div>
        </div>
      </div>
    </div>
  );
};
