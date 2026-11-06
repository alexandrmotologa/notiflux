import React, { useState, useEffect } from "react";
import { ScrollText, RefreshCw, CheckCircle2, XCircle, AlertCircle, ChevronDown, ChevronRight } from "lucide-react";
import type { DeliveryLog } from "../types";
import { fetchLogs } from "../api";

export const DeliveryLogsView: React.FC = () => {
  const [logs, setLogs] = useState<DeliveryLog[]>([]);
  const [channelFilter, setChannelFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [expandedLogId, setExpandedLogId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const load = async () => {
    setIsLoading(true);
    try {
      const data = await fetchLogs(100, channelFilter, statusFilter);
      setLogs(data.logs || []);
    } catch {
      // Handle error
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [channelFilter, statusFilter]);

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-white tracking-tight">Delivery Logs</h1>
          <p className="text-sm text-gray-400">Chronological history of notification dispatch attempts across all channels</p>
        </div>

        <div className="flex items-center gap-3">
          <select
            value={channelFilter}
            onChange={(e) => setChannelFilter(e.target.value)}
            className="bg-[#182234] border border-[#243247] rounded-lg px-3 py-1.5 text-xs text-gray-200 focus:outline-none focus:border-indigo-500"
          >
            <option value="">All Channels</option>
            <option value="in_app">In-App</option>
            <option value="email">Email</option>
            <option value="web_push">Web Push</option>
            <option value="telegram">Telegram</option>
            <option value="slack">Slack</option>
            <option value="webhook">Webhook</option>
          </select>

          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="bg-[#182234] border border-[#243247] rounded-lg px-3 py-1.5 text-xs text-gray-200 focus:outline-none focus:border-indigo-500"
          >
            <option value="">All Statuses</option>
            <option value="sent">Sent</option>
            <option value="failed">Failed</option>
            <option value="skipped">Skipped</option>
          </select>

          <button
            onClick={load}
            disabled={isLoading}
            className="bg-[#182234] hover:bg-[#243247] text-gray-300 font-medium py-1.5 px-3 rounded-lg text-xs flex items-center gap-1.5 border border-[#243247] transition-colors"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? "animate-spin" : ""}`} />
            Refresh
          </button>
        </div>
      </div>

      <div className="bg-[#111827] border border-[#243247] rounded-xl shadow-sm overflow-hidden">
        {logs.length === 0 ? (
          <div className="text-center py-20 text-gray-400 text-sm">
            <ScrollText className="w-8 h-8 text-gray-600 mx-auto mb-2" />
            No delivery logs match your filter criteria.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-[#182234] text-gray-400 border-b border-[#243247] uppercase font-semibold text-[11px] tracking-wider">
                <tr>
                  <th className="py-3 px-4 w-10"></th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4">Workflow</th>
                  <th className="py-3 px-4">Subscriber</th>
                  <th className="py-3 px-4">Channel</th>
                  <th className="py-3 px-4">Dispatched At</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#243247] text-gray-300">
                {logs.map((log) => {
                  const isExpanded = expandedLogId === log.id;
                  return (
                    <React.Fragment key={log.id}>
                      <tr
                        onClick={() => setExpandedLogId(isExpanded ? null : log.id)}
                        className="hover:bg-[#182234]/50 cursor-pointer transition-colors"
                      >
                        <td className="py-3 px-4 text-gray-400">
                          {isExpanded ? (
                            <ChevronDown className="w-4 h-4 text-indigo-400" />
                          ) : (
                            <ChevronRight className="w-4 h-4" />
                          )}
                        </td>
                        <td className="py-3 px-4">
                          {log.status === "sent" ? (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                              <CheckCircle2 className="w-3 h-3" /> Sent
                            </span>
                          ) : log.status === "failed" ? (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-rose-500/10 text-rose-400 border border-rose-500/20">
                              <XCircle className="w-3 h-3" /> Failed
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20">
                              <AlertCircle className="w-3 h-3" /> Skipped
                            </span>
                          )}
                        </td>
                        <td className="py-3 px-4 font-mono text-gray-200 font-medium">
                          {log.workflow_id}
                        </td>
                        <td className="py-3 px-4 font-mono text-gray-300">
                          {log.subscriber_id}
                        </td>
                        <td className="py-3 px-4">
                          <span className="px-2 py-0.5 rounded bg-[#182234] border border-[#243247] text-indigo-300 font-mono text-[11px]">
                            {log.channel}
                          </span>
                        </td>
                        <td className="py-3 px-4 font-mono text-gray-400">
                          {new Date(log.dispatched_at).toLocaleString()}
                        </td>
                      </tr>

                      {isExpanded && (
                        <tr className="bg-[#0b0f17]">
                          <td colSpan={6} className="p-4 space-y-3 border-b border-[#243247]">
                            {log.error_message && (
                              <div className="bg-rose-500/10 border border-rose-500/20 text-rose-300 p-3 rounded-lg text-xs font-mono">
                                <strong>Error:</strong> {log.error_message}
                              </div>
                            )}

                            <div>
                              <span className="text-[11px] font-semibold uppercase text-gray-400 tracking-wider block mb-1">
                                Dispatched Payload Context
                              </span>
                              <pre className="bg-[#182234] p-3 rounded-lg text-xs font-mono text-gray-300 overflow-x-auto border border-[#243247]">
                                {log.payload || "{}"}
                              </pre>
                            </div>
                          </td>
                        </tr>
                      )}
                    </React.Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};
