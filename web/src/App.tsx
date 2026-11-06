import React, { useState, useEffect } from "react";
import { Menu } from "lucide-react";
import { Sidebar } from "./components/Sidebar";
import { Dashboard } from "./components/Dashboard";
import { WorkflowsView } from "./components/WorkflowsView";
import { TemplatesView } from "./components/TemplatesView";
import { DeliveryLogsView } from "./components/DeliveryLogsView";
import { SubscribersView } from "./components/SubscribersView";
import { SettingsView } from "./components/SettingsView";
import { SystemHealthView } from "./components/SystemHealthView";
import { fetchHealth, fetchWorkflows, fetchLogs } from "./api";
import type { SystemHealth, Workflow, DeliveryLog } from "./types";

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState("dashboard");
  const [health, setHealth] = useState<SystemHealth | null>(null);
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [logs, setLogs] = useState<DeliveryLog[]>([]);
  const [targetWorkflowForTemplates, setTargetWorkflowForTemplates] = useState<string | undefined>();
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  const loadData = async () => {
    try {
      const [h, wfs, l] = await Promise.all([
        fetchHealth(),
        fetchWorkflows(),
        fetchLogs(20),
      ]);
      setHealth(h);
      setWorkflows(wfs);
      setLogs(l.logs || []);
    } catch {
      // Offline fallback
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleEditTemplates = (workflowId: string) => {
    setTargetWorkflowForTemplates(workflowId);
    setActiveTab("templates");
  };

  return (
    <div className="flex h-screen bg-[#0b0f17] text-gray-100 overflow-hidden font-sans">
      {/* Sidebar */}
      <Sidebar
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        memoryMB={health?.memory_alloc_mb}
        isOpenMobile={isMobileMenuOpen}
        onCloseMobile={() => setIsMobileMenuOpen(false)}
      />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Top bar */}
        <header className="h-16 bg-[#111827] border-b border-[#243247] px-4 sm:px-6 flex items-center justify-between shrink-0">
          <div className="flex items-center gap-3 text-sm">
            <button
              onClick={() => setIsMobileMenuOpen(true)}
              className="md:hidden text-gray-400 hover:text-white p-1 rounded-lg"
              aria-label="Open navigation menu"
            >
              <Menu className="w-5 h-5" />
            </button>
            <span className="text-gray-400 font-medium hidden sm:inline">Notiflux Studio</span>
            <span className="text-gray-600 hidden sm:inline">/</span>
            <span className="text-white font-semibold capitalize">{activeTab}</span>
          </div>

          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2 text-xs text-gray-400 mr-2">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
              <span>Daemon Online</span>
            </div>

            {/* Embedded Live Widget Test in Studio Header */}
            <div className="flex items-center gap-2 border-l border-[#243247] pl-4">
              <span className="text-[11px] text-gray-400 font-medium hidden sm:inline">
                Live Widget (usr_demo):
              </span>
              <div className="relative">
                {/* @ts-ignore */}
                <notiflux-bell
                  api-url=""
                  subscriber-id="usr_demo"
                  theme="dark"
                />
              </div>
            </div>
          </div>
        </header>

        {/* Scrollable View Container */}
        <main className="flex-1 overflow-y-auto p-6">
          <div className="max-w-7xl mx-auto">
            {activeTab === "dashboard" && (
              <Dashboard
                health={health}
                logs={logs}
                workflows={workflows}
                onTriggerSuccess={loadData}
                onNavigate={setActiveTab}
              />
            )}

            {activeTab === "workflows" && (
              <WorkflowsView
                workflows={workflows}
                onRefresh={loadData}
                onEditTemplates={handleEditTemplates}
              />
            )}

            {activeTab === "templates" && (
              <TemplatesView
                workflows={workflows}
                initialWorkflowId={targetWorkflowForTemplates}
              />
            )}

            {activeTab === "logs" && <DeliveryLogsView />}

            {activeTab === "subscribers" && <SubscribersView />}

            {activeTab === "settings" && <SettingsView />}

            {activeTab === "system" && (
              <SystemHealthView health={health} onRefresh={loadData} />
            )}
          </div>
        </main>
      </div>
    </div>
  );
};
