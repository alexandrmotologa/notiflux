(function(r,o){typeof exports=="object"&&typeof module<"u"?o(exports,require("react")):typeof define=="function"&&define.amd?define(["exports","react"],o):(r=typeof globalThis<"u"?globalThis:r||self,o(r.NotifluxBell={},r.React))})(this,(function(r,o){"use strict";var y=Object.defineProperty;var k=(r,o,u)=>o in r?y(r,o,{enumerable:!0,configurable:!0,writable:!0,value:u}):r[o]=u;var d=(r,o,u)=>k(r,typeof o!="symbol"?o+"":o,u);const u=`
:host {
  display: inline-block;
  position: relative;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  box-sizing: border-box;
  --nf-accent-color: var(--nf-accent, #6366f1);
  --nf-border-color: var(--nf-border, #374151);
  --nf-bg-color: var(--nf-bg, #1f2937);
  --nf-panel-bg-color: var(--nf-panel-bg, #111827);
  --nf-text-color: var(--nf-text, #f3f4f6);
  --nf-radius-size: var(--nf-radius, 12px);
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
  border: 1px solid var(--nf-border-color);
  background: var(--nf-bg-color);
  color: var(--nf-text-color);
  cursor: pointer;
  outline: none;
  transition: all 0.2s ease;
}

.bell-btn:hover {
  background: var(--nf-hover, #374151);
  border-color: var(--nf-accent-color);
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
  border: 2px solid var(--nf-bg-color);
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
  background: var(--nf-panel-bg-color);
  border: 1px solid var(--nf-border-color);
  border-radius: var(--nf-radius-size);
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
  border-bottom: 1px solid var(--nf-border-color);
}

.header-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--nf-text-color);
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.header-actions button {
  background: none;
  border: none;
  color: var(--nf-accent-color);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 4px;
}

.header-actions button:hover {
  background: rgba(99, 102, 241, 0.1);
}

.push-btn {
  color: #10b981 !important;
  font-size: 11px !important;
  border: 1px solid rgba(16, 185, 129, 0.3) !important;
  padding: 2px 6px !important;
  border-radius: 4px !important;
}

.push-btn:hover {
  background: rgba(16, 185, 129, 0.1) !important;
}

.tabs {
  display: flex;
  padding: 4px 16px;
  gap: 12px;
  border-bottom: 1px solid var(--nf-border-color);
  background: var(--nf-panel-bg-color);
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
  color: var(--nf-accent-color);
  border-bottom-color: var(--nf-accent-color);
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
  background: var(--nf-accent-color);
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
  color: var(--nf-text-color);
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
`;class b extends HTMLElement{constructor(){super();d(this,"shadow");d(this,"unreadCount",0);d(this,"notifications",[]);d(this,"activeTab","all");d(this,"isOpen",!1);d(this,"eventSource",null);d(this,"reconnectTimeout",1e3);d(this,"handleDocumentClick",t=>{!this.contains(t.target)&&this.isOpen&&(this.isOpen=!1,this.render())});this.shadow=this.attachShadow({mode:"open"})}static get observedAttributes(){return["api-url","subscriber-id","theme","sound"]}attributeChangedCallback(t,e,i){e!==i&&((t==="subscriber-id"||t==="api-url")&&(this.initConnection(),this.fetchFeed()),this.render())}connectedCallback(){this.render(),this.initConnection(),this.fetchFeed(),document.addEventListener("click",this.handleDocumentClick)}disconnectedCallback(){this.eventSource&&(this.eventSource.close(),this.eventSource=null),document.removeEventListener("click",this.handleDocumentClick)}get apiUrl(){return this.getAttribute("api-url")||""}get subscriberId(){return this.getAttribute("subscriber-id")||""}playChime(){if(this.getAttribute("sound")!=="false")try{const t=window.AudioContext||window.webkitAudioContext;if(!t)return;const e=new t,i=e.createOscillator(),a=e.createGain();i.type="sine",i.frequency.setValueAtTime(587.33,e.currentTime),i.frequency.exponentialRampToValueAtTime(880,e.currentTime+.12),a.gain.setValueAtTime(.12,e.currentTime),a.gain.exponentialRampToValueAtTime(.001,e.currentTime+.38),i.connect(a),a.connect(e.destination),i.start(),i.stop(e.currentTime+.4)}catch{}}async fetchFeed(){if(this.subscriberId)try{const t=await fetch(`${this.apiUrl}/v1/feed?subscriber_id=${encodeURIComponent(this.subscriberId)}&limit=30`);if(t.ok){const e=await t.json();this.notifications=e.notifications||[],this.unreadCount=e.unread_count||0,this.render()}}catch{}}initConnection(){if(this.eventSource&&(this.eventSource.close(),this.eventSource=null),!this.subscriberId)return;const t=`${this.apiUrl}/v1/feed/stream?subscriber_id=${encodeURIComponent(this.subscriberId)}`;this.eventSource=new EventSource(t),this.eventSource.addEventListener("connected",e=>{try{const i=JSON.parse(e.data);typeof i.unread_count=="number"&&(this.unreadCount=i.unread_count,this.render())}catch{}this.reconnectTimeout=1e3}),this.eventSource.addEventListener("notification.new",e=>{try{const i=JSON.parse(e.data);i.notification&&(this.notifications.unshift(i.notification),this.playChime(),typeof Notification<"u"&&Notification.permission==="granted"&&document.hidden&&new Notification(i.notification.title,{body:i.notification.body})),typeof i.unread_count=="number"?this.unreadCount=i.unread_count:this.unreadCount++,this.render(),this.dispatchEvent(new CustomEvent("notification-received",{detail:i.notification}))}catch{}}),this.eventSource.addEventListener("notification.read",e=>{try{const i=JSON.parse(e.data),a=this.notifications.find(l=>l.id===i.id);a&&(a.is_read=!0),typeof i.unread_count=="number"&&(this.unreadCount=i.unread_count),this.render()}catch{}}),this.eventSource.addEventListener("notification.read_all",()=>{this.notifications.forEach(e=>e.is_read=!0),this.unreadCount=0,this.render()}),this.eventSource.onerror=()=>{this.eventSource&&(this.eventSource.close(),this.eventSource=null),setTimeout(()=>this.initConnection(),Math.min(this.reconnectTimeout*=2,16e3))}}async markAsRead(t){if(!t.is_read){t.is_read=!0,this.unreadCount=Math.max(0,this.unreadCount-1),this.render();try{await fetch(`${this.apiUrl}/v1/feed/${t.id}/read`,{method:"PATCH"})}catch{}}}async markAllRead(){this.notifications.forEach(t=>t.is_read=!0),this.unreadCount=0,this.render();try{await fetch(`${this.apiUrl}/v1/feed/read-all`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({subscriber_id:this.subscriberId})})}catch{}}handleItemClick(t){this.markAsRead(t),this.dispatchEvent(new CustomEvent("notification-click",{detail:t})),t.action_url&&window.open(t.action_url,"_blank")}requestDesktopPush(){typeof Notification<"u"&&Notification.requestPermission().then(()=>{this.render()})}formatTime(t){const e=Math.floor((Date.now()-new Date(t).getTime())/1e3);return e<60?"just now":e<3600?`${Math.floor(e/60)}m ago`:e<86400?`${Math.floor(e/3600)}h ago`:`${Math.floor(e/86400)}d ago`}render(){var a,l,c,f,h;const t=this.activeTab==="unread"?this.notifications.filter(n=>!n.is_read):this.notifications,e=this.unreadCount>99?"99+":String(this.unreadCount),i=typeof Notification<"u"&&Notification.permission==="default";this.shadow.innerHTML=`
      <style>${u}</style>
      <button class="bell-btn" aria-label="Notifications" id="nf-bell-btn">
        <svg class="bell-icon" viewBox="0 0 24 24">
          <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path>
          <path d="M13.73 21a2 2 0 0 1-3.46 0"></path>
        </svg>
        <span class="badge" ${this.unreadCount===0?"hidden":""}>${e}</span>
      </button>

      <div class="popover" ${this.isOpen?"":"hidden"} id="nf-popover">
        <div class="header">
          <div class="header-title">
            <span>Notifications</span>
          </div>
          <div class="header-actions">
            ${i?'<button id="nf-prompt-push" class="push-btn">Enable alerts</button>':""}
            ${this.unreadCount>0?'<button id="nf-mark-all">Mark all read</button>':""}
          </div>
        </div>

        <div class="tabs">
          <button class="tab ${this.activeTab==="all"?"active":""}" id="nf-tab-all">All</button>
          <button class="tab ${this.activeTab==="unread"?"active":""}" id="nf-tab-unread">
            Unread ${this.unreadCount>0?`(${this.unreadCount})`:""}
          </button>
        </div>

        <ul class="list">
          ${t.length===0?'<div class="empty">No notifications yet</div>':""}
          ${t.map((n,s)=>`
            <li class="item ${n.is_read?"":"unread"}" data-index="${s}">
              ${n.is_read?"":'<span class="unread-dot"></span>'}
              <div class="item-content">
                <div class="item-title">${m(n.title)}</div>
                <div class="item-body">${m(n.body)}</div>
                <div class="item-time">${this.formatTime(n.created_at)}</div>
              </div>
            </li>
          `).join("")}
        </ul>
      </div>
    `,(a=this.shadow.getElementById("nf-bell-btn"))==null||a.addEventListener("click",n=>{n.stopPropagation(),this.isOpen=!this.isOpen,this.render()}),(l=this.shadow.getElementById("nf-mark-all"))==null||l.addEventListener("click",()=>this.markAllRead()),(c=this.shadow.getElementById("nf-prompt-push"))==null||c.addEventListener("click",()=>this.requestDesktopPush()),(f=this.shadow.getElementById("nf-tab-all"))==null||f.addEventListener("click",()=>{this.activeTab="all",this.render()}),(h=this.shadow.getElementById("nf-tab-unread"))==null||h.addEventListener("click",()=>{this.activeTab="unread",this.render()}),this.shadow.querySelectorAll(".item").forEach(n=>{const s=Number(n.getAttribute("data-index")),g=t[s];g&&n.addEventListener("click",()=>this.handleItemClick(g))})}}function m(p){return p.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#039;")}typeof customElements<"u"&&!customElements.get("notiflux-bell")&&customElements.define("notiflux-bell",b);const x=({apiUrl:p,subscriberId:v,theme:t="dark",className:e,onNotificationClick:i,onNotificationReceived:a})=>{const l=o.useRef(null);return o.useEffect(()=>{const c=l.current;if(!c)return;const f=n=>{const s=n;i&&s.detail&&i(s.detail)},h=n=>{const s=n;a&&s.detail&&a(s.detail)};return c.addEventListener("notification-click",f),c.addEventListener("notification-received",h),()=>{c.removeEventListener("notification-click",f),c.removeEventListener("notification-received",h)}},[i,a]),o.createElement("notiflux-bell",{ref:l,"api-url":p,"subscriber-id":v,theme:t,class:e})};r.NotificationBell=x,r.NotifluxBell=b,Object.defineProperty(r,Symbol.toStringTag,{value:"Module"})}));
