// OE Watcher Chrome Extension - Service Worker (Background)
const DEFAULT_CONFIG = {
  enabled: true,
  apiUrl: "http://127.0.0.1:8080",
  blurIntensity: "medium", // low, medium, high
  lastSync: null,
  personsCount: 574
};

// Helper to load bundled fallback data
async function loadBundledFallback() {
  try {
    const url = chrome.runtime.getURL('default_persons.json');
    const res = await fetch(url);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = await res.json();
    return {
      persons: data.persons || [],
      aliases: data.aliases || {},
      excludeList: data.exclude_list || []
    };
  } catch (err) {
    console.error('[OE Watcher] Fallback load error:', err);
    return { persons: [], aliases: {}, excludeList: [] };
  }
}

// Initial setup on install
chrome.runtime.onInstalled.addListener(async () => {
  const current = await chrome.storage.local.get(['settings', 'persons']);
  if (!current.settings) {
    await chrome.storage.local.set({ settings: DEFAULT_CONFIG });
  }

  // Pre-seed with bundled fallback if empty
  if (!current.persons || current.persons.length === 0) {
    const fallback = await loadBundledFallback();
    await chrome.storage.local.set({
      persons: fallback.persons,
      aliases: fallback.aliases,
      excludeList: fallback.excludeList
    });
  }

  // Then attempt live sync with local OE Watcher daemon
  await syncWithOEWatcher();
});

// Periodic background sync (every 15 minutes)
if (chrome.alarms) {
  chrome.alarms.create('sync_oelist', { periodInMinutes: 15 });
  chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name === 'sync_oelist') {
      syncWithOEWatcher();
    }
  });
}

// Core sync function to fetch from OE Watcher local daemon
async function syncWithOEWatcher() {
  const { settings = DEFAULT_CONFIG } = await chrome.storage.local.get('settings');
  const baseUrl = settings.apiUrl || "http://127.0.0.1:8080";

  try {
    // 1. Fetch from unauthenticated extension sync endpoint
    const res = await fetch(`${baseUrl}/api/extension/sync`, {
      headers: { 'Accept': 'application/json' },
      cache: 'no-store'
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = await res.json();

    const rawPersons = data.persons || [];
    const aliases = data.aliases || {};
    const excludeList = data.exclude_list || [];

    const blacklist = [];
    for (const p of rawPersons) {
      if (p.name) {
        blacklist.push({
          name: p.name.trim(),
          tags: p.tags || [],
          topics: p.topics || []
        });
      }
    }

    const updatedSettings = {
      ...settings,
      lastSync: new Date().toISOString(),
      personsCount: blacklist.length
    };

    await chrome.storage.local.set({
      settings: updatedSettings,
      persons: blacklist,
      aliases: aliases,
      excludeList: excludeList,
      apiConnected: true
    });

    console.log(`[OE Watcher Extension] Senkronizasyon başarılı: ${blacklist.length} kişi yüklendi.`);
    return { success: true, count: blacklist.length };
  } catch (err) {
    console.warn('[OE Watcher Extension] Yerel API erişilemedi, dahili veritabanı kontrol ediliyor:', err.message);

    // Fallback: check if we already have persons; if not, load from default_persons.json
    const existing = await chrome.storage.local.get('persons');
    if (!existing.persons || existing.persons.length === 0) {
      const fallback = await loadBundledFallback();
      const updatedSettings = {
        ...settings,
        lastSync: new Date().toISOString(),
        personsCount: fallback.persons.length
      };
      await chrome.storage.local.set({
        settings: updatedSettings,
        persons: fallback.persons,
        aliases: fallback.aliases,
        excludeList: fallback.excludeList,
        apiConnected: false
      });
      return { success: true, count: fallback.persons.length, fallback: true };
    }

    await chrome.storage.local.set({ apiConnected: false });
    return { success: false, error: err.message };
  }
}

// Handle runtime messages from content scripts and popup
chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.type === 'GET_DATA') {
    (async () => {
      let data = await chrome.storage.local.get(['settings', 'persons', 'aliases', 'excludeList', 'apiConnected']);
      if (!data.persons || data.persons.length === 0) {
        const fallback = await loadBundledFallback();
        data.persons = fallback.persons;
        data.aliases = fallback.aliases;
        data.excludeList = fallback.excludeList;
        await chrome.storage.local.set({
          persons: fallback.persons,
          aliases: fallback.aliases,
          excludeList: fallback.excludeList
        });
      }
      sendResponse(data);
    })();
    return true; // Keep channel open for async response
  }

  if (msg.type === 'SYNC_NOW') {
    syncWithOEWatcher().then(res => {
      sendResponse(res);
    });
    return true;
  }

  if (msg.type === 'UPDATE_BADGE' && sender.tab && sender.tab.id) {
    const count = msg.count || 0;
    if (count > 0) {
      chrome.action.setBadgeText({ tabId: sender.tab.id, text: String(count) });
      chrome.action.setBadgeBackgroundColor({ tabId: sender.tab.id, color: '#e11d48' });
    } else {
      chrome.action.setBadgeText({ tabId: sender.tab.id, text: '' });
    }
    sendResponse({ ok: true });
    return false;
  }
});

// Run sync at worker startup
syncWithOEWatcher();
