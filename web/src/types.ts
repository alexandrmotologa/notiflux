export interface WorkflowStep {
  channel: string;
  delay_seconds?: number;
  condition?: string;
}

export interface Workflow {
  id: string;
  name: string;
  description?: string;
  channels: string[];
  steps?: WorkflowStep[];
  digest_enabled?: boolean;
  digest_window_seconds?: number;
  digest_max_items?: number;
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
  discord_webhook_url?: string;
  web_push_subscription?: {
    endpoint: string;
    keys: {
      p256dh: string;
      auth: string;
    };
  };
  timezone?: string;
  quiet_hours_start?: string;
  quiet_hours_end?: string;
  created_at: string;
  updated_at: string;
}

export interface DeliveryLog {
  id: string;
  notification_id?: string;
  workflow_id: string;
  subscriber_id: string;
  channel: string;
  status: "sent" | "failed" | "skipped" | "scheduled";
  error_message?: string;
  payload?: string;
  tracking_id?: string;
  target_url?: string;
  opened_at?: string;
  clicked_at?: string;
  dispatched_at: string;
}

export interface APIKeyRecord {
  key: string;
  name: string;
  created_at: string;
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
