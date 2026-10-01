// OE Watcher Chrome Extension - Popup Logic
document.addEventListener('DOMContentLoaded', async () => {
  const toggleEnabled = document.getElementById('toggleEnabled');
  const toggleAutoSkip = document.getElementById('toggleAutoSkip');
  const statusBadge = document.getElementById('statusBadge');
  const statusText = document.getElementById('statusText');
  const statTabCount = document.getElementById('statTabCount');
  const statTotalPersons = document.getElementById('statTotalPersons');
  const btnSync = document.getElementById('btnSync');
  const btnOpenDashboard = document.getElementById('btnOpenDashboard');
  const lastSyncText = document.getElementById('lastSyncText');

  const whitelistCard = document.getElementById('whitelistCard');
  const whitelistIcon = document.getElementById('whitelistIcon');
  const whitelistTitle = document.getElementById('whitelistTitle');
  const whitelistDesc = document.getElementById('whitelistDesc');
  const btnToggleExcludeUrl = document.getElementById('btnToggleExcludeUrl');

  let activeTab = null;
  let isCurrentPageExcluded = false;
  let matchingExcludeEntry = null;

  // 1. Load initial storage
  const data = await chrome.storage.local.get(['settings', 'persons', 'apiConnected', 'excludeList']);
  const settings = data.settings || {};
  const persons = data.persons || [];
  let excludeList = data.excludeList || [];

  // Toggle switches
  toggleEnabled.checked = settings.enabled !== false;
  if (toggleAutoSkip) {
    toggleAutoSkip.checked = settings.autoSkip !== false;
  }

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

  function updateWhitelistUI(isExcluded, url) {
    isCurrentPageExcluded = isExcluded;
    if (!url || !url.startsWith('http')) {
      if (whitelistCard) whitelistCard.style.display = 'none';
      return;
    }
    if (whitelistCard) whitelistCard.style.display = 'flex';

    if (isExcluded) {
      whitelistCard.classList.add('is-whitelisted');
      whitelistIcon.textContent = '🛡️';
      whitelistTitle.textContent = 'Bu Sayfa Muaf Listesinde';
      whitelistDesc.textContent = 'Bu adreste filtreleme ve buzlama kapalı';
      btnToggleExcludeUrl.textContent = '✕ Muafiyeti Kaldır';
      btnToggleExcludeUrl.className = 'btn btn-whitelist remove';
      statTabCount.textContent = 'MUAF';
      statTabCount.className = 'stat-val text-blue';
    } else {
      whitelistCard.classList.remove('is-whitelisted');
      whitelistIcon.textContent = '🌐';
      whitelistTitle.textContent = 'Sayfa Filtreleniyor';
      whitelistDesc.textContent = 'Kara listedeki isimler filtrelenir';
      btnToggleExcludeUrl.textContent = '🛡️ Bu URL\'yi Muaf Tut';
      btnToggleExcludeUrl.className = 'btn btn-whitelist';
      statTabCount.className = 'stat-val text-red';
    }
  }

  // 2. Query active tab for real-time shielded count and whitelist status
  try {
    const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
    activeTab = tabs[0];
    if (activeTab && activeTab.id) {
      chrome.tabs.sendMessage(activeTab.id, { type: 'GET_PAGE_STATS' }, (res) => {
        if (!chrome.runtime.lastError && res) {
          if (res.isExcluded) {
            updateWhitelistUI(true, activeTab.url);
          } else {
            updateWhitelistUI(false, activeTab.url);
            if (typeof res.shieldedCount === 'number') {
              statTabCount.textContent = String(res.shieldedCount);
            }
          }
        } else {
          // Fallback if content script not responding yet
          updateWhitelistUI(false, activeTab.url);
          chrome.action.getBadgeText({ tabId: activeTab.id }).then(b => {
            if (b === 'MUAF') {
              updateWhitelistUI(true, activeTab.url);
            } else {
              statTabCount.textContent = b || '0';
            }
          }).catch(() => {});
        }
      });
    }
  } catch (_) {}

  // 3. Toggle change handlers
  toggleEnabled.addEventListener('change', async () => {
    const updatedSettings = { ...settings, enabled: toggleEnabled.checked };
    await chrome.storage.local.set({ settings: updatedSettings });
  });

  if (toggleAutoSkip) {
    toggleAutoSkip.addEventListener('change', async () => {
      chrome.runtime.sendMessage({
        type: 'SET_AUTO_SKIP',
        autoSkip: toggleAutoSkip.checked
      });
    });
  }

  // 4. Whitelist button handler
  if (btnToggleExcludeUrl) {
    btnToggleExcludeUrl.addEventListener('click', async () => {
      if (!activeTab || !activeTab.url || !activeTab.url.startsWith('http')) return;
      btnToggleExcludeUrl.disabled = true;

      const currentUrl = activeTab.url;
      const targetType = isCurrentPageExcluded ? 'REMOVE_EXCLUDE' : 'ADD_EXCLUDE';

      chrome.runtime.sendMessage({ type: targetType, url: currentUrl }, (res) => {
        btnToggleExcludeUrl.disabled = false;
        if (res && res.success) {
          updateWhitelistUI(!isCurrentPageExcluded, currentUrl);

          // Rescan or reload active tab
          chrome.tabs.sendMessage(activeTab.id, { type: 'RESCAN' }, (scanRes) => {
            if (scanRes && typeof scanRes.shieldedCount === 'number') {
              if (scanRes.isExcluded) {
                updateWhitelistUI(true, currentUrl);
              } else {
                statTabCount.textContent = String(scanRes.shieldedCount);
              }
            }
          });
        }
      });
    });
  }

  // 5. Sync button handler
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
        if (activeTab && activeTab.id) {
          chrome.tabs.sendMessage(activeTab.id, { type: 'RESCAN' }, (scanRes) => {
            if (scanRes) {
              if (scanRes.isExcluded) {
                updateWhitelistUI(true, activeTab.url);
              } else if (typeof scanRes.shieldedCount === 'number') {
                statTabCount.textContent = String(scanRes.shieldedCount);
              }
            }
          });
        }
      } else {
        updateApiStatus(false);
      }
    });
  });

  // 6. Open Dashboard button
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
