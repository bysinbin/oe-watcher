// OE Watcher Chrome Extension - Popup Logic
document.addEventListener('DOMContentLoaded', async () => {
  const toggleEnabled = document.getElementById('toggleEnabled');
  const statusBadge = document.getElementById('statusBadge');
  const statusText = document.getElementById('statusText');
  const statTabCount = document.getElementById('statTabCount');
  const statTotalPersons = document.getElementById('statTotalPersons');
  const btnSync = document.getElementById('btnSync');
  const btnOpenDashboard = document.getElementById('btnOpenDashboard');
  const lastSyncText = document.getElementById('lastSyncText');

  // 1. Load initial storage
  const data = await chrome.storage.local.get(['settings', 'persons', 'apiConnected']);
  const settings = data.settings || {};
  const persons = data.persons || [];

  // Toggle switch
  toggleEnabled.checked = settings.enabled !== false;

  // Person count
  const count = persons.length || settings.personsCount || 0;
  statTotalPersons.textContent = count > 0 ? count : '574';

  // API status
  updateApiStatus(data.apiConnected);

  // Last sync
  if (settings.lastSync) {
    const d = new Date(settings.lastSync);
    lastSyncText.textContent = `Son senkron: ${d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
  }

  // 2. Query active tab for real-time shielded count
  try {
    const [activeTab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (activeTab && activeTab.id) {
      chrome.tabs.sendMessage(activeTab.id, { type: 'GET_PAGE_STATS' }, (res) => {
        if (!chrome.runtime.lastError && res && typeof res.shieldedCount === 'number') {
          statTabCount.textContent = String(res.shieldedCount);
        } else {
          chrome.action.getBadgeText({ tabId: activeTab.id }).then(b => {
            statTabCount.textContent = b || '0';
          }).catch(() => {});
        }
      });
    }
  } catch (_) {}

  // 3. Toggle change handler
  toggleEnabled.addEventListener('change', async () => {
    const updatedSettings = { ...settings, enabled: toggleEnabled.checked };
    await chrome.storage.local.set({ settings: updatedSettings });
  });

  // 4. Sync button handler
  btnSync.addEventListener('click', async () => {
    btnSync.disabled = true;
    btnSync.innerHTML = '<span class="btn-icon">⏳</span> Senkronize Ediliyor...';

    chrome.runtime.sendMessage({ type: 'SYNC_NOW' }, (res) => {
      btnSync.disabled = false;
      btnSync.innerHTML = '<span class="btn-icon">🔄</span> Listeyi Senkronize Et';

      if (res && res.success) {
        updateApiStatus(true);
        statTotalPersons.textContent = res.count;
        const now = new Date();
        lastSyncText.textContent = `Son senkron: ${now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;

        // Trigger rescan on active tab
        chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
          if (tabs[0] && tabs[0].id) {
            chrome.tabs.sendMessage(tabs[0].id, { type: 'RESCAN' }, (scanRes) => {
              if (scanRes && typeof scanRes.shieldedCount === 'number') {
                statTabCount.textContent = String(scanRes.shieldedCount);
              }
            });
          }
        });
      } else {
        updateApiStatus(false);
      }
    });
  });

  // 5. Open Dashboard button
  btnOpenDashboard.addEventListener('click', () => {
    const url = settings.apiUrl || "http://127.0.0.1:8080";
    chrome.tabs.create({ url });
  });

  function updateApiStatus(connected) {
    if (connected) {
      statusBadge.className = 'status-indicator connected';
      statusText.textContent = 'OE Watcher Bağlı';
    } else {
      statusBadge.className = 'status-indicator offline';
      statusText.textContent = 'Önbellek Devrede';
    }
  }
});
