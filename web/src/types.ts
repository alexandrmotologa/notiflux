export interface Workflow {
  id: string;
  name: string;
  description?: string;
  channels: string[];
  created_at: string;
  updated_at: string;
}

export interface Template {
  id: string;
  workflow_id: string;
  channel: string;
  subject?: string;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface Subscriber {
  id: string;
  email?: string;
  telegram_chat_id?: string;
  slack_webhook_url?: string;
  web_push_subscription?: {
    endpoint: string;
    keys: {
      p256dh: string;
      auth: string;
    };
  };
  created_at: string;
  updated_at: string;
}

export interface DeliveryLog {
  id: string;
  notification_id?: string;
  workflow_id: string;
  subscriber_id: string;
  channel: string;
  status: "sent" | "failed" | "skipped";
  error_message?: string;
  payload?: string;
  dispatched_at: string;
}

export interface SystemHealth {
  status: string;
  version: string;
  uptime_seconds: number;
  memory_alloc_mb: number;
  memory_sys_mb: number;
  goroutines: number;
  active_sse_connections: number;
}
