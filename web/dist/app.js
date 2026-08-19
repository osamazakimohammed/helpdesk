// Digitera Helpdesk - Complete Role-Aware Frontend Application

const state = {
  user: JSON.parse(localStorage.getItem('helpdesk_user') || 'null'),
  token: localStorage.getItem('helpdesk_token') || '',
  authMode: 'login', // 'login' or 'signup'
  
  // Agent Workspace State
  tickets: [],
  selectedTicket: null,
  timelineEvents: [],
  activeFilter: 'open',
  searchQuery: '',
  composerMode: 'reply', // 'reply' or 'note'
  replyText: '',

  // Customer Portal State
  portalTickets: [],
  selectedPortalTicket: null,
  portalReplyText: '',

  // Knowledge Base State
  kbSpaces: [],
  selectedSpace: null,
  kbSearchQuery: '',
};

// API Helper with automatic auth header and error handling
async function api(path, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...(state.token ? { 'Authorization': `Bearer ${state.token}` } : {}),
    ...options.headers,
  };
  const res = await fetch(`/api/v1${path}`, { ...options, headers });
  if (res.status === 401 && !path.includes('/auth/')) {
    handleLogout();
    throw new Error('Unauthorized');
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error(err.message || 'API request failed');
  }
  return res.json();
}

function navigate(url) {
  history.pushState(null, '', url);
  render();
}

window.addEventListener('popstate', render);

// Format Helpers
function formatRelativeTime(dateStr) {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  const diffMs = Date.now() - d.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return 'just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDays = Math.floor(diffHr / 24);
  return `${diffDays}d ago`;
}



function getPriorityDot(priorityKey) {
  switch (priorityKey) {
    case 'urgent': return '<span class="w-2.5 h-2.5 rounded-full bg-rose-500 inline-block shadow-sm shadow-rose-500/50" title="Urgent"></span>';
    case 'high': return '<span class="w-2.5 h-2.5 rounded-full bg-amber-500 inline-block shadow-sm shadow-amber-500/50" title="High"></span>';
    case 'medium': return '<span class="w-2.5 h-2.5 rounded-full bg-blue-500 inline-block shadow-sm shadow-blue-500/50" title="Medium"></span>';
    default: return '<span class="w-2.5 h-2.5 rounded-full bg-slate-500 inline-block" title="Low"></span>';
  }
}

function getStatusBadge(statusKey, statusLabel) {
  switch (statusKey) {
    case 'new': return `<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-indigo-500/15 text-indigo-400 border border-indigo-500/30">${statusLabel || 'New'}</span>`;
    case 'open': return `<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">${statusLabel || 'Open'}</span>`;
    case 'pending': return `<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-500/15 text-amber-400 border border-amber-500/30">${statusLabel || 'Pending'}</span>`;
    case 'on_hold': return `<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-purple-500/15 text-purple-400 border border-purple-500/30">${statusLabel || 'On Hold'}</span>`;
    case 'resolved': return `<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-cyan-500/15 text-cyan-400 border border-cyan-500/30">${statusLabel || 'Resolved'}</span>`;
    case 'closed': return `<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-slate-800 text-slate-400 border border-slate-700">${statusLabel || 'Closed'}</span>`;
    default: return `<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-slate-800 text-slate-300">${statusLabel || statusKey}</span>`;
  }
}

// ----------------- Dynamic Role-Aware Top Navigation -----------------
function renderNavbar() {
  const path = window.location.pathname;
  const user = state.user;
  const role = user ? user.role : 'guest';

  // Strict Role-Based Nav Items
  let navLinks = '';

  if (role === 'admin' || role === 'manager') {
    navLinks = `
      <a href="/app" onclick="event.preventDefault(); navigate('/app')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/app') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">Agent Workspace</a>
      <a href="/portal" onclick="event.preventDefault(); navigate('/portal')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/portal') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">Customer Portal</a>
      <a href="/submit" onclick="event.preventDefault(); navigate('/submit')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/submit') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">Submit Request</a>
      <a href="/kb" onclick="event.preventDefault(); navigate('/kb')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/kb') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">Help Center</a>
      <a href="/api/docs" target="_blank" class="px-3.5 py-1.5 rounded-lg text-xs font-bold text-slate-400 hover:text-white hover:bg-slate-800/60 transition flex items-center gap-1.5">OpenAPI ↗</a>
    `;
  } else if (role === 'agent' || role === 'restricted_agent') {
    navLinks = `
      <a href="/app" onclick="event.preventDefault(); navigate('/app')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/app') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">Agent Workspace</a>
      <a href="/kb" onclick="event.preventDefault(); navigate('/kb')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/kb') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">Help Center</a>
    `;
  } else if (role === 'contact' || role === 'customer') {
    navLinks = `
      <a href="/portal" onclick="event.preventDefault(); navigate('/portal')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/portal') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">My Support Tickets</a>
      <a href="/submit" onclick="event.preventDefault(); navigate('/submit')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/submit') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">Submit New Request</a>
      <a href="/kb" onclick="event.preventDefault(); navigate('/kb')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/kb') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">Knowledge Base</a>
    `;
  } else {
    // Anonymous Visitor Navigation
    navLinks = `
      <a href="/kb" onclick="event.preventDefault(); navigate('/kb')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/kb') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">Knowledge Base</a>
      <a href="/submit" onclick="event.preventDefault(); navigate('/submit')" class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition ${path.startsWith('/submit') ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'}">Submit Ticket</a>
    `;
  }

  return `
    <header class="bg-slate-900/95 backdrop-blur border-b border-slate-800 sticky top-0 z-50 px-6 py-3 flex items-center justify-between shadow-xl">
      <div class="flex items-center gap-8">
        <a href="/" onclick="event.preventDefault(); navigate(getDefaultRoute())" class="flex items-center gap-3 group">
          <div class="w-8 h-8 rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center font-extrabold text-white shadow-md shadow-indigo-500/25 group-hover:scale-105 transition-transform">
            ⚡
          </div>
          <span class="font-extrabold text-base tracking-tight text-white">Digitera Helpdesk</span>
        </a>
        <nav class="hidden md:flex items-center gap-1.5">
          ${navLinks}
        </nav>
      </div>

      <div class="flex items-center gap-3">
        ${user ? `
          <div class="flex items-center gap-3 bg-slate-800/90 px-3.5 py-1.5 rounded-xl border border-slate-700/80 shadow-sm">
            <div class="w-7 h-7 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center font-bold text-xs text-white shadow-inner">
              ${(user.full_name || user.email)[0].toUpperCase()}
            </div>
            <div class="text-xs">
              <div class="font-bold text-slate-200 leading-tight">${user.full_name || user.email}</div>
              <div class="text-[10px] font-mono uppercase font-bold tracking-wider ${user.role === 'admin' ? 'text-indigo-400' : user.role === 'agent' ? 'text-emerald-400' : 'text-cyan-400'}">${user.role}</div>
            </div>
            <button onclick="handleLogout()" class="text-xs text-rose-400 hover:text-rose-300 ml-2 font-bold px-2 py-1 rounded hover:bg-rose-500/10 transition">
              Sign Out
            </button>
          </div>
        ` : `
          <div class="flex items-center gap-2">
            <button onclick="navigate('/login')" class="bg-indigo-600 hover:bg-indigo-500 text-white font-bold text-xs px-4 py-2 rounded-xl shadow-md shadow-indigo-600/30 transition">
              Sign In / Register
            </button>
          </div>
        `}
      </div>
    </header>
  `;
}

function getDefaultRoute() {
  if (!state.user) return '/login';
  if (state.user.role === 'admin' || state.user.role === 'agent') return '/app';
  return '/portal';
}

// ----------------- Auth Gateway: Sign In & Sign Up (/login) -----------------
function renderAuthGateway() {
  return `
    <div class="flex-1 flex flex-col items-center justify-center p-6 bg-slate-950">
      <div class="w-full max-w-md bg-slate-900/90 backdrop-blur-xl p-8 rounded-3xl border border-slate-800 shadow-2xl shadow-black/50 space-y-6">
        <div class="text-center space-y-2">
          <div class="w-12 h-12 rounded-2xl bg-gradient-to-tr from-indigo-600 to-violet-500 mx-auto flex items-center justify-center font-black text-xl text-white shadow-lg shadow-indigo-500/30">
            ⚡
          </div>
          <h1 class="text-2xl font-black text-white tracking-tight">Welcome to Helpdesk</h1>
          <p class="text-xs text-slate-400">Enterprise support platform. Sign in to access your portal.</p>
        </div>

        <!-- Auth Mode Tabs -->
        <div class="flex p-1 bg-slate-950 rounded-xl border border-slate-800">
          <button 
            onclick="setAuthMode('login')" 
            class="flex-1 py-2 text-xs font-bold rounded-lg transition ${state.authMode === 'login' ? 'bg-indigo-600 text-white shadow-sm' : 'text-slate-400 hover:text-white'}"
          >
            Sign In
          </button>
          <button 
            onclick="setAuthMode('signup')" 
            class="flex-1 py-2 text-xs font-bold rounded-lg transition ${state.authMode === 'signup' ? 'bg-indigo-600 text-white shadow-sm' : 'text-slate-400 hover:text-white'}"
          >
            Create Account
          </button>
        </div>

        ${state.authMode === 'login' ? `
          <!-- Sign In Form -->
          <form onsubmit="handleLoginSubmit(event)" class="space-y-4">
            <div>
              <label class="block text-xs font-bold text-slate-300 mb-1.5">Username or Email</label>
              <input type="text" name="email" id="login-email" required placeholder="admin / support / customer" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition" />
            </div>
            <div>
              <label class="block text-xs font-bold text-slate-300 mb-1.5">Password</label>
              <input type="password" name="password" id="login-password" required placeholder="admin / support / customer" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition" />
            </div>
            <button type="submit" class="w-full bg-indigo-600 hover:bg-indigo-500 text-white font-extrabold text-xs py-3 rounded-xl shadow-lg shadow-indigo-600/30 transition">
              Sign In to Workspace
            </button>
          </form>
        ` : `
          <!-- Sign Up Form -->
          <form onsubmit="handleRegisterSubmit(event)" class="space-y-4">
            <div>
              <label class="block text-xs font-bold text-slate-300 mb-1.5">Full Name</label>
              <input type="text" name="full_name" required placeholder="Jane Doe" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition" />
            </div>
            <div>
              <label class="block text-xs font-bold text-slate-300 mb-1.5">Email Address</label>
              <input type="email" name="email" required placeholder="jane@company.com" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition" />
            </div>
            <div>
              <label class="block text-xs font-bold text-slate-300 mb-1.5">Password</label>
              <input type="password" name="password" required placeholder="Choose a strong password" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition" />
            </div>
            <div>
              <label class="block text-xs font-bold text-slate-300 mb-1.5">Account Role</label>
              <select name="role" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-indigo-500">
                <option value="contact">Customer / Client (Self-Service Portal)</option>
                <option value="agent">Support Agent (Agent Workspace)</option>
              </select>
            </div>
            <button type="submit" class="w-full bg-indigo-600 hover:bg-indigo-500 text-white font-extrabold text-xs py-3 rounded-xl shadow-lg shadow-indigo-600/30 transition">
              Create Account & Sign In
            </button>
          </form>
        `}

        <div class="text-center pt-2">
          <a href="/kb" onclick="event.preventDefault(); navigate('/kb')" class="text-xs font-medium text-slate-400 hover:text-slate-300 transition">
            Continue as Guest to Knowledge Base →
          </a>
        </div>
      </div>
    </div>
  `;
}

function setAuthMode(mode) {
  state.authMode = mode;
  render();
}

async function handleLoginSubmit(e) {
  e.preventDefault();
  const form = e.target;
  try {
    const res = await api('/auth/login', {
      method: 'POST',
      body: JSON.stringify({
        email: form.email.value,
        password: form.password.value,
      }),
    });
    state.token = res.token;
    state.user = res.user;
    localStorage.setItem('helpdesk_token', res.token);
    localStorage.setItem('helpdesk_user', JSON.stringify(res.user));

    if (res.user.role === 'admin' || res.user.role === 'agent') {
      navigate('/app');
    } else {
      navigate('/portal');
    }
  } catch (err) {
    alert('Sign in failed: ' + err.message);
  }
}

async function handleRegisterSubmit(e) {
  e.preventDefault();
  const form = e.target;
  try {
    const res = await api('/auth/register', {
      method: 'POST',
      body: JSON.stringify({
        full_name: form.full_name.value,
        email: form.email.value,
        password: form.password.value,
        role: form.role.value,
      }),
    });
    state.token = res.token;
    state.user = res.user;
    localStorage.setItem('helpdesk_token', res.token);
    localStorage.setItem('helpdesk_user', JSON.stringify(res.user));

    if (res.user.role === 'admin' || res.user.role === 'agent') {
      navigate('/app');
    } else {
      navigate('/portal');
    }
  } catch (err) {
    alert('Registration failed: ' + err.message);
  }
}

function handleLogout() {
  state.token = '';
  state.user = null;
  localStorage.removeItem('helpdesk_token');
  localStorage.removeItem('helpdesk_user');
  navigate('/login');
}

// ----------------- Surface 1: Agent Workspace (/app) -----------------
async function loadAgentData() {
  if (!state.token || !state.user || (state.user.role !== 'admin' && state.user.role !== 'agent')) {
    return;
  }
  try {
    const params = new URLSearchParams();
    if (state.activeFilter && state.activeFilter !== 'all') {
      params.set('status_category', state.activeFilter);
    }
    if (state.searchQuery) {
      params.set('search', state.searchQuery);
    }

    const data = await api(`/app/tickets?${params.toString()}`);
    state.tickets = data.items || [];

    if (state.tickets.length > 0) {
      const currentId = state.selectedTicket ? state.selectedTicket.id : null;
      const target = currentId ? state.tickets.find(t => t.id === currentId) : state.tickets[0];
      await selectTicket(target ? target.id : state.tickets[0].id);
    } else {
      state.selectedTicket = null;
      render();
    }
  } catch (err) {
    console.error('Failed to load tickets:', err);
    render();
  }
}

async function selectTicket(ticketID) {
  try {
    const [detail, eventsData] = await Promise.all([
      api(`/app/tickets/${ticketID}`),
      api(`/app/tickets/${ticketID}/events`),
    ]);
    state.selectedTicket = detail;
    state.timelineEvents = eventsData.items || [];
    render();
  } catch (err) {
    console.error('Failed to load ticket detail:', err);
  }
}

async function handleUpdateTicketField(field, value) {
  if (!state.selectedTicket) return;
  try {
    const payload = {};
    payload[field] = value;
    await api(`/app/tickets/${state.selectedTicket.id}`, {
      method: 'PATCH',
      body: JSON.stringify(payload),
    });
    await loadAgentData();
    if (state.selectedTicket) {
      await selectTicket(state.selectedTicket.id);
    }
  } catch (err) {
    alert('Update failed: ' + err.message);
  }
}

async function handleSendReply() {
  if (!state.selectedTicket) {
    alert('Please select a ticket first.');
    return;
  }

  const input = document.getElementById('composer-input');
  const text = (input ? input.value : (state.replyText || '')).trim();
  if (!text) {
    alert('Please enter a message before sending.');
    if (input) input.focus();
    return;
  }

  const isNote = state.composerMode === 'note';
  const submitBtn = document.getElementById('composer-submit-btn');
  if (submitBtn) {
    submitBtn.disabled = true;
    submitBtn.innerText = 'Sending...';
  }

  try {
    await api(`/app/tickets/${state.selectedTicket.id}/events`, {
      method: 'POST',
      body: JSON.stringify({
        kind: isNote ? 'internal_note' : 'outbound_email',
        visibility: isNote ? 'internal' : 'public',
        body_text: text,
      }),
    });
    if (input) input.value = '';
    state.replyText = '';
    await selectTicket(state.selectedTicket.id);
    await loadAgentData();
  } catch (err) {
    alert('Failed to post reply: ' + err.message);
  } finally {
    if (submitBtn) {
      submitBtn.disabled = false;
      submitBtn.innerText = isNote ? 'Save Internal Note' : 'Send Public Reply';
    }
  }
}

function renderAgentApp() {
  if (!state.user || (state.user.role !== 'admin' && state.user.role !== 'agent')) {
    return renderAuthGateway();
  }

  const selected = state.selectedTicket;

  return `
    <div class="flex-1 flex overflow-hidden">
      <!-- 1. Left Sidebar Navigation -->
      <aside class="w-64 bg-slate-900 border-r border-slate-800 flex flex-col justify-between p-4 shrink-0 shadow-lg">
        <div class="space-y-6">
          <div>
            <div class="text-[11px] font-extrabold uppercase tracking-wider text-slate-400 mb-2.5 px-2.5">Ticket Queues</div>
            <nav class="space-y-1">
              <button onclick="setFilter('open')" class="w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-xs font-bold transition ${state.activeFilter === 'open' ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-300 hover:bg-slate-800'}">
                <span class="flex items-center gap-2.5">📥 All Open</span>
                <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 font-bold">${state.tickets.filter(t => t.status_category === 'open').length}</span>
              </button>
              <button onclick="setFilter('pending')" class="w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-xs font-bold transition ${state.activeFilter === 'pending' ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-300 hover:bg-slate-800'}">
                <span class="flex items-center gap-2.5">⏳ Pending Response</span>
              </button>
              <button onclick="setFilter('resolved')" class="w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-xs font-bold transition ${state.activeFilter === 'resolved' ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-300 hover:bg-slate-800'}">
                <span class="flex items-center gap-2.5">✅ Resolved / Closed</span>
              </button>
              <button onclick="setFilter('all')" class="w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-xs font-bold transition ${state.activeFilter === 'all' ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40 shadow-sm' : 'text-slate-300 hover:bg-slate-800'}">
                <span class="flex items-center gap-2.5">📁 All Tickets</span>
              </button>
            </nav>
          </div>

          <div>
            <div class="text-[11px] font-extrabold uppercase tracking-wider text-slate-400 mb-2.5 px-2.5">Resources</div>
            <nav class="space-y-1">
              <button onclick="navigate('/kb')" class="w-full flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-xs font-bold text-slate-300 hover:bg-slate-800 hover:text-white transition">
                📚 Help Center Guides
              </button>
            </nav>
          </div>
        </div>

        <div class="p-3.5 rounded-2xl bg-slate-800/80 border border-slate-700/70 text-xs">
          <div class="flex items-center justify-between mb-1">
            <span class="text-slate-400 text-[11px] font-semibold">Agent Availability</span>
            <span class="flex items-center gap-1.5 text-emerald-400 font-bold text-[11px]"><span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span> Online</span>
          </div>
          <div class="text-[10px] text-slate-500 font-mono">1 Query Keyset List</div>
        </div>
      </aside>

      <!-- 2. Middle Ticket List Column -->
      <section class="w-96 bg-slate-900/40 border-r border-slate-800 flex flex-col shrink-0">
        <div class="p-4 border-b border-slate-800 space-y-3">
          <div class="flex items-center justify-between">
            <h2 class="font-black text-sm text-white capitalize flex items-center gap-2">
              ${state.activeFilter} Queue
              <span class="text-xs font-mono px-2 py-0.5 bg-slate-800 text-slate-300 rounded-full border border-slate-700 font-bold">${state.tickets.length}</span>
            </h2>
            <button onclick="loadAgentData()" class="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition text-xs font-bold flex items-center gap-1">
              🔄 Refresh
            </button>
          </div>
          <div>
            <input 
              type="text" 
              placeholder="Filter by subject, contact, ref..."
              value="${state.searchQuery}"
              oninput="handleSearchInput(event)"
              class="w-full bg-slate-800/90 border border-slate-700 rounded-xl px-3.5 py-2 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition"
            />
          </div>
        </div>

        <!-- Ticket Card Stream -->
        <div class="flex-1 overflow-y-auto divide-y divide-slate-800/60">
          ${state.tickets.length === 0 ? `
            <div class="p-10 text-center text-slate-500 text-xs">
              <div class="text-3xl mb-2">🎉</div>
              <p class="font-bold text-slate-400">Queue is clear</p>
              <p class="text-[11px] mt-1 text-slate-500">No active tickets matching this filter.</p>
            </div>
          ` : state.tickets.map(t => `
            <div 
              onclick="selectTicket('${t.id}')"
              class="p-4 cursor-pointer transition relative ${selected && selected.id === t.id ? 'bg-indigo-950/40 border-l-4 border-indigo-500' : 'hover:bg-slate-800/40'}"
            >
              <div class="flex items-center justify-between gap-2 mb-1.5">
                <div class="flex items-center gap-2">
                  ${getPriorityDot(t.priority_key)}
                  <span class="font-mono text-xs font-extrabold text-slate-300">${t.reference_no}</span>
                </div>
                <span class="text-[10px] text-slate-500 font-mono">${formatRelativeTime(t.updated_at)}</span>
              </div>
              <h3 dir="auto" class="font-bold text-xs text-slate-100 line-clamp-1 mb-1">${t.subject}</h3>
              <p dir="auto" class="text-[11px] text-slate-400 line-clamp-2 mb-2.5 leading-relaxed">${t.description_text || ''}</p>
              <div class="flex items-center justify-between text-[10px] text-slate-400">
                <div class="flex items-center gap-1.5 font-semibold">
                  <div class="w-4 h-4 rounded-full bg-slate-700 flex items-center justify-center font-bold text-[9px] text-slate-200">
                    ${(t.contact_name || t.contact_email)[0].toUpperCase()}
                  </div>
                  <span>${t.contact_name || t.contact_email}</span>
                </div>
                ${getStatusBadge(t.status_key, t.status_label)}
              </div>
            </div>
          `).join('')}
        </div>
      </section>

      <!-- 3. Right Ticket Detail & Conversation Stream -->
      <main class="flex-1 bg-slate-950 flex flex-col overflow-hidden">
        ${!selected ? `
          <div class="flex-1 flex flex-col items-center justify-center text-slate-500 text-xs p-8">
            <div class="text-4xl mb-3">📬</div>
            <h2 class="text-base font-bold text-slate-300">Select a ticket from the queue</h2>
            <p class="text-slate-500 text-xs mt-1">View timeline, update status, and write customer responses.</p>
          </div>
        ` : `
          <!-- Detail Header -->
          <div class="bg-slate-900/90 backdrop-blur border-b border-slate-800 p-4 flex items-center justify-between shrink-0 shadow-sm">
            <div class="flex items-center gap-4">
              <div>
                <div class="flex items-center gap-2.5">
                  <span class="font-mono text-xs font-black text-indigo-400 bg-indigo-500/10 px-2 py-0.5 rounded-lg border border-indigo-500/30">${selected.reference_no}</span>
                  <h1 class="font-extrabold text-base text-white">${selected.subject}</h1>
                </div>
                <div class="flex items-center gap-2 text-xs text-slate-400 mt-1">
                  <span>Contact: <strong class="text-slate-200 font-semibold">${selected.contact_name}</strong> (${selected.contact_email})</span>
                  <span>•</span>
                  <span>Org: <strong class="text-slate-200 font-semibold">${selected.organization_name || 'Individual'}</strong></span>
                </div>
              </div>
            </div>

            <!-- Quick Status Change -->
            <div class="flex items-center gap-3">
              <select onchange="handleUpdateTicketField('status_key', this.value)" class="bg-slate-800 border border-slate-700 text-slate-200 text-xs font-bold rounded-xl px-3 py-2 focus:outline-none focus:border-indigo-500">
                <option value="new" ${selected.status_key === 'new' ? 'selected' : ''}>Status: New</option>
                <option value="open" ${selected.status_key === 'open' ? 'selected' : ''}>Status: Open</option>
                <option value="pending" ${selected.status_key === 'pending' ? 'selected' : ''}>Status: Pending Customer</option>
                <option value="on_hold" ${selected.status_key === 'on_hold' ? 'selected' : ''}>Status: On Hold</option>
                <option value="resolved" ${selected.status_key === 'resolved' ? 'selected' : ''}>Status: Resolved</option>
                <option value="closed" ${selected.status_key === 'closed' ? 'selected' : ''}>Status: Closed</option>
              </select>
            </div>
          </div>

          <!-- Main Scrollable Conversation Timeline & Side Metadata -->
          <div class="flex-1 flex overflow-hidden">
            <!-- Timeline Stream -->
            <div class="flex-1 overflow-y-auto p-6 space-y-6">
              <!-- Initial Customer Message -->
              <div class="p-5 rounded-2xl event-customer-inbound space-y-3 shadow-md">
                <div class="flex items-center justify-between border-b border-slate-700/50 pb-3">
                  <div class="flex items-center gap-3">
                    <div class="w-8 h-8 rounded-full bg-indigo-600 flex items-center justify-center font-extrabold text-xs text-white shadow">
                      ${(selected.contact_name || selected.contact_email)[0].toUpperCase()}
                    </div>
                    <div>
                      <div class="font-bold text-xs text-slate-200">${selected.contact_name}</div>
                      <div class="text-[11px] text-slate-400">${selected.contact_email}</div>
                    </div>
                  </div>
                  <span class="text-xs text-slate-400 font-mono">${formatRelativeTime(selected.created_at)}</span>
                </div>
                <div class="text-xs text-slate-200 leading-relaxed space-y-2">
                  ${selected.description_html || `<p>${selected.description_text}</p>`}
                </div>
              </div>

              <!-- Subsequent Timeline Events -->
              ${state.timelineEvents.map(ev => `
                <div class="p-5 rounded-2xl ${ev.visibility === 'internal' ? 'event-internal-note' : 'event-public-reply'} space-y-3 shadow-md">
                  <div class="flex items-center justify-between border-b ${ev.visibility === 'internal' ? 'border-amber-500/20' : 'border-slate-700/50'} pb-3">
                    <div class="flex items-center gap-3">
                      <div class="w-8 h-8 rounded-full ${ev.visibility === 'internal' ? 'bg-amber-600' : 'bg-slate-700'} flex items-center justify-center font-extrabold text-xs text-white">
                        ${(ev.author_name || 'A')[0].toUpperCase()}
                      </div>
                      <div>
                        <div class="flex items-center gap-2">
                          <span class="font-bold text-xs text-slate-200">${ev.author_name || 'Support Agent'}</span>
                          ${ev.visibility === 'internal' ? `
                            <span class="px-2 py-0.5 rounded text-[10px] font-extrabold bg-amber-500/20 text-amber-300 border border-amber-500/30">Private Internal Note</span>
                          ` : `
                            <span class="px-2 py-0.5 rounded text-[10px] font-extrabold bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">Public Reply</span>
                          `}
                        </div>
                        <div class="text-[11px] text-slate-400 capitalize">${ev.kind.replace('_', ' ')}</div>
                      </div>
                    </div>
                    <span class="text-xs text-slate-400 font-mono">${formatRelativeTime(ev.occurred_at)}</span>
                  </div>
                  <div class="text-xs text-slate-200 leading-relaxed">
                    ${ev.body_html || `<p>${ev.body_text}</p>`}
                  </div>
                </div>
              `).join('')}

              <!-- Dual Reply / Note Composer -->
              <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800 shadow-2xl space-y-3">
                <div class="flex items-center justify-between border-b border-slate-800 pb-2.5">
                  <div class="flex items-center gap-2">
                    <button 
                      onclick="setComposerMode('reply')" 
                      class="px-3.5 py-1.5 rounded-xl text-xs font-extrabold transition ${state.composerMode === 'reply' ? 'bg-indigo-600 text-white shadow-md' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
                    >
                      ✉️ Public Reply
                    </button>
                    <button 
                      onclick="setComposerMode('note')" 
                      class="px-3.5 py-1.5 rounded-xl text-xs font-extrabold transition ${state.composerMode === 'note' ? 'bg-amber-600 text-white shadow-md' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
                    >
                      🔒 Internal Private Note
                    </button>
                  </div>
                </div>

                <textarea
                  id="composer-input"
                  rows="4"
                  dir="auto"
                  placeholder="${state.composerMode === 'reply' ? 'Type public reply to the customer...' : 'Type team-only private note...'}"
                  oninput="state.replyText = this.value"
                  onkeydown="if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') { event.preventDefault(); handleSendReply(); }"
                  class="w-full bg-slate-950 border border-slate-800 rounded-xl p-3.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 leading-relaxed resize-none"
                >${state.replyText || ''}</textarea>

                <div class="flex items-center justify-between pt-1">
                  <div class="text-[11px] text-slate-500">
                    Press <kbd class="px-1.5 py-0.5 bg-slate-800 rounded text-slate-300 font-mono text-[10px]">Ctrl + Enter</kbd> to submit
                  </div>
                  <button 
                    id="composer-submit-btn"
                    onclick="handleSendReply()" 
                    class="${state.composerMode === 'note' ? 'bg-amber-600 hover:bg-amber-500' : 'bg-indigo-600 hover:bg-indigo-500'} text-white font-extrabold text-xs px-5 py-2 rounded-xl shadow-lg transition disabled:opacity-50"
                  >
                    ${state.composerMode === 'note' ? 'Save Internal Note' : 'Send Public Reply'}
                  </button>
                </div>
              </div>
            </div>

            <!-- Right Metadata Card -->
            <aside class="w-72 bg-slate-900/70 border-l border-slate-800 p-5 space-y-6 shrink-0 overflow-y-auto text-xs">
              <div class="space-y-3 pb-4 border-b border-slate-800">
                <div class="text-[11px] font-extrabold uppercase tracking-wider text-slate-400">Customer Profile</div>
                <div class="flex items-center gap-3">
                  <div class="w-10 h-10 rounded-full bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center font-extrabold text-sm text-white shadow">
                    ${(selected.contact_name || selected.contact_email)[0].toUpperCase()}
                  </div>
                  <div>
                    <div class="font-bold text-sm text-slate-100">${selected.contact_name}</div>
                    <div class="text-xs text-slate-400">${selected.contact_email}</div>
                  </div>
                </div>
                <div class="space-y-1.5 pt-2 text-[11px]">
                  <div class="flex justify-between text-slate-400">
                    <span>Organization</span>
                    <strong class="text-slate-200">${selected.organization_name || 'Individual'}</strong>
                  </div>
                  <div class="flex justify-between text-slate-400">
                    <span>Language</span>
                    <strong class="text-slate-200 uppercase">${selected.contact_locale || 'en'}</strong>
                  </div>
                </div>
              </div>
            </aside>
          </div>
        `}
      </main>
    </div>
  `;
}

// ----------------- Surface 2: Customer Portal (/portal) -----------------
async function loadPortalData() {
  if (!state.token) return;
  try {
    const data = await api('/portal/tickets');
    state.portalTickets = data.items || [];
    if (state.portalTickets.length > 0) {
      const currentSelectedId = state.selectedPortalTicket ? (state.selectedPortalTicket.id || state.selectedPortalTicket.ticket?.id) : null;
      const target = currentSelectedId ? state.portalTickets.find(t => t.id === currentSelectedId) : state.portalTickets[0];
      await selectPortalTicket(target ? target.id : state.portalTickets[0].id);
    } else {
      state.selectedPortalTicket = null;
      render();
    }
  } catch (err) {
    console.error('Failed to load portal tickets:', err);
    render();
  }
}

async function selectPortalTicket(id) {
  try {
    const data = await api(`/portal/tickets/${id}`);
    const ticketData = data.ticket || data;
    state.selectedPortalTicket = {
      ...ticketData,
      events: data.events || [],
    };
    render();
  } catch (err) {
    console.error('Failed to load portal ticket detail:', err);
  }
}

async function handleCustomerReply() {
  const input = document.getElementById('portal-reply-input');
  const text = (input ? input.value : (state.portalReplyText || '')).trim();
  if (!state.selectedPortalTicket) {
    alert('Please select a ticket first.');
    return;
  }
  if (!text) {
    alert('Please enter a message before sending.');
    if (input) input.focus();
    return;
  }

  const ticketId = state.selectedPortalTicket.id || state.selectedPortalTicket.ticket?.id;
  if (!ticketId) return;

  const btn = document.getElementById('portal-reply-btn');
  if (btn) {
    btn.disabled = true;
    btn.innerText = 'Sending...';
  }

  try {
    await api(`/portal/tickets/${ticketId}/reply`, {
      method: 'POST',
      body: JSON.stringify({ body_text: text }),
    });
    if (input) input.value = '';
    state.portalReplyText = '';
    await selectPortalTicket(ticketId);
  } catch (err) {
    alert('Failed to send reply: ' + err.message);
  } finally {
    if (btn) {
      btn.disabled = false;
      btn.innerText = 'Send Message';
    }
  }
}

function renderCustomerPortal() {
  if (!state.user) {
    return renderAuthGateway();
  }

  return `
    <div class="max-w-6xl mx-auto p-8 w-full space-y-8 flex-1">
      <div class="flex items-center justify-between border-b border-slate-800 pb-6">
        <div>
          <h1 class="text-2xl font-black text-white">Your Support Portal</h1>
          <p class="text-xs text-slate-400 mt-1">Track ticket status, view responses, and reply directly to our support engineers.</p>
        </div>
        <button onclick="navigate('/submit')" class="bg-indigo-600 hover:bg-indigo-500 text-white font-extrabold text-xs px-4 py-2.5 rounded-xl shadow-lg shadow-indigo-600/30 transition flex items-center gap-2">
          ➕ Submit New Request
        </button>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <!-- Ticket List Column -->
        <div class="space-y-3">
          <h2 class="font-extrabold text-sm text-slate-300">Your Tickets (${state.portalTickets.length})</h2>
          <div class="space-y-3">
            ${state.portalTickets.length === 0 ? `
              <div class="p-8 bg-slate-900 rounded-2xl border border-slate-800 text-center text-xs text-slate-500">
                You have no active support tickets.
              </div>
            ` : state.portalTickets.map(t => `
              <div 
                onclick="selectPortalTicket('${t.id}')"
                class="p-4 rounded-2xl cursor-pointer transition border ${state.selectedPortalTicket && state.selectedPortalTicket.id === t.id ? 'bg-indigo-950/40 border-indigo-500' : 'bg-slate-900 border-slate-800 hover:bg-slate-850'}"
              >
                <div class="flex items-center justify-between mb-1.5">
                  <span class="font-mono text-xs font-extrabold text-indigo-400">${t.reference_no}</span>
                  ${getStatusBadge(t.status_key, t.status_label)}
                </div>
                <h3 dir="auto" class="font-bold text-xs text-white line-clamp-1 mb-1">${t.subject}</h3>
                <span class="text-[10px] text-slate-500 font-mono">${formatRelativeTime(t.updated_at)}</span>
              </div>
            `).join('')}
          </div>
        </div>

        <!-- Ticket Detail Column -->
        <div class="md:col-span-2 bg-slate-900 rounded-2xl border border-slate-800 p-6 space-y-6">
          ${!state.selectedPortalTicket ? `
            <div class="p-12 text-center text-slate-500 text-xs">
              Select a ticket from the left list to view conversation.
            </div>
          ` : `
            <div class="flex items-center justify-between border-b border-slate-800 pb-4">
              <div>
                <span class="font-mono text-xs font-black text-indigo-400">${state.selectedPortalTicket.reference_no}</span>
                <h2 dir="auto" class="text-base font-extrabold text-white mt-1">${state.selectedPortalTicket.subject}</h2>
              </div>
              ${getStatusBadge(state.selectedPortalTicket.status_key, state.selectedPortalTicket.status_label)}
            </div>

            <!-- Thread -->
            <div class="space-y-4 max-h-96 overflow-y-auto pr-2">
              <div class="p-4 rounded-xl bg-slate-800/80 border border-slate-700 text-xs space-y-2">
                <div class="font-bold text-indigo-300">Your Initial Request:</div>
                <div dir="auto" class="text-slate-200 leading-relaxed">${state.selectedPortalTicket.description_html || state.selectedPortalTicket.description_text}</div>
              </div>

              ${(state.selectedPortalTicket.events || []).map(ev => `
                <div class="p-4 rounded-xl bg-slate-800/50 border border-slate-700 text-xs space-y-1.5">
                  <div class="flex justify-between text-[11px] font-bold text-slate-400">
                    <span>${ev.author_name || 'Support Agent'}</span>
                    <span class="font-mono font-normal">${formatRelativeTime(ev.occurred_at)}</span>
                  </div>
                  <div dir="auto" class="text-slate-200 leading-relaxed">${ev.body_html || ev.body_text}</div>
                </div>
              `).join('')}
            </div>

            <!-- Customer Reply Input -->
            <div class="space-y-3 pt-4 border-t border-slate-800">
              <textarea 
                rows="3" 
                dir="auto"
                id="portal-reply-input"
                placeholder="Write a message to our support team..."
                onkeydown="if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') { event.preventDefault(); handleCustomerReply(); }"
                class="w-full bg-slate-950 border border-slate-800 rounded-xl p-3.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 resize-none"
              ></textarea>
              <button 
                id="portal-reply-btn"
                onclick="handleCustomerReply()" 
                class="bg-indigo-600 hover:bg-indigo-500 text-white font-extrabold text-xs px-5 py-2.5 rounded-xl transition shadow disabled:opacity-50"
              >
                Send Message
              </button>
            </div>
          `}
        </div>
      </div>
    </div>
  `;
}

// ----------------- Surface 3: Public Knowledge Base (/kb) -----------------
async function loadKBData() {
  try {
    const spaces = await api('/kb/spaces');
    state.kbSpaces = spaces || [];
  } catch (err) {
    console.error('Failed to load KB data:', err);
  }
}

function renderKnowledgeBase() {
  return `
    <div class="max-w-5xl mx-auto p-8 w-full space-y-12 flex-1">
      <div class="text-center space-y-4 py-8">
        <h1 class="text-4xl font-black text-white tracking-tight">How can we help you?</h1>
        <p class="text-slate-400 text-sm max-w-lg mx-auto">Explore documentation, deployment guides, API references, and troubleshooting.</p>
        <div class="max-w-xl mx-auto relative">
          <input 
            type="text" 
            placeholder="Search articles (e.g. Deployment, PostgreSQL, Email)..."
            class="w-full bg-slate-900 border border-slate-700 rounded-2xl px-5 py-3.5 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 shadow-xl"
          />
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        ${state.kbSpaces.map(s => `
          <div class="p-6 rounded-2xl bg-slate-900/90 border border-slate-800 hover:border-indigo-500/50 transition cursor-pointer group shadow-lg">
            <div class="w-10 h-10 rounded-xl bg-indigo-600/20 text-indigo-400 flex items-center justify-center font-bold text-lg mb-4 group-hover:scale-110 transition">
              📖
            </div>
            <h3 class="font-bold text-base text-white group-hover:text-indigo-400 transition">${s.name}</h3>
            <p class="text-xs text-slate-400 mt-2 leading-relaxed">Official guides and documentation for our support platform.</p>
            <div class="mt-4 pt-4 border-t border-slate-800/80 flex items-center justify-between text-xs text-indigo-400 font-semibold">
              <span>Browse Categories</span>
              <span>→</span>
            </div>
          </div>
        `).join('')}
      </div>
    </div>
  `;
}

// ----------------- Surface 5: Anonymous Intake Form (/submit) -----------------
async function handleSubmitTicket(e) {
  e.preventDefault();
  const form = e.target;
  const submitBtn = form.querySelector('button[type="submit"]');
  submitBtn.disabled = true;
  submitBtn.innerText = 'Submitting...';

  try {
    const res = await api('/submit/ticket', {
      method: 'POST',
      body: JSON.stringify({
        full_name: form.full_name.value.trim(),
        email: form.email.value.trim(),
        subject: form.subject.value.trim(),
        description: form.description.value.trim(),
      }),
    });
    alert(`Success! Ticket created with Reference: ${res.reference_no}`);
    form.reset();
    if (state.user) {
      if (state.user.role === 'contact') {
        navigate('/portal');
      } else {
        navigate('/app');
      }
    } else {
      navigate('/portal');
    }
  } catch (err) {
    alert('Submission failed: ' + err.message);
  } finally {
    submitBtn.disabled = false;
    submitBtn.innerText = 'Submit Ticket';
  }
}

function renderSubmitForm() {
  const defaultName = state.user ? (state.user.full_name || '') : '';
  const defaultEmail = state.user ? (state.user.email || '') : '';

  return `
    <div class="max-w-2xl mx-auto p-8 w-full space-y-6 flex-1">
      <div class="border-b border-slate-800 pb-4">
        <h1 class="text-2xl font-black text-white">Submit a Support Ticket</h1>
        <p class="text-xs text-slate-400 mt-1">Please provide details about your inquiry so our team can resolve it quickly.</p>
      </div>

      <form onsubmit="handleSubmitTicket(event)" class="space-y-4 bg-slate-900 p-6 rounded-2xl border border-slate-800 shadow-xl">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-xs font-bold text-slate-300 mb-1.5">Your Full Name *</label>
            <input type="text" name="full_name" required value="${defaultName}" placeholder="Jane Doe" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-indigo-500" />
          </div>
          <div>
            <label class="block text-xs font-bold text-slate-300 mb-1.5">Your Work Email *</label>
            <input type="email" name="email" required value="${defaultEmail}" placeholder="name@company.com" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-indigo-500" />
          </div>
        </div>

        <div>
          <label class="block text-xs font-bold text-slate-300 mb-1.5">Subject *</label>
          <input type="text" name="subject" required placeholder="Brief summary of the issue" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-indigo-500" />
        </div>

        <div>
          <label class="block text-xs font-bold text-slate-300 mb-1.5">Detailed Description *</label>
          <textarea name="description" required rows="5" placeholder="Steps to reproduce, environment, error messages..." class="w-full bg-slate-950 border border-slate-800 rounded-xl p-3.5 text-xs text-white focus:outline-none focus:border-indigo-500 leading-relaxed resize-none"></textarea>
        </div>

        <button type="submit" class="w-full bg-indigo-600 hover:bg-indigo-500 text-white font-extrabold text-xs py-3 rounded-xl shadow-lg shadow-indigo-600/30 transition">
          Submit Ticket
        </button>
      </form>
    </div>
  `;
}

// ----------------- State Mutators -----------------
function setFilter(filter) {
  state.activeFilter = filter;
  loadAgentData();
}

function handleSearchInput(e) {
  state.searchQuery = e.target.value;
  loadAgentData();
}

function setComposerMode(mode) {
  state.composerMode = mode;
  render();
}

function setAdminTab(tab) {
  state.adminTab = tab;
  render();
}



// ----------------- Main Render Router -----------------
function render() {
  const app = document.getElementById('app');
  const path = window.location.pathname;

  let bodyHTML = '';

  if (path === '/login') {
    bodyHTML = renderAuthGateway();
  } else if (path.startsWith('/portal')) {
    bodyHTML = renderCustomerPortal();
  } else if (path.startsWith('/kb')) {
    bodyHTML = renderKnowledgeBase();
  } else if (path.startsWith('/submit')) {
    bodyHTML = renderSubmitForm();
  } else if (path.startsWith('/app')) {
    bodyHTML = renderAgentApp();
  } else {
    // Default root route
    if (!state.user) {
      bodyHTML = renderAuthGateway();
    } else if (state.user.role === 'admin' || state.user.role === 'agent') {
      bodyHTML = renderAgentApp();
    } else {
      bodyHTML = renderCustomerPortal();
    }
  }

  app.innerHTML = `
    ${renderNavbar()}
    ${bodyHTML}
  `;
}

window.addEventListener('DOMContentLoaded', () => {
  render();
  const path = window.location.pathname;
  if (path.startsWith('/portal')) loadPortalData();
  else if (path.startsWith('/kb')) loadKBData();
  else if (path.startsWith('/app') || path === '/') {
    if (state.token && state.user && (state.user.role === 'admin' || state.user.role === 'agent')) {
      loadAgentData();
    }
  }
});

// Window Bindings for Inline Listeners
window.navigate = navigate;
window.selectTicket = selectTicket;
window.selectPortalTicket = selectPortalTicket;
window.setFilter = setFilter;
window.handleSearchInput = handleSearchInput;
window.handleUpdateTicketField = handleUpdateTicketField;
window.setComposerMode = setComposerMode;
window.setAuthMode = setAuthMode;
window.handleSendReply = handleSendReply;
window.handleCustomerReply = handleCustomerReply;
window.handleSubmitTicket = handleSubmitTicket;
window.handleLoginSubmit = handleLoginSubmit;
window.handleRegisterSubmit = handleRegisterSubmit;
window.handleLogout = handleLogout;
