import React from "react";
import "./vanilla";
import type { NotificationItem } from "./vanilla";
export interface NotificationBellProps {
    apiUrl: string;
    subscriberId: string;
    theme?: "dark" | "light";
    className?: string;
    onNotificationClick?: (notification: NotificationItem) => void;
    onNotificationReceived?: (notification: NotificationItem) => void;
}
export declare const NotificationBell: React.FC<NotificationBellProps>;
