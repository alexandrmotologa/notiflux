import React, { useEffect, useRef } from "react";
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

export const NotificationBell: React.FC<NotificationBellProps> = ({
  apiUrl,
  subscriberId,
  theme = "dark",
  className,
  onNotificationClick,
  onNotificationReceived,
}) => {
  const ref = useRef<HTMLElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const handleClick = (e: Event) => {
      const customEvent = e as CustomEvent<NotificationItem>;
      if (onNotificationClick && customEvent.detail) {
        onNotificationClick(customEvent.detail);
      }
    };

    const handleReceived = (e: Event) => {
      const customEvent = e as CustomEvent<NotificationItem>;
      if (onNotificationReceived && customEvent.detail) {
        onNotificationReceived(customEvent.detail);
      }
    };

    el.addEventListener("notification-click", handleClick);
    el.addEventListener("notification-received", handleReceived);

    return () => {
      el.removeEventListener("notification-click", handleClick);
      el.removeEventListener("notification-received", handleReceived);
    };
  }, [onNotificationClick, onNotificationReceived]);

  return React.createElement("notiflux-bell", {
    ref,
    "api-url": apiUrl,
    "subscriber-id": subscriberId,
    theme,
    class: className,
  });
};
