const DEFAULT_CONFIG = {
  enabled: true,
  autoSkip: true,
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

// Context Menu Setup
function setupContextMenus() {
  if (!chrome.contextMenus) return;
  chrome.contextMenus.removeAll(() => {
    chrome.contextMenus.create({
      id: 'oe-root',
      title: 'OE Watcher Kalkanı',
      contexts: ['selection', 'page']
    });

    chrome.contextMenus.create({
      id: 'oe-add-person',
      parentId: 'oe-root',
      title: 'Seçili Kişiyi Kara Listeye Ekle: "%s"',
      contexts: ['selection']
    });

    chrome.contextMenus.create({
      id: 'oe-add-keyword',
      parentId: 'oe-root',
      title: 'Seçili Kelimeyi / Tag\'i Engelle: "%s"',
      contexts: ['selection']
    });

    chrome.contextMenus.create({
      id: 'oe-add-exclude-selection',
      parentId: 'oe-root',
      title: 'Seçili Metni Muaf Tut (Beyaz Liste): "%s"',
      contexts: ['selection']
    });

    chrome.contextMenus.create({
      id: 'oe-exclude-page',
      parentId: 'oe-root',
      title: 'Bu Sayfayı / Alan Adını Muaf Tut',
      contexts: ['page']
    });
  });
}

// Initial setup on install
chrome.runtime.onInstalled.addListener(async () => {
  setupContextMenus();
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
      excludeList: fallback.excludeList,
      keywords: []
    });
  }

  // Then attempt live sync with local OE Watcher daemon
  await syncWithOEWatcher();
});

// Context Menu Click Handler
if (chrome.contextMenus) {
  chrome.contextMenus.onClicked.addListener(async (info, tab) => {
    const { settings = DEFAULT_CONFIG } = await chrome.storage.local.get('settings');
    const baseUrl = settings.apiUrl || "http://127.0.0.1:8080";
    const selectedText = (info.selectionText || '').trim();

    if (info.menuItemId === 'oe-add-person' && selectedText) {
      try {
        await fetch(`${baseUrl}/api/extension/person`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: selectedText })
        });
        await syncWithOEWatcher();
        if (tab && tab.id) {
          chrome.tabs.sendMessage(tab.id, { type: 'RESCAN' }).catch(() => {});
        }
      } catch (e) {
        console.error('Failed to add person via context menu:', e);
      }
    } else if (info.menuItemId === 'oe-add-keyword' && selectedText) {
      try {
        await fetch(`${baseUrl}/api/extension/keyword`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ keyword: selectedText })
        });
        await syncWithOEWatcher();
        if (tab && tab.id) {
          chrome.tabs.sendMessage(tab.id, { type: 'RESCAN' }).catch(() => {});
        }
      } catch (e) {
        console.error('Failed to add keyword via context menu:', e);
      }
    } else if (info.menuItemId === 'oe-add-exclude-selection' && selectedText) {
      try {
        await fetch(`${baseUrl}/api/extension/exclude`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: selectedText })
        });
        await syncWithOEWatcher();
        if (tab && tab.id) {
          chrome.tabs.sendMessage(tab.id, { type: 'RESCAN' }).catch(() => {});
        }
      } catch (e) {
        console.error('Failed to add exclude via context menu:', e);
      }
    } else if (info.menuItemId === 'oe-exclude-page') {
      const pageUrl = (info.pageUrl || (tab && tab.url) || '').trim();
      if (pageUrl) {
        try {
          const urlObj = new URL(pageUrl);
          const domain = urlObj.hostname;
          await fetch(`${baseUrl}/api/extension/exclude`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: domain })
          });
          await syncWithOEWatcher();
          if (tab && tab.id) {
            chrome.tabs.sendMessage(tab.id, { type: 'RESCAN' }).catch(() => {});
          }
        } catch (e) {
          console.error('Failed to exclude page via context menu:', e);
        }
      }
    }
  });
}

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
    const keywords = data.keywords || [];

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

    const autoSkip = data.auto_skip !== undefined ? !!data.auto_skip : (settings.autoSkip !== undefined ? settings.autoSkip : true);

    const updatedSettings = {
      ...settings,
      autoSkip: autoSkip,
      lastSync: new Date().toISOString(),
      personsCount: blacklist.length
    };

    await chrome.storage.local.set({
      settings: updatedSettings,
      persons: blacklist,
      aliases: aliases,
      excludeList: excludeList,
      keywords: keywords,
      apiConnected: true
    });

    console.log(`[OE Watcher Extension] Senkronizasyon başarılı: ${blacklist.length} kişi, ${keywords.length} kelime yüklendi. (AutoSkip: ${autoSkip})`);
    return { success: true, count: blacklist.length };
  } catch (err) {
    console.warn('[OE Watcher Extension] Yerel API erişilemedi, dahili veritabanı kontrol ediliyor:', err.message);

    // Fallback: check if we already have persons; if not, load from default_persons.json
    const existing = await chrome.storage.local.get('persons');
    if (!existing.persons || existing.persons.length === 0) {
      const fallback = await loadBundledFallback();
      const updatedSettings = {
        ...settings,
        autoSkip: settings.autoSkip !== undefined ? settings.autoSkip : true,
        lastSync: new Date().toISOString(),
        personsCount: fallback.persons.length
      };
      await chrome.storage.local.set({
        settings: updatedSettings,
        persons: fallback.persons,
        aliases: fallback.aliases,
        excludeList: fallback.excludeList,
        keywords: [],
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
      let data = await chrome.storage.local.get(['settings', 'persons', 'aliases', 'excludeList', 'keywords', 'apiConnected']);
      if (!data.persons || data.persons.length === 0) {
        const fallback = await loadBundledFallback();
        data.persons = fallback.persons;
        data.aliases = fallback.aliases;
        data.excludeList = fallback.excludeList;
        data.keywords = [];
        await chrome.storage.local.set({
          persons: fallback.persons,
          aliases: fallback.aliases,
          excludeList: fallback.excludeList,
          keywords: []
        });
      }
      sendResponse(data);
    })();
    return true; // Keep channel open for async response
  }

  if (msg.type === 'SET_AUTO_SKIP') {
    (async () => {
      const { settings = DEFAULT_CONFIG } = await chrome.storage.local.get('settings');
      const updated = { ...settings, autoSkip: !!msg.autoSkip };
      await chrome.storage.local.set({ settings: updated });

      chrome.tabs.query({}, (tabs) => {
        for (const t of tabs) {
          if (t.id) chrome.tabs.sendMessage(t.id, { type: 'CONFIG_CHANGED', settings: updated }).catch(() => {});
        }
      });

      const baseUrl = settings.apiUrl || "http://127.0.0.1:8080";
      try {
        await fetch(`${baseUrl}/api/extension/auto-skip`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ auto_skip: !!msg.autoSkip })
        });
      } catch (e) {}

      sendResponse({ success: true, autoSkip: updated.autoSkip });
    })();
    return true;
  }

  if (msg.type === 'MEDIA_SKIPPED') {
    (async () => {
      console.log('[OE Watcher Background] Medya otomatik atlandı:', msg.person, msg.track);
      const { settings = DEFAULT_CONFIG } = await chrome.storage.local.get('settings');
      const baseUrl = settings.apiUrl || "http://127.0.0.1:8080";
      try {
        await fetch(`${baseUrl}/api/extension/skip`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            track: msg.track,
            person: msg.person
          })
        });
      } catch (e) {}
      sendResponse({ success: true });
    })();
    return true;
  }

  if (msg.type === 'SYNC_NOW') {
    syncWithOEWatcher().then(res => {
      sendResponse(res);
    });
    return true;
  }

  if (msg.type === 'ADD_EXCLUDE') {
    (async () => {
      const url = (msg.url || '').trim();
      if (!url) {
        sendResponse({ success: false, error: 'URL boş olamaz' });
        return;
      }

      const data = await chrome.storage.local.get(['excludeList', 'settings']);
      const list = data.excludeList || [];
      if (!list.includes(url)) {
        list.push(url);
        await chrome.storage.local.set({ excludeList: list });
      }

      const baseUrl = (data.settings && data.settings.apiUrl) || 'http://127.0.0.1:8080';
      try {
        await fetch(`${baseUrl}/api/extension/exclude`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: url })
        });
      } catch (_) {}

      sendResponse({ success: true, list });
    })();
    return true;
  }

  if (msg.type === 'REMOVE_EXCLUDE') {
    (async () => {
      const url = (msg.url || '').trim();
      if (!url) {
        sendResponse({ success: false, error: 'URL boş olamaz' });
        return;
      }

      const data = await chrome.storage.local.get(['excludeList', 'settings']);
      let list = data.excludeList || [];
      list = list.filter(item => item !== url);
      await chrome.storage.local.set({ excludeList: list });

      const baseUrl = (data.settings && data.settings.apiUrl) || 'http://127.0.0.1:8080';
      try {
        await fetch(`${baseUrl}/api/extension/exclude`, {
          method: 'DELETE',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: url })
        });
      } catch (_) {}

      sendResponse({ success: true, list });
    })();
    return true;
  }

  if (msg.type === 'UPDATE_BADGE' && sender.tab && sender.tab.id) {
    if (msg.isExcluded) {
      chrome.action.setBadgeText({ tabId: sender.tab.id, text: 'MUAF' });
      chrome.action.setBadgeBackgroundColor({ tabId: sender.tab.id, color: '#10b981' });
    } else {
      const count = msg.count || 0;
      if (count > 0) {
        chrome.action.setBadgeText({ tabId: sender.tab.id, text: String(count) });
        chrome.action.setBadgeBackgroundColor({ tabId: sender.tab.id, color: '#e11d48' });
      } else {
        chrome.action.setBadgeText({ tabId: sender.tab.id, text: '' });
      }
    }
    sendResponse({ ok: true });
    return false;
  }
});

// Run sync at worker startup
syncWithOEWatcher();
