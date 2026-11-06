import React, { useState } from "react";
import { Key, Send, Copy, Check, ShieldCheck } from "lucide-react";
import { generateVAPID, testChannel } from "../api";

export const SettingsView: React.FC = () => {
  const [vapidKeys, setVapidKeys] = useState<{ public_key: string; private_key: string } | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [testTargetChannel, setTestTargetChannel] = useState("email");
  const [testRecipient, setTestRecipient] = useState("admin@example.com");
  const [testResult, setTestResult] = useState<string | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [isTesting, setIsTesting] = useState(false);

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

  const handleTestChannel = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsTesting(true);
    setTestResult(null);

    const sub: any = { id: "test_admin" };
    if (testTargetChannel === "email") sub.email = testRecipient;
    if (testTargetChannel === "telegram") sub.telegram_chat_id = testRecipient;
    if (testTargetChannel === "slack") sub.slack_webhook_url = testRecipient;

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
        <h1 className="text-xl font-bold text-white tracking-tight">Channel Settings &amp; VAPID</h1>
        <p className="text-sm text-gray-400">Cryptographic key generation and channel connectivity testing</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* VAPID Generator */}
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
              </select>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">
                {testTargetChannel === "email"
                  ? "Recipient Email Address"
                  : testTargetChannel === "telegram"
                  ? "Telegram Chat ID"
                  : "Slack Webhook URL (optional override)"}
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
