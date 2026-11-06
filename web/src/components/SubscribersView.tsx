import React, { useState, useEffect } from "react";
import {
  Users,
  Mail,
  Send,
  Smartphone,
  MessageSquare,
  Bell,
  Check,
  X,
  RefreshCw,
  Moon,
  Globe,
  Radio,
} from "lucide-react";
import type { Subscriber } from "../types";
import { fetchSubscribers, fetchSubscriber, updateSubscriberPreferences } from "../api";

export const SubscribersView: React.FC = () => {
  const [subscribers, setSubscribers] = useState<Subscriber[]>([]);
  const [selectedSubId, setSelectedSubId] = useState<string | null>(null);
  const [selectedDetails, setSelectedDetails] = useState<{
    subscriber: Subscriber;
    preferences: Record<string, boolean>;
    unread_count: number;
  } | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const load = async () => {
    setIsLoading(true);
    try {
      const data = await fetchSubscribers(50);
      setSubscribers(data.subscribers || []);
    } catch {
      // Handle error
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const selectSubscriber = async (id: string) => {
    setSelectedSubId(id);
    try {
      const data = await fetchSubscriber(id);
      setSelectedDetails(data);
    } catch {
      // Handle error
    }
  };

  const toggleChannelPref = async (channel: string) => {
    if (!selectedDetails || !selectedSubId) return;
    const currentStatus = selectedDetails.preferences[channel] !== false;
    const updatedPrefs = {
      ...selectedDetails.preferences,
      [channel]: !currentStatus,
    };

    setSelectedDetails({
      ...selectedDetails,
      preferences: updatedPrefs,
    });

    try {
      await updateSubscriberPreferences(selectedSubId, { [channel]: !currentStatus });
    } catch {
      // Revert on error
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-white tracking-tight">Subscribers</h1>
          <p className="text-sm text-gray-400">
            Recipient profiles, credential registry, quiet hours, and notification channel opt-ins
          </p>
        </div>
        <button
          onClick={load}
          disabled={isLoading}
          className="bg-[#182234] hover:bg-[#243247] text-gray-300 font-medium py-1.5 px-3 rounded-lg text-xs flex items-center gap-1.5 border border-[#243247] transition-colors"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? "animate-spin" : ""}`} />
          Refresh
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Subscribers Table */}
        <div className="lg:col-span-7 bg-[#111827] border border-[#243247] rounded-xl shadow-sm overflow-hidden">
          {subscribers.length === 0 ? (
            <div className="text-center py-20 text-gray-400 text-sm">
              <Users className="w-8 h-8 text-gray-600 mx-auto mb-2" />
              No subscribers registered yet. Trigger a notification to automatically create one.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-[#182234] text-gray-400 border-b border-[#243247] uppercase font-semibold text-[11px] tracking-wider">
                  <tr>
                    <th className="py-3 px-4">Subscriber ID</th>
                    <th className="py-3 px-4">Email</th>
                    <th className="py-3 px-4">Channels</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#243247] text-gray-300">
                  {subscribers.map((sub) => {
                    const isSelected = selectedSubId === sub.id;
                    return (
                      <tr
                        key={sub.id}
                        onClick={() => selectSubscriber(sub.id)}
                        className={`hover:bg-[#182234]/60 cursor-pointer transition-colors ${
                          isSelected ? "bg-indigo-600/10 border-l-2 border-indigo-500" : ""
                        }`}
                      >
                        <td className="py-3 px-4 font-mono font-medium text-white">
                          {sub.id}
                        </td>
                        <td className="py-3 px-4 text-gray-400">
                          {sub.email || "-"}
                        </td>
                        <td className="py-3 px-4 flex items-center gap-1.5 text-gray-400">
                          {sub.email && (
                            <span title="Email registered">
                              <Mail className="w-3.5 h-3.5 text-indigo-400" />
                            </span>
                          )}
                          {sub.telegram_chat_id && (
                            <span title="Telegram registered">
                              <Send className="w-3.5 h-3.5 text-sky-400" />
                            </span>
                          )}
                          {sub.web_push_subscription && (
                            <span title="Web Push registered">
                              <Smartphone className="w-3.5 h-3.5 text-emerald-400" />
                            </span>
                          )}
                          {sub.slack_webhook_url && (
                            <span title="Slack registered">
                              <MessageSquare className="w-3.5 h-3.5 text-amber-400" />
                            </span>
                          )}
                          {sub.discord_webhook_url && (
                            <span title="Discord registered">
                              <Radio className="w-3.5 h-3.5 text-indigo-300" />
                            </span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Selected Subscriber Inspector */}
        <div className="lg:col-span-5 bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm space-y-4">
          <h2 className="text-sm font-semibold text-white pb-3 border-b border-[#243247]">
            Recipient Profile &amp; Preferences
          </h2>

          {!selectedDetails ? (
            <div className="text-center py-16 text-gray-500 text-xs">
              Select a subscriber to inspect registered channel credentials, quiet hours, and toggle opt-ins.
            </div>
          ) : (
            <div className="space-y-4">
              <div className="bg-[#182234] border border-[#243247] rounded-lg p-3 text-xs space-y-2">
                <div className="flex justify-between">
                  <span className="text-gray-400">Subscriber ID:</span>
                  <span className="font-mono text-white font-medium">{selectedDetails.subscriber.id}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">Email:</span>
                  <span className="text-gray-300">{selectedDetails.subscriber.email || "None"}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">Telegram Chat:</span>
                  <span className="text-gray-300">{selectedDetails.subscriber.telegram_chat_id || "None"}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">Discord Webhook:</span>
                  <span className="text-gray-300 font-mono text-[11px] truncate max-w-[200px]">
                    {selectedDetails.subscriber.discord_webhook_url || "None"}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-gray-400 flex items-center gap-1">
                    <Globe className="w-3 h-3 text-gray-400" /> Timezone:
                  </span>
                  <span className="text-gray-300">{selectedDetails.subscriber.timezone || "UTC"}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-gray-400 flex items-center gap-1">
                    <Moon className="w-3 h-3 text-indigo-400" /> Quiet Hours:
                  </span>
                  <span className="text-gray-300">
                    {selectedDetails.subscriber.quiet_hours_start && selectedDetails.subscriber.quiet_hours_end
                      ? `${selectedDetails.subscriber.quiet_hours_start} to ${selectedDetails.subscriber.quiet_hours_end}`
                      : "Disabled"}
                  </span>
                </div>
                <div className="flex justify-between pt-1 border-t border-[#243247]">
                  <span className="text-gray-400">In-App Unread Count:</span>
                  <span className="text-indigo-400 font-bold">{selectedDetails.unread_count}</span>
                </div>
              </div>

              <div>
                <span className="text-xs font-medium uppercase tracking-wider text-gray-400 block mb-2">
                  Channel Opt-In Preferences
                </span>
                <div className="space-y-2">
                  {[
                    { id: "in_app", label: "In-App Notification Bell", icon: Bell },
                    { id: "email", label: "Transactional Email", icon: Mail },
                    { id: "web_push", label: "Web Push Notifications", icon: Smartphone },
                    { id: "telegram", label: "Telegram Messages", icon: Send },
                    { id: "slack", label: "Slack Alerts", icon: MessageSquare },
                    { id: "discord", label: "Discord Webhook", icon: Radio },
                  ].map((ch) => {
                    const isEnabled = selectedDetails.preferences[ch.id] !== false;
                    const Icon = ch.icon;
                    return (
                      <div
                        key={ch.id}
                        onClick={() => toggleChannelPref(ch.id)}
                        className="flex items-center justify-between p-2.5 rounded-lg bg-[#182234] border border-[#243247] hover:border-gray-600 cursor-pointer text-xs"
                      >
                        <div className="flex items-center gap-2">
                          <Icon className="w-4 h-4 text-gray-400" />
                          <span className="font-medium text-gray-200">{ch.label}</span>
                        </div>
                        <span
                          className={`px-2 py-0.5 rounded text-[10px] font-semibold flex items-center gap-1 ${
                            isEnabled
                              ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
                              : "bg-rose-500/10 text-rose-400 border border-rose-500/20"
                          }`}
                        >
                          {isEnabled ? <Check className="w-3 h-3" /> : <X className="w-3 h-3" />}
                          {isEnabled ? "Enabled" : "Muted"}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
