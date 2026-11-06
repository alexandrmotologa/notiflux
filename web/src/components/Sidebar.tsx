import React from "react";
import {
  LayoutDashboard,
  GitBranch,
  FileCode,
  ScrollText,
  Users,
  Settings,
  Activity,
  Bell,
  X,
} from "lucide-react";

interface SidebarProps {
  activeTab: string;
  setActiveTab: (tab: string) => void;
  memoryMB?: number;
  isOpenMobile?: boolean;
  onCloseMobile?: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  activeTab,
  setActiveTab,
  memoryMB,
  isOpenMobile = false,
  onCloseMobile,
}) => {
  const navItems = [
    { id: "dashboard", label: "Dashboard", icon: LayoutDashboard },
    { id: "workflows", label: "Workflows", icon: GitBranch },
    { id: "templates", label: "Templates", icon: FileCode },
    { id: "logs", label: "Delivery Logs", icon: ScrollText },
    { id: "subscribers", label: "Subscribers", icon: Users },
    { id: "settings", label: "Channels & VAPID", icon: Settings },
    { id: "system", label: "System Health", icon: Activity },
  ];

  const handleSelectTab = (id: string) => {
    setActiveTab(id);
    if (onCloseMobile) {
      onCloseMobile();
    }
  };

  const sidebarContent = (
    <div className="flex flex-col h-full justify-between">
      <div>
        {/* Brand header */}
        <div className="h-16 flex items-center justify-between px-6 border-b border-[#243247]">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-lg bg-indigo-600/20 border border-indigo-500/40 flex items-center justify-center text-indigo-400">
              <Bell className="w-5 h-5" />
            </div>
            <div>
              <span className="font-bold text-lg text-white tracking-tight">Notiflux</span>
              <span className="text-[10px] uppercase font-semibold text-indigo-400 bg-indigo-500/10 px-1.5 py-0.5 rounded border border-indigo-500/20 ml-2">
                Studio
              </span>
            </div>
          </div>
          {onCloseMobile && (
            <button
              onClick={onCloseMobile}
              className="md:hidden text-gray-400 hover:text-white p-1 rounded-lg"
              aria-label="Close navigation"
            >
              <X className="w-5 h-5" />
            </button>
          )}
        </div>

        {/* Navigation list */}
        <nav className="p-4 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = activeTab === item.id;
            return (
              <button
                key={item.id}
                onClick={() => handleSelectTab(item.id)}
                className={`w-full flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? "bg-indigo-600/15 text-indigo-400 border border-indigo-500/30"
                    : "text-gray-400 hover:text-gray-200 hover:bg-[#182234]"
                }`}
              >
                <Icon className={`w-4 h-4 ${isActive ? "text-indigo-400" : "text-gray-400"}`} />
                {item.label}
              </button>
            );
          })}
        </nav>
      </div>

      {/* Memory Footprint Badge */}
      <div className="p-4 border-t border-[#243247]">
        <div className="bg-[#182234] border border-[#243247] rounded-lg p-3 text-xs">
          <div className="flex items-center justify-between text-gray-400 mb-1.5">
            <span className="font-medium">RAM Utilization</span>
            <span className="text-emerald-400 font-mono font-semibold">
              {memoryMB !== undefined ? `${memoryMB} MB` : "< 45 MB"}
            </span>
          </div>
          <div className="w-full bg-[#0b0f17] h-1.5 rounded-full overflow-hidden">
            <div
              className="bg-emerald-500 h-full rounded-full transition-all duration-500"
              style={{ width: `${Math.min(100, ((memoryMB || 15) / 45) * 100)}%` }}
            />
          </div>
          <div className="text-[10px] text-gray-400 mt-1.5 flex justify-between">
            <span>Target: &lt; 45MB</span>
            <span>Single Binary</span>
          </div>
        </div>
      </div>
    </div>
  );

  return (
    <>
      {/* Desktop Persistent Sidebar */}
      <aside className="hidden md:flex w-64 bg-[#111827] border-r border-[#243247] flex-col shrink-0 select-none">
        {sidebarContent}
      </aside>

      {/* Mobile Slide-over Drawer */}
      {isOpenMobile && (
        <div className="fixed inset-0 z-50 md:hidden">
          <div
            className="fixed inset-0 bg-black/60 backdrop-blur-sm transition-opacity"
            onClick={onCloseMobile}
          />
          <aside className="fixed inset-y-0 left-0 w-64 bg-[#111827] border-r border-[#243247] flex flex-col z-50 select-none shadow-2xl animate-in slide-in-from-left duration-200">
            {sidebarContent}
          </aside>
        </div>
      )}
    </>
  );
};
