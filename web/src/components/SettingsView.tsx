import React, { useState, useEffect } from "react";
import { Key, Send, Copy, Check, ShieldCheck, Database, Plus, Trash2 } from "lucide-react";
import {
  generateVAPID,
  testChannel,
  fetchAPIKeys,
  createAPIKey,
  deleteAPIKey,
  triggerHotBackup,
} from "../api";
import type { APIKeyRecord } from "../types";

export const SettingsView: React.FC = () => {
  const [vapidKeys, setVapidKeys] = useState<{ public_key: string; private_key: string } | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [testTargetChannel, setTestTargetChannel] = useState("email");
  const [testRecipient, setTestRecipient] = useState("admin@example.com");
  const [testResult, setTestResult] = useState<string | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [isTesting, setIsTesting] = useState(false);

  // API Keys state
  const [apiKeys, setApiKeys] = useState<APIKeyRecord[]>([]);
  const [newKeyName, setNewKeyName] = useState("");
  const [isCreatingKey, setIsCreatingKey] = useState(false);
  const [newlyCreatedKey, setNewlyCreatedKey] = useState<string | null>(null);

  // Backup state
  const [isBackingUp, setIsBackingUp] = useState(false);
  const [backupResult, setBackupResult] = useState<{ backup_file: string; timestamp: string } | null>(null);

  const loadKeys = async () => {
    try {
      const res = await fetchAPIKeys();
      setApiKeys(res.keys || []);
    } catch {
      // Handle error
    }
  };

  useEffect(() => {
    loadKeys();
  }, []);

  const handleGenerateVAPID = async () => {
    setIsGenerating(true);
    try {
      const keys = await generateVAPID();
      setVapidKeys(keys);
    } catch {
      // Handle error
    } finally {
      setIsGenerating(false);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(label);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const handleCreateAPIKey = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newKeyName.trim()) return;
    setIsCreatingKey(true);
    setNewlyCreatedKey(null);
    try {
      const created = await createAPIKey(newKeyName.trim());
      setNewlyCreatedKey(created.key);
      setNewKeyName("");
      loadKeys();
    } catch (err: any) {
      alert(`Failed to create API key: ${err.message}`);
    } finally {
      setIsCreatingKey(false);
    }
  };

  const handleDeleteAPIKey = async (key: string) => {
    if (confirm("Are you sure you want to revoke this API key? This action is immediate.")) {
      try {
        await deleteAPIKey(key);
        loadKeys();
      } catch (err: any) {
        alert(`Failed to delete API key: ${err.message}`);
      }
    }
  };

  const handleHotBackup = async () => {
    setIsBackingUp(true);
    setBackupResult(null);
    try {
      const res = await triggerHotBackup();
      setBackupResult({
        backup_file: res.backup_file,
        timestamp: res.timestamp,
      });
    } catch (err: any) {
      alert(`Backup failed: ${err.message}`);
    } finally {
      setIsBackingUp(false);
    }
  };

  const handleTestChannel = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsTesting(true);
    setTestResult(null);

    const sub: any = { id: "test_admin" };
    if (testTargetChannel === "email") sub.email = testRecipient;
    if (testTargetChannel === "telegram") sub.telegram_chat_id = testRecipient;
    if (testTargetChannel === "slack") sub.slack_webhook_url = testRecipient;
    if (testTargetChannel === "discord") sub.discord_webhook_url = testRecipient;

    try {
      const res = await testChannel(testTargetChannel, sub);
      if (res.success) {
        setTestResult(`Success! Dispatched test ping to channel: ${testTargetChannel}`);
      } else {
        setTestResult(`Failed: ${res.error || "Channel dispatch rejected"}`);
      }
    } catch (err: any) {
      setTestResult(`Error: ${err.message}`);
    } finally {
      setIsTesting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-white tracking-tight">System Settings &amp; Security</h1>
        <p className="text-sm text-gray-400">
          API keys, online hot backups, VAPID cryptographic generation, and channel connectivity verification
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Workspace API Keys */}
        <div className="bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm space-y-4">
          <div className="flex items-center justify-between pb-3 border-b border-[#243247]">
            <div className="flex items-center gap-2">
              <Key className="w-4 h-4 text-indigo-400" />
              <h2 className="text-sm font-semibold text-white">Workspace API Keys</h2>
            </div>
          </div>

          <p className="text-xs text-gray-400">
            Generate scoped or service-specific API keys. Authenticate via{" "}
            <code className="text-indigo-300 font-mono">Authorization: Bearer &lt;key&gt;</code>.
          </p>

          <form onSubmit={handleCreateAPIKey} className="flex gap-2">
            <input
              type="text"
              placeholder="e.g. Production Backend or CI Pipeline"
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
              className="flex-1 bg-[#182234] border border-[#243247] rounded-lg px-3 py-1.5 text-xs text-gray-200 focus:outline-none focus:border-indigo-500"
              required
            />
            <button
              type="submit"
              disabled={isCreatingKey}
              className="bg-indigo-600 hover:bg-indigo-500 text-white font-medium py-1.5 px-3 rounded-lg text-xs flex items-center gap-1.5 transition-colors disabled:opacity-50"
            >
              <Plus className="w-3.5 h-3.5" />
              {isCreatingKey ? "Generating..." : "Generate Key"}
            </button>
          </form>

          {newlyCreatedKey && (
            <div className="p-3 bg-emerald-500/10 border border-emerald-500/30 rounded-lg space-y-1">
              <div className="flex items-center justify-between text-xs text-emerald-400 font-medium">
                <span>New API Key Generated:</span>
                <button
                  onClick={() => copyToClipboard(newlyCreatedKey, "new-key")}
                  className="text-emerald-400 hover:text-emerald-300 flex items-center gap-1 text-[11px]"
                >
                  {copiedKey === "new-key" ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                  {copiedKey === "new-key" ? "Copied!" : "Copy Key"}
                </button>
              </div>
              <div className="font-mono text-xs text-gray-200 break-all select-all">
                {newlyCreatedKey}
              </div>
            </div>
          )}

          <div className="divide-y divide-[#243247] overflow-hidden max-h-56 overflow-y-auto">
            {apiKeys.length === 0 ? (
              <div className="text-center py-6 text-xs text-gray-500">
                No custom keys created yet. Default server API key active from environment.
              </div>
            ) : (
              apiKeys.map((k) => (
                <div key={k.key} className="py-2.5 flex items-center justify-between text-xs">
                  <div>
                    <div className="font-medium text-gray-200">{k.name}</div>
                    <div className="font-mono text-gray-500 text-[11px]">
                      {k.key.substring(0, 12)}...{k.key.substring(k.key.length - 4)}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => copyToClipboard(k.key, k.key)}
                      className="p-1 text-gray-400 hover:text-gray-200 rounded"
                      title="Copy Key"
                    >
                      {copiedKey === k.key ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                    </button>
                    <button
                      onClick={() => handleDeleteAPIKey(k.key)}
                      className="p-1 text-gray-400 hover:text-rose-400 rounded"
                      title="Revoke Key"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Online Hot Backup */}
        <div className="bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm space-y-4">
          <div className="flex items-center justify-between pb-3 border-b border-[#243247]">
            <div className="flex items-center gap-2">
              <Database className="w-4 h-4 text-emerald-400" />
              <h2 className="text-sm font-semibold text-white">Online Hot SQLite Backup</h2>
            </div>
            <button
              onClick={handleHotBackup}
              disabled={isBackingUp}
              className="bg-emerald-600 hover:bg-emerald-500 text-white font-medium py-1.5 px-3 rounded-lg text-xs flex items-center gap-1.5 transition-colors disabled:opacity-50"
            >
              <Database className="w-3.5 h-3.5" />
              {isBackingUp ? "Backing up..." : "Create Hot Backup"}
            </button>
          </div>

          <p className="text-xs text-gray-400">
            Performs an atomic online point-in-time snapshot of the database using SQLite VACUUM INTO, safe during
            concurrent WAL reads and writes with zero downtime.
          </p>

          {backupResult && (
            <div className="p-3 bg-emerald-500/10 border border-emerald-500/30 rounded-lg text-xs space-y-1">
              <div className="font-semibold text-emerald-400">Hot Backup Created Successfully</div>
              <div className="text-gray-300 font-mono text-[11px]">
                Target: {backupResult.backup_file}
              </div>
              <div className="text-gray-400 text-[10px]">
                Timestamp: {new Date(backupResult.timestamp).toLocaleString()}
              </div>
            </div>
          )}

          <div className="bg-[#182234] border border-[#243247] rounded-lg p-3 text-xs space-y-1.5 text-gray-400">
            <div className="font-semibold text-gray-300 mb-1">CLI Hot Backup Subcommand:</div>
            <code className="text-indigo-300 font-mono text-[11px] block select-all">
              notiflux backup --out /var/backups/notiflux_daily.db
            </code>
          </div>
        </div>

        {/* RFC 8292 VAPID Generator */}
        <div className="bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm space-y-4">
          <div className="flex items-center justify-between pb-3 border-b border-[#243247]">
            <div className="flex items-center gap-2">
              <Key className="w-4 h-4 text-indigo-400" />
              <h2 className="text-sm font-semibold text-white">RFC 8292 VAPID Keypair Generator</h2>
            </div>
            <button
              onClick={handleGenerateVAPID}
              disabled={isGenerating}
              className="bg-indigo-600 hover:bg-indigo-500 text-white font-medium py-1.5 px-3 rounded-lg text-xs flex items-center gap-1.5 transition-colors"
            >
              {isGenerating ? "Generating..." : "Generate New Keys"}
            </button>
          </div>

          <p className="text-xs text-gray-400">
            Browser Web Push notifications require ECDSA P-256 keys. Pass these as environment variables{" "}
            <code className="text-indigo-300 font-mono">NOTIFLUX_VAPID_PUBLIC_KEY</code> and{" "}
            <code className="text-indigo-300 font-mono">NOTIFLUX_VAPID_PRIVATE_KEY</code>.
          </p>

          {vapidKeys ? (
            <div className="space-y-3 pt-2">
              <div>
                <div className="flex items-center justify-between text-xs text-gray-400 mb-1">
                  <span>Public Key (Client &amp; Server)</span>
                  <button
                    onClick={() => copyToClipboard(vapidKeys.public_key, "public")}
                    className="text-indigo-400 hover:text-indigo-300 flex items-center gap-1 text-[11px]"
                  >
                    {copiedKey === "public" ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                    {copiedKey === "public" ? "Copied!" : "Copy"}
                  </button>
                </div>
                <div className="bg-[#182234] border border-[#243247] rounded-lg p-2.5 font-mono text-[11px] text-gray-300 break-all select-all">
                  {vapidKeys.public_key}
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between text-xs text-gray-400 mb-1">
                  <span>Private Key (Server Only)</span>
                  <button
                    onClick={() => copyToClipboard(vapidKeys.private_key, "private")}
                    className="text-indigo-400 hover:text-indigo-300 flex items-center gap-1 text-[11px]"
                  >
                    {copiedKey === "private" ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                    {copiedKey === "private" ? "Copied!" : "Copy"}
                  </button>
                </div>
                <div className="bg-[#182234] border border-[#243247] rounded-lg p-2.5 font-mono text-[11px] text-gray-300 break-all select-all">
                  {vapidKeys.private_key}
                </div>
              </div>
            </div>
          ) : (
            <div className="text-center py-10 bg-[#182234]/50 border border-dashed border-[#243247] rounded-lg text-xs text-gray-500">
              Click &quot;Generate New Keys&quot; to produce an ECDSA P-256 pair.
            </div>
          )}
        </div>

        {/* Channel Test Dispatcher */}
        <div className="bg-[#111827] border border-[#243247] rounded-xl p-5 shadow-sm space-y-4">
          <div className="flex items-center gap-2 pb-3 border-b border-[#243247]">
            <ShieldCheck className="w-4 h-4 text-emerald-400" />
            <h2 className="text-sm font-semibold text-white">Channel Connectivity Verification</h2>
          </div>

          <form onSubmit={handleTestChannel} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Channel to Test</label>
              <select
                value={testTargetChannel}
                onChange={(e) => setTestTargetChannel(e.target.value)}
                className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500"
              >
                <option value="email">Email (SMTP / Resend)</option>
                <option value="telegram">Telegram Bot</option>
                <option value="slack">Slack Webhook</option>
                <option value="discord">Discord Webhook</option>
              </select>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">
                {testTargetChannel === "email"
                  ? "Recipient Email Address"
                  : testTargetChannel === "telegram"
                  ? "Telegram Chat ID"
                  : testTargetChannel === "slack"
                  ? "Slack Webhook URL (optional override)"
                  : "Discord Webhook URL (optional override)"}
              </label>
              <input
                type="text"
                value={testRecipient}
                onChange={(e) => setTestRecipient(e.target.value)}
                className="w-full bg-[#182234] border border-[#243247] rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-indigo-500"
                required
              />
            </div>

            <button
              type="submit"
              disabled={isTesting}
              className="w-full bg-[#182234] hover:bg-indigo-600/20 text-indigo-400 border border-indigo-500/30 font-medium py-2 px-4 rounded-lg text-xs flex items-center justify-center gap-2 transition-colors"
            >
              <Send className="w-3.5 h-3.5" />
              {isTesting ? "Sending Test Ping..." : "Send Test Ping"}
            </button>

            {testResult && (
              <div
                className={`p-3 rounded-lg text-xs font-mono ${
                  testResult.startsWith("Success")
                    ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
                    : "bg-rose-500/10 text-rose-400 border border-rose-500/20"
                }`}
              >
                {testResult}
              </div>
            )}
          </form>
        </div>
      </div>
    </div>
  );
};
