// OE Watcher Chrome Extension - Service Worker (Background)
const DEFAULT_CONFIG = {
  enabled: true,
  apiUrl: "http://127.0.0.1:8080",
  blurIntensity: "medium", // low, medium, high
  lastSync: null,
  personsCount: 0
};

// Initial setup on install
chrome.runtime.onInstalled.addListener(async () => {
  const current = await chrome.storage.local.get(['settings', 'persons', 'rules']);
  if (!current.settings) {
    await chrome.storage.local.set({ settings: DEFAULT_CONFIG });
  }
  await syncWithOEWatcher();
});

// Sync data periodically (every 15 minutes)
chrome.alarms.create('sync_oelist', { periodInMinutes: 15 });
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'sync_oelist') {
    syncWithOEWatcher();
  }
});

// Core sync function to fetch from OE Watcher local daemon
async function syncWithOEWatcher() {
  try {
    const { settings = DEFAULT_CONFIG } = await chrome.storage.local.get('settings');
    const baseUrl = settings.apiUrl || "http://127.0.0.1:8080";

    // 1. Fetch persons list
    const pRes = await fetch(`${baseUrl}/api/persons?category=HEPSİ`, {
      headers: { 'Accept': 'application/json' }
    });
    if (!pRes.ok) throw new Error(`HTTP ${pRes.status}`);
    const pData = await pRes.json();
    const persons = pData.persons || [];

    // 2. Fetch rules (aliases & exclude list)
    let aliases = {};
    let excludeList = [];
    try {
      const rRes = await fetch(`${baseUrl}/api/rules`, {
        headers: { 'Accept': 'application/json' }
      });
      if (rRes.ok) {
        const rData = await rRes.json();
        aliases = rData.aliases || {};
        excludeList = rData.exclude_list || [];
      }
    } catch (_) {}

    // Extract names for fast matching
    const blacklistNames = [];
    for (const p of persons) {
      if (p.display_name) {
        blacklistNames.push({
          id: p.id,
          name: p.display_name.trim(),
          tags: p.tags || [],
          topics: p.topics || []
        });
      }
    }

    const updatedSettings = {
      ...settings,
      lastSync: new Date().toISOString(),
      personsCount: blacklistNames.length
    };

    await chrome.storage.local.set({
      settings: updatedSettings,
      persons: blacklistNames,
      aliases: aliases,
      excludeList: excludeList,
      apiConnected: true
    });

    console.log(`[OE Watcher Extension] Senkronizasyon başarılı: ${blacklistNames.length} kişi yüklendi.`);
    return { success: true, count: blacklistNames.length };
  } catch (err) {
    console.warn('[OE Watcher Extension] Yerel API senkronizasyon hatası:', err.message);
    const { settings = DEFAULT_CONFIG } = await chrome.storage.local.get('settings');
    await chrome.storage.local.set({
      settings: { ...settings },
      apiConnected: false
    });
    return { success: false, error: err.message };
  }
}

// Handle runtime messages from content scripts and popup
chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.type === 'GET_DATA') {
    chrome.storage.local.get(['settings', 'persons', 'aliases', 'excludeList', 'apiConnected']).then(data => {
      sendResponse(data);
    });
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
