import type { Workflow, Template, Subscriber, DeliveryLog, SystemHealth } from "./types";

const BASE_URL = "";

export async function fetchHealth(): Promise<SystemHealth> {
  const res = await fetch(`${BASE_URL}/v1/health`);
  return res.json();
}

export async function fetchWorkflows(): Promise<Workflow[]> {
  const res = await fetch(`${BASE_URL}/v1/workflows`);
  return res.json();
}

export async function createWorkflow(wf: Partial<Workflow>): Promise<Workflow> {
  const res = await fetch(`${BASE_URL}/v1/workflows`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(wf),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || "Failed creating workflow");
  }
  return res.json();
}

export async function deleteWorkflow(id: string): Promise<void> {
  const res = await fetch(`${BASE_URL}/v1/workflows/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error("Failed deleting workflow");
}

export async function fetchTemplates(workflowId: string): Promise<Template[]> {
  const res = await fetch(`${BASE_URL}/v1/workflows/${workflowId}/templates`);
  return res.json();
}

export async function updateTemplates(workflowId: string, templates: { channel: string; subject: string; body: string }[]): Promise<void> {
  const res = await fetch(`${BASE_URL}/v1/workflows/${workflowId}/templates`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ templates }),
  });
  if (!res.ok) throw new Error("Failed updating templates");
}

export async function previewTemplate(channel: string, subject: string, body: string, data: Record<string, any>) {
  const res = await fetch(`${BASE_URL}/v1/templates/preview`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ channel, subject, body, data }),
  });
  return res.json();
}

export async function fetchLogs(limit = 50, channel = "", status = ""): Promise<{ logs: DeliveryLog[]; total: number }> {
  const params = new URLSearchParams({ limit: String(limit) });
  if (channel) params.set("channel", channel);
  if (status) params.set("status", status);
  const res = await fetch(`${BASE_URL}/v1/logs?${params}`);
  return res.json();
}

export async function fetchSubscribers(limit = 50): Promise<{ subscribers: Subscriber[]; total: number }> {
  const res = await fetch(`${BASE_URL}/v1/subscribers?limit=${limit}`);
  return res.json();
}

export async function fetchSubscriber(id: string): Promise<{ subscriber: Subscriber; preferences: Record<string, boolean>; unread_count: number }> {
  const res = await fetch(`${BASE_URL}/v1/subscribers/${id}`);
  return res.json();
}

export async function updateSubscriberPreferences(id: string, channels: Record<string, boolean>) {
  const res = await fetch(`${BASE_URL}/v1/preferences/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ channels }),
  });
  return res.json();
}

export async function triggerNotification(payload: any) {
  const res = await fetch(`${BASE_URL}/v1/trigger`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error || "Trigger failed");
  }
  return data;
}

export async function generateVAPID(): Promise<{ public_key: string; private_key: string }> {
  const res = await fetch(`${BASE_URL}/v1/system/vapid`, { method: "POST" });
  return res.json();
}

export async function testChannel(channel: string, subscriber: Partial<Subscriber>) {
  const res = await fetch(`${BASE_URL}/v1/system/test-channel`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ channel, subscriber }),
  });
  return res.json();
}
