export interface NotificationItem {
  id: string;
  workflow_id: string;
  title: string;
  body: string;
  action_url?: string;
  is_read: boolean;
  created_at: string;
}

const styles = `
:host {
  display: inline-block;
  position: relative;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  box-sizing: border-box;
}

*, *::before, *::after {
  box-sizing: inherit;
}

.bell-btn {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: 1px solid var(--nf-border, #374151);
  background: var(--nf-bg, #1f2937);
  color: var(--nf-text, #f3f4f6);
  cursor: pointer;
  outline: none;
  transition: all 0.2s ease;
}

.bell-btn:hover {
  background: var(--nf-hover, #374151);
  border-color: var(--nf-accent, #6366f1);
}

.bell-icon {
  width: 20px;
  height: 20px;
  fill: none;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.badge {
  position: absolute;
  top: -4px;
  right: -4px;
  background: var(--nf-badge-bg, #ef4444);
  color: #ffffff;
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  padding: 3px 6px;
  border-radius: 9999px;
  border: 2px solid var(--nf-bg, #111827);
  animation: nf-pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

.badge[hidden] {
  display: none !important;
}

@keyframes nf-pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.1); }
}

.popover {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  width: 360px;
  max-width: calc(100vw - 32px);
  max-height: 480px;
  background: var(--nf-panel-bg, #111827);
  border: 1px solid var(--nf-border, #374151);
  border-radius: 12px;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5), 0 8px 10px -6px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  z-index: 999999;
  overflow: hidden;
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.popover[hidden] {
  display: none !important;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--nf-border, #374151);
}

.header-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--nf-text, #f9fafb);
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-actions button {
  background: none;
  border: none;
  color: var(--nf-accent, #818cf8);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 4px;
}

.header-actions button:hover {
  background: rgba(99, 102, 241, 0.1);
}

.tabs {
  display: flex;
  padding: 4px 16px;
  gap: 12px;
  border-bottom: 1px solid var(--nf-border, #1f2937);
  background: var(--nf-tabs-bg, #111827);
}

.tab {
  background: none;
  border: none;
  padding: 6px 0;
  color: var(--nf-muted, #9ca3af);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  border-bottom: 2px solid transparent;
}

.tab.active {
  color: var(--nf-accent, #818cf8);
  border-bottom-color: var(--nf-accent, #818cf8);
}

.list {
  flex: 1;
  overflow-y: auto;
  margin: 0;
  padding: 0;
  list-style: none;
}

.item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--nf-border-subtle, #1f2937);
  cursor: pointer;
  transition: background 0.15s ease;
}

.item:hover {
  background: var(--nf-hover, #1f2937);
}

.item.unread {
  background: rgba(99, 102, 241, 0.05);
}

.unread-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--nf-accent, #6366f1);
  margin-top: 5px;
  flex-shrink: 0;
}

.item-content {
  flex: 1;
  min-width: 0;
}

.item-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--nf-text, #f9fafb);
  margin-bottom: 2px;
}

.item-body {
  font-size: 12px;
  color: var(--nf-muted, #9ca3af);
  line-height: 1.4;
  margin-bottom: 4px;
  word-break: break-word;
}

.item-time {
  font-size: 11px;
  color: var(--nf-dim, #6b7280);
}

.empty {
  padding: 32px 16px;
  text-align: center;
  color: var(--nf-muted, #9ca3af);
  font-size: 13px;
}
`;

export class NotifluxBell extends HTMLElement {
  private shadow: ShadowRoot;
  private unreadCount = 0;
  private notifications: NotificationItem[] = [];
  private activeTab: "all" | "unread" = "all";
  private isOpen = false;
  private eventSource: EventSource | null = null;
  private reconnectTimeout = 1000;

  constructor() {
    super();
    this.shadow = this.attachShadow({ mode: "open" });
  }

  static get observedAttributes() {
    return ["api-url", "subscriber-id", "theme"];
  }

  attributeChangedCallback(name: string, oldVal: string, newVal: string) {
    if (oldVal !== newVal) {
      if (name === "subscriber-id" || name === "api-url") {
        this.initConnection();
        this.fetchFeed();
      }
      this.render();
    }
  }

  connectedCallback() {
    this.render();
    this.initConnection();
    this.fetchFeed();

    // Close on outside click
    document.addEventListener("click", this.handleDocumentClick);
  }

  disconnectedCallback() {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    document.removeEventListener("click", this.handleDocumentClick);
  }

  private handleDocumentClick = (e: MouseEvent) => {
    if (!this.contains(e.target as Node) && this.isOpen) {
      this.isOpen = false;
      this.render();
    }
  };

  private get apiUrl(): string {
    return this.getAttribute("api-url") || "http://localhost:8085";
  }

  private get subscriberId(): string {
    return this.getAttribute("subscriber-id") || "";
  }

  private async fetchFeed() {
    if (!this.subscriberId) return;
    try {
      const res = await fetch(`${this.apiUrl}/v1/feed?subscriber_id=${encodeURIComponent(this.subscriberId)}&limit=30`);
      if (res.ok) {
        const data = await res.json();
        this.notifications = data.notifications || [];
        this.unreadCount = data.unread_count || 0;
        this.render();
      }
    } catch {
      // Offline or network error
    }
  }

  private initConnection() {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    if (!this.subscriberId) return;

    const streamUrl = `${this.apiUrl}/v1/feed/stream?subscriber_id=${encodeURIComponent(this.subscriberId)}`;
    this.eventSource = new EventSource(streamUrl);

    this.eventSource.addEventListener("connected", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        if (typeof data.unread_count === "number") {
          this.unreadCount = data.unread_count;
          this.render();
        }
      } catch {}
      this.reconnectTimeout = 1000;
    });

    this.eventSource.addEventListener("notification.new", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        if (data.notification) {
          this.notifications.unshift(data.notification);
        }
        if (typeof data.unread_count === "number") {
          this.unreadCount = data.unread_count;
        } else {
          this.unreadCount++;
        }
        this.render();
        this.dispatchEvent(new CustomEvent("notification-received", { detail: data.notification }));
      } catch {}
    });

    this.eventSource.addEventListener("notification.read", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        const item = this.notifications.find((n) => n.id === data.id);
        if (item) item.is_read = true;
        if (typeof data.unread_count === "number") {
          this.unreadCount = data.unread_count;
        }
        this.render();
      } catch {}
    });

    this.eventSource.addEventListener("notification.read_all", () => {
      this.notifications.forEach((n) => (n.is_read = true));
      this.unreadCount = 0;
      this.render();
    });

    this.eventSource.onerror = () => {
      if (this.eventSource) {
        this.eventSource.close();
        this.eventSource = null;
      }
      setTimeout(() => this.initConnection(), Math.min(this.reconnectTimeout *= 2, 16000));
    };
  }

  private async markAsRead(item: NotificationItem) {
    if (item.is_read) return;
    item.is_read = true;
    this.unreadCount = Math.max(0, this.unreadCount - 1);
    this.render();

    try {
      await fetch(`${this.apiUrl}/v1/feed/${item.id}/read`, { method: "PATCH" });
    } catch {}
  }

  private async markAllRead() {
    this.notifications.forEach((n) => (n.is_read = true));
    this.unreadCount = 0;
    this.render();

    try {
      await fetch(`${this.apiUrl}/v1/feed/read-all`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ subscriber_id: this.subscriberId }),
      });
    } catch {}
  }

  private handleItemClick(item: NotificationItem) {
    this.markAsRead(item);
    this.dispatchEvent(new CustomEvent("notification-click", { detail: item }));
    if (item.action_url) {
      window.open(item.action_url, "_blank");
    }
  }

  private formatTime(isoStr: string): string {
    const diff = Math.floor((Date.now() - new Date(isoStr).getTime()) / 1000);
    if (diff < 60) return "just now";
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return `${Math.floor(diff / 86400)}d ago`;
  }

  private render() {
    const filtered = this.activeTab === "unread"
      ? this.notifications.filter((n) => !n.is_read)
      : this.notifications;

    const badgeText = this.unreadCount > 99 ? "99+" : String(this.unreadCount);

    this.shadow.innerHTML = `
      <style>${styles}</style>
      <button class="bell-btn" aria-label="Notifications" id="nf-bell-btn">
        <svg class="bell-icon" viewBox="0 0 24 24">
          <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path>
          <path d="M13.73 21a2 2 0 0 1-3.46 0"></path>
        </svg>
        <span class="badge" ${this.unreadCount === 0 ? "hidden" : ""}>${badgeText}</span>
      </button>

      <div class="popover" ${!this.isOpen ? "hidden" : ""} id="nf-popover">
        <div class="header">
          <div class="header-title">
            <span>Notifications</span>
          </div>
          <div class="header-actions">
            ${this.unreadCount > 0 ? `<button id="nf-mark-all">Mark all as read</button>` : ""}
          </div>
        </div>

        <div class="tabs">
          <button class="tab ${this.activeTab === "all" ? "active" : ""}" id="nf-tab-all">All</button>
          <button class="tab ${this.activeTab === "unread" ? "active" : ""}" id="nf-tab-unread">
            Unread ${this.unreadCount > 0 ? `(${this.unreadCount})` : ""}
          </button>
        </div>

        <ul class="list">
          ${filtered.length === 0 ? `<div class="empty">No notifications yet</div>` : ""}
          ${filtered
            .map(
              (item, i) => `
            <li class="item ${item.is_read ? "" : "unread"}" data-index="${i}">
              ${!item.is_read ? `<span class="unread-dot"></span>` : ""}
              <div class="item-content">
                <div class="item-title">${escapeHTML(item.title)}</div>
                <div class="item-body">${escapeHTML(item.body)}</div>
                <div class="item-time">${this.formatTime(item.created_at)}</div>
              </div>
            </li>
          `
            )
            .join("")}
        </ul>
      </div>
    `;

    // Attach internal event listeners
    this.shadow.getElementById("nf-bell-btn")?.addEventListener("click", (e) => {
      e.stopPropagation();
      this.isOpen = !this.isOpen;
      this.render();
    });

    this.shadow.getElementById("nf-mark-all")?.addEventListener("click", () => this.markAllRead());

    this.shadow.getElementById("nf-tab-all")?.addEventListener("click", () => {
      this.activeTab = "all";
      this.render();
    });

    this.shadow.getElementById("nf-tab-unread")?.addEventListener("click", () => {
      this.activeTab = "unread";
      this.render();
    });

    this.shadow.querySelectorAll(".item").forEach((el) => {
      const idx = Number(el.getAttribute("data-index"));
      const item = filtered[idx];
      if (item) {
        el.addEventListener("click", () => this.handleItemClick(item));
      }
    });
  }
}

function escapeHTML(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

if (typeof customElements !== "undefined" && !customElements.get("notiflux-bell")) {
  customElements.define("notiflux-bell", NotifluxBell);
}
