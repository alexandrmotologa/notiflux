export interface NotificationItem {
    id: string;
    workflow_id: string;
    title: string;
    body: string;
    action_url?: string;
    is_read: boolean;
    created_at: string;
}
export declare class NotifluxBell extends HTMLElement {
    private shadow;
    private unreadCount;
    private notifications;
    private activeTab;
    private isOpen;
    private eventSource;
    private reconnectTimeout;
    constructor();
    static get observedAttributes(): string[];
    attributeChangedCallback(name: string, oldVal: string, newVal: string): void;
    connectedCallback(): void;
    disconnectedCallback(): void;
    private handleDocumentClick;
    private get apiUrl();
    private get subscriberId();
    private playChime;
    private fetchFeed;
    private initConnection;
    private markAsRead;
    private markAllRead;
    private handleItemClick;
    private requestDesktopPush;
    private formatTime;
    private render;
}
