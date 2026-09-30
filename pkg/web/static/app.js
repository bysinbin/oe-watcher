document.addEventListener('DOMContentLoaded', () => {
  let selectedCategory = 'HEPSİ';
  let selectedTopic = 'TÜM KONULAR';
  let currentSearchQuery = '';
  let autoSkipEnabled = false;

  // DOM Elements
  const trackTitle = document.getElementById('trackTitle');
  const trackArtist = document.getElementById('trackArtist');
  const trackAlbum = document.getElementById('trackAlbum');
  const playerSourceBadge = document.getElementById('playerSourceBadge');
  const btnSkipManual = document.getElementById('btnSkipManual');
  const autoSkipToggle = document.getElementById('autoSkipToggle');
  const btnPlayPause = document.getElementById('btnPlayPause');
  const playPauseIcon = document.getElementById('playPauseIcon');
  const playPauseText = document.getElementById('playPauseText');
  const connStatus = document.getElementById('connStatus');
  const btnSync = document.getElementById('btnSync');
  const btnSyncText = document.getElementById('btnSyncText');
  const btnOpenSettings = document.getElementById('btnOpenSettings');

  // Stats Elements
  const statTotalDetections = document.getElementById('statTotalDetections');
  const statAutoSkips = document.getElementById('statAutoSkips');
  const statManualSkips = document.getElementById('statManualSkips');
  const statTopPerson = document.getElementById('statTopPerson');

  // Alert Box Elements
  const matchAlertBox = document.getElementById('matchAlertBox');
  const alertMatchName = document.getElementById('alertMatchName');
  const alertMatchTags = document.getElementById('alertMatchTags');
  const alertMatchTopics = document.getElementById('alertMatchTopics');
  const alertMatchCategory = document.getElementById('alertMatchCategory');
  const btnAlertSkip = document.getElementById('btnAlertSkip');
  const btnAlertSnooze = document.getElementById('btnAlertSnooze');
  const btnAlertKeep = document.getElementById('btnAlertKeep');
  const btnAlertAddAlias = document.getElementById('btnAlertAddAlias');
  const btnAlertAddExclude = document.getElementById('btnAlertAddExclude');

  // Modal Elements
  const aliasModal = document.getElementById('aliasModal');
  const modalTargetName = document.getElementById('modalTargetName');
  const modalAliasInput = document.getElementById('modalAliasInput');
  const btnModalClose = document.getElementById('btnModalClose');
  const btnModalCancel = document.getElementById('btnModalCancel');
  const btnModalSave = document.getElementById('btnModalSave');

  // Settings Modal Elements
  const settingsModal = document.getElementById('settingsModal');
  const settingsForm = document.getElementById('settingsForm');
  const btnSettingsClose = document.getElementById('btnSettingsClose');
  const btnSettingsCancel = document.getElementById('btnSettingsCancel');
  const cfgAutoSkip = document.getElementById('cfgAutoSkip');
  const cfgEnableDialog = document.getElementById('cfgEnableDialog');
  const cfgFuzzyMatching = document.getElementById('cfgFuzzyMatching');
  const cfgFuzzyThreshold = document.getElementById('cfgFuzzyThreshold');
  const fuzzyThresholdVal = document.getElementById('fuzzyThresholdVal');
  const cfgPollInterval = document.getElementById('cfgPollInterval');
  const cfgCategory = document.getElementById('cfgCategory');
  const cfgWebPassword = document.getElementById('cfgWebPassword');
  const cfgWebPort = document.getElementById('cfgWebPort');
  const cfgApiUrl = document.getElementById('cfgApiUrl');
  const cfgAutoSyncHours = document.getElementById('cfgAutoSyncHours');

  // Rules I/O Elements
  const btnExportRules = document.getElementById('btnExportRules');
  const btnImportRules = document.getElementById('btnImportRules');
  const rulesFileInput = document.getElementById('rulesFileInput');

  let currentModalTarget = '';
  let lastActiveMatch = null;

  // Person List & Search Elements
  const searchInput = document.getElementById('searchInput');
  const personsList = document.getElementById('personsList');
  const personCountBadge = document.getElementById('personCountBadge');
  const historyList = document.getElementById('historyList');

  // Login Overlay Elements
  const loginOverlay = document.getElementById('loginOverlay');
  const loginForm = document.getElementById('loginForm');
  const loginPassword = document.getElementById('loginPassword');
  const btnTogglePassword = document.getElementById('btnTogglePassword');
  const loginErrorMsg = document.getElementById('loginErrorMsg');
  const btnLogout = document.getElementById('btnLogout');

  let authToken = localStorage.getItem('oe_auth_token') || '';
  let authRequired = false;
  let pollingTimer = null;

  // Intercept fetch to automatically attach Bearer token and handle 401
  const origFetch = window.fetch;
  window.fetch = async (input, init = {}) => {
    init.headers = init.headers || {};
    const urlStr = typeof input === 'string' ? input : (input.url || '');
    if (authToken && urlStr.includes('/api/')) {
      if (init.headers instanceof Headers) {
        if (!init.headers.has('Authorization')) init.headers.set('Authorization', 'Bearer ' + authToken);
      } else if (!init.headers['Authorization']) {
        init.headers['Authorization'] = 'Bearer ' + authToken;
      }
    }
    const res = await origFetch(input, init);
    if (res.status === 401 && !urlStr.includes('/api/auth-status') && !urlStr.includes('/api/login')) {
      showLoginOverlay('Oturum süresi doldu veya şifre gerekli.');
    }
    return res;
  };

  // Category & Topic Grids
  const categoriesGrid = document.getElementById('categoriesGrid');
  const topicsGrid = document.getElementById('topicsGrid');

  // Initialize
  setupFilterButtons();
  setupEventListeners();
  setupRulesManagement();
  setupAuthHandlers();
  checkAuthAndStart();

  async function checkAuthAndStart() {
    try {
      const res = await apiFetch('/api/auth-status');
      const data = await res.json();
      authRequired = data.required;

      if (authRequired) {
        if (!data.authenticated) {
          showLoginOverlay();
          return;
        }
        if (btnLogout) btnLogout.style.display = 'inline-flex';
      }

      hideLoginOverlay();
      startApp();
    } catch (err) {
      console.error('Auth check error:', err);
      startApp();
    }
  }

  function startApp() {
    loadPersons();
    loadRules();
    setupSSE();
    pollStatus();
    if (!pollingTimer) {
      pollingTimer = setInterval(pollStatus, 3500);
    }
  }

  function showLoginOverlay(msg) {
    if (eventSource) {
      try { eventSource.close(); } catch(e) {}
      eventSource = null;
    }
    if (loginOverlay) {
      loginOverlay.style.display = 'flex';
      if (loginPassword) {
        loginPassword.value = '';
        setTimeout(() => loginPassword.focus(), 60);
      }
    }
    if (loginErrorMsg) {
      if (msg) {
        loginErrorMsg.textContent = msg;
        loginErrorMsg.style.display = 'block';
      } else {
        loginErrorMsg.style.display = 'none';
      }
    }
    if (btnLogout) btnLogout.style.display = 'none';
    if (pollingTimer) {
      clearInterval(pollingTimer);
      pollingTimer = null;
    }
  }

  function hideLoginOverlay() {
    if (loginOverlay) loginOverlay.style.display = 'none';
    if (loginErrorMsg) loginErrorMsg.style.display = 'none';
    if (authRequired && btnLogout) btnLogout.style.display = 'inline-flex';
  }

  function setupAuthHandlers() {
    if (loginForm) {
      loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const pwd = loginPassword.value;
        try {
          const res = await fetch('/api/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ password: pwd })
          });
          const data = await res.json();
          if (res.ok && data.success) {
            authToken = data.token || '';
            localStorage.setItem('oe_auth_token', authToken);
            hideLoginOverlay();
            showToast('🔒 Giriş başarılı!');
            startApp();
          } else {
            loginErrorMsg.textContent = data.error || 'Hatalı parola!';
            loginErrorMsg.style.display = 'block';
            loginPassword.select();
          }
        } catch (err) {
          loginErrorMsg.textContent = 'Bağlantı hatası: ' + err.message;
          loginErrorMsg.style.display = 'block';
        }
      });
    }

    if (btnTogglePassword && loginPassword) {
      btnTogglePassword.addEventListener('click', () => {
        const isPwd = loginPassword.getAttribute('type') === 'password';
        loginPassword.setAttribute('type', isPwd ? 'text' : 'password');
        btnTogglePassword.textContent = isPwd ? '🔒' : '👁️';
      });
    }

    if (btnLogout) {
      btnLogout.addEventListener('click', async () => {
        try {
          await apiFetch('/api/logout', { method: 'POST' });
        } catch (err) {}
        authToken = '';
        localStorage.removeItem('oe_auth_token');
        showToast('Oturum kapatıldı.');
        showLoginOverlay();
      });
    }
  }

  function setupFilterButtons() {
    // Category Buttons
    categoriesGrid.querySelectorAll('.filter-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        categoriesGrid.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        selectedCategory = btn.getAttribute('data-category');
        updateSettingsFilters();
        loadPersons();
      });
    });

    // Topic Buttons
    topicsGrid.querySelectorAll('.filter-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        topicsGrid.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        selectedTopic = btn.getAttribute('data-topic');
        updateSettingsFilters();
        loadPersons();
      });
    });
  }

  function setupEventListeners() {
    // Manual Skip
    btnSkipManual.addEventListener('click', triggerSkip);
    btnAlertSkip.addEventListener('click', triggerSkip);

    if (btnPlayPause) {
      btnPlayPause.addEventListener('click', async () => {
        try {
          await fetch('/api/player/play-pause', { method: 'POST' });
        } catch (err) {
          console.error('Play/Pause error:', err);
        }
      });
    }

    btnAlertKeep.addEventListener('click', () => {
      matchAlertBox.style.display = 'none';
    });

    if (btnAlertSnooze) {
      btnAlertSnooze.addEventListener('click', async () => {
        if (!lastActiveMatch || !lastActiveMatch.person) return;
        const name = lastActiveMatch.person.display_name;
        try {
          const res = await fetch('/api/snooze', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ person: name, duration_minutes: 60 })
          });
          const data = await res.json();
          if (data.success) {
            matchAlertBox.style.display = 'none';
            showToast(`⏳ '${name}' 1 saatliğine muaf tutuldu.`);
          }
        } catch (err) {
          console.error('Snooze error:', err);
        }
      });
    }

    // Auto-Skip Toggle
    autoSkipToggle.addEventListener('change', async (e) => {
      autoSkipEnabled = e.target.checked;
      try {
        const res = await fetch('/api/settings', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ auto_skip: autoSkipEnabled })
        });
        if (res.ok) {
          showToast(autoSkipEnabled ? '⚡ Otomatik Geçiş [AÇIK] ve config.json güncellendi' : '🔔 Otomatik Geçiş [KAPALI] ve config.json güncellendi');
        }
      } catch (err) {
        console.error('Settings update error:', err);
      }
    });

    // Search input
    let searchTimeout;
    searchInput.addEventListener('input', (e) => {
      clearTimeout(searchTimeout);
      currentSearchQuery = e.target.value.trim();
      searchTimeout = setTimeout(() => {
        loadPersons();
      }, 300);
    });

    // API Sync
    btnSync.addEventListener('click', async () => {
      btnSync.disabled = true;
      btnSyncText.textContent = 'Senkronize Ediliyor...';
      try {
        const res = await fetch('/api/sync', { method: 'POST' });
        const data = await res.json();
        if (data.success) {
          btnSyncText.textContent = 'Başarılı!';
          loadPersons();
          setTimeout(() => {
            btnSyncText.textContent = 'API Senkronize Et';
            btnSync.disabled = false;
          }, 2000);
        } else {
          alert('Sync Hatası: ' + (data.error || 'Bilinmeyen hata'));
          btnSync.disabled = false;
        }
      } catch (err) {
        alert('Sync bağlantı hatası: ' + err.message);
        btnSync.disabled = false;
      }
    });

    // Alert Quick Actions
    if (btnAlertAddAlias) {
      btnAlertAddAlias.addEventListener('click', () => {
        if (lastActiveMatch && lastActiveMatch.person) {
          openAliasModal(lastActiveMatch.person.display_name);
        }
      });
    }

    if (btnAlertAddExclude) {
      btnAlertAddExclude.addEventListener('click', () => {
        if (lastActiveMatch && lastActiveMatch.person) {
          quickAddExclude(lastActiveMatch.person.display_name);
        }
      });
    }

    // Modal Events
    if (btnModalClose) btnModalClose.addEventListener('click', closeAliasModal);
    if (btnModalCancel) btnModalCancel.addEventListener('click', closeAliasModal);
    if (btnModalSave) btnModalSave.addEventListener('click', saveModalAlias);

    if (aliasModal) {
      aliasModal.addEventListener('click', (e) => {
        if (e.target === aliasModal) closeAliasModal();
      });
    }

    if (modalAliasInput) {
      modalAliasInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          saveModalAlias();
        } else if (e.key === 'Escape') {
          closeAliasModal();
        }
      });
    }

    // Rules I/O Events (Export / Import)
    if (btnExportRules) {
      btnExportRules.addEventListener('click', () => {
        window.open('/api/rules/export', '_blank');
      });
    }

    if (btnImportRules && rulesFileInput) {
      btnImportRules.addEventListener('click', () => rulesFileInput.click());
      rulesFileInput.addEventListener('change', (e) => {
        const file = e.target.files[0];
        if (!file) return;
        const reader = new FileReader();
        reader.onload = async (evt) => {
          try {
            const jsonContent = JSON.parse(evt.target.result);
            const res = await fetch('/api/rules/import', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify(jsonContent)
            });
            const data = await res.json();
            if (res.ok && data.success) {
              showToast('✅ Kurallar başarıyla içe aktarıldı!');
              loadRules();
              loadPersons();
            } else {
              alert('Yükleme hatası: ' + (data.error || 'Bilinmeyen hata'));
            }
          } catch (err) {
            alert('JSON dosyası okunamadı: ' + err.message);
          }
        };
        reader.readAsText(file);
        rulesFileInput.value = '';
      });
    }

    // Settings Modal Handlers
    function openSettingsModal() {
      fetch('/api/config')
        .then(res => res.json())
        .then(cfg => {
          if (!cfg) return;
          if (cfgAutoSkip) cfgAutoSkip.checked = !!cfg.auto_skip;
          if (cfgEnableDialog) cfgEnableDialog.checked = !!cfg.enable_dialog;
          if (cfgFuzzyMatching) cfgFuzzyMatching.checked = cfg.fuzzy_matching !== false;
          const th = cfg.fuzzy_threshold ? Math.round(cfg.fuzzy_threshold * 100) : 85;
          if (cfgFuzzyThreshold) cfgFuzzyThreshold.value = th;
          if (fuzzyThresholdVal) fuzzyThresholdVal.textContent = `%${th}`;
          if (cfgPollInterval) cfgPollInterval.value = cfg.poll_interval_ms || 1500;
          if (cfgCategory) cfgCategory.value = cfg.category || 'HEPSİ';
          if (cfgWebPassword) cfgWebPassword.value = cfg.web_password || '';
          if (cfgWebPort) cfgWebPort.value = cfg.web_port || 8080;
          if (cfgApiUrl) cfgApiUrl.value = cfg.api_url || '';
          if (cfgAutoSyncHours) cfgAutoSyncHours.value = cfg.auto_sync_hours !== undefined ? cfg.auto_sync_hours : 12;

          if (settingsModal) settingsModal.style.display = 'flex';
        })
        .catch(err => console.error('Config fetch failed:', err));
    }

    function closeSettingsModal() {
      if (settingsModal) settingsModal.style.display = 'none';
    }

    if (btnOpenSettings) btnOpenSettings.addEventListener('click', openSettingsModal);
    if (btnSettingsClose) btnSettingsClose.addEventListener('click', closeSettingsModal);
    if (btnSettingsCancel) btnSettingsCancel.addEventListener('click', closeSettingsModal);
    if (settingsModal) {
      settingsModal.addEventListener('click', (e) => {
        if (e.target === settingsModal) closeSettingsModal();
      });
    }
    if (cfgFuzzyThreshold && fuzzyThresholdVal) {
      cfgFuzzyThreshold.addEventListener('input', () => {
        fuzzyThresholdVal.textContent = `%${cfgFuzzyThreshold.value}`;
      });
    }
    if (settingsForm) {
      settingsForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const payload = {
          auto_skip: cfgAutoSkip ? cfgAutoSkip.checked : false,
          enable_dialog: cfgEnableDialog ? cfgEnableDialog.checked : true,
          fuzzy_matching: cfgFuzzyMatching ? cfgFuzzyMatching.checked : true,
          fuzzy_threshold: cfgFuzzyThreshold ? (parseFloat(cfgFuzzyThreshold.value) / 100) : 0.85,
          poll_interval_ms: cfgPollInterval ? parseInt(cfgPollInterval.value) : 1500,
          category: cfgCategory ? cfgCategory.value : 'HEPSİ',
          web_password: cfgWebPassword ? cfgWebPassword.value : 'admin',
          web_port: cfgWebPort ? parseInt(cfgWebPort.value) : 8080,
          api_url: cfgApiUrl ? cfgApiUrl.value.trim() : '',
          auto_sync_hours: cfgAutoSyncHours ? parseInt(cfgAutoSyncHours.value) : 12,
          bind_address: '127.0.0.1'
        };

        try {
          const res = await fetch('/api/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
          });
          const data = await res.json();
          if (res.ok && data.success) {
            closeSettingsModal();
            showToast('💾 Ayarlar başarıyla config.json dosyasına kaydedildi!');
            autoSkipToggle.checked = payload.auto_skip;
            autoSkipEnabled = payload.auto_skip;
          } else {
            alert('Ayar kaydetme hatası: ' + (data.error || 'Bilinmeyen hata'));
          }
        } catch (err) {
          alert('Bağlantı hatası: ' + err.message);
        }
      });
    }

    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape' && aliasModal && aliasModal.style.display !== 'none') {
        closeAliasModal();
      }
    });

    // Delegated clicks for Person List (+ Alias and 🛡️ Muaf)
    if (personsList) {
      personsList.addEventListener('click', (e) => {
        const aliasBtn = e.target.closest('.btn-add-alias');
        if (aliasBtn) {
          const name = aliasBtn.getAttribute('data-name');
          openAliasModal(name);
          return;
        }

        const excludeBtn = e.target.closest('.btn-add-exclude');
        if (excludeBtn) {
          const name = excludeBtn.getAttribute('data-name');
          quickAddExclude(name);
          return;
        }
      });
    }
  }

  async function updateSettingsFilters() {
    try {
      await fetch('/api/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          active_categories: [selectedCategory],
          active_topics: [selectedTopic]
        })
      });
    } catch (err) {
      console.error('Filter update error:', err);
    }
  }

  async function triggerSkip() {
    btnSkipManual.disabled = true;
    btnAlertSkip.disabled = true;
    try {
      const res = await fetch('/api/skip', { method: 'POST' });
      const data = await res.json();
      if (data.success) {
        matchAlertBox.style.display = 'none';
      }
    } catch (err) {
      console.error('Skip failed:', err);
    } finally {
      setTimeout(() => {
        btnSkipManual.disabled = false;
        btnAlertSkip.disabled = false;
      }, 1000);
    }
  }

  let eventSource = null;
  function setupSSE() {
    if (!window.EventSource) return;
    if (eventSource) {
      try { eventSource.close(); } catch(e) {}
      eventSource = null;
    }

    try {
      eventSource = new EventSource('/api/events');
      eventSource.addEventListener('state', (e) => {
        try {
          const state = JSON.parse(e.data);
          applyState(state);
        } catch(err) {
          console.error("SSE parse error", err);
        }
      });

      eventSource.onopen = () => {
        if (connStatus) {
          connStatus.textContent = 'Canlı (0ms SSE Aktif)';
          connStatus.style.color = '#38ef7d';
        }
      };

      eventSource.onerror = (err) => {
        if (connStatus) {
          connStatus.textContent = 'Bağlanıyor...';
          connStatus.style.color = '#f5af19';
        }
      };
    } catch(e) {
      console.warn('SSE not initialized', e);
    }
  }

  function applyState(state) {
    if (!state) return;
    if (connStatus && (!eventSource || eventSource.readyState !== EventSource.OPEN)) {
      connStatus.textContent = 'Canlı İzleniyor';
      connStatus.style.color = '#8c909e';
    }

    // Update Player Card
    const media = state.current_media;
    if (media && (media.artist || media.title)) {
      trackTitle.textContent = media.title || 'İsimsiz Medya';
      trackArtist.textContent = media.artist || (media.app || 'Bilinmiyor');
      
      let albumStr = media.album ? `Albüm: ${media.album}` : (media.url || '');
      if (media.contributors) {
        albumStr = (albumStr ? albumStr + ' • ' : '') + `Katkıda Bulunanlar: ${media.contributors}`;
      }
      trackAlbum.textContent = albumStr;
      
      playerSourceBadge.textContent = `${media.app || 'MEDYA'} (${(media.state || '').toUpperCase()})`;
      playerSourceBadge.className = 'badge ' + (media.state === 'playing' ? 'playing' : '');
      btnSkipManual.disabled = false;

      if (btnPlayPause) {
        btnPlayPause.disabled = false;
        if (media.state === 'playing') {
          if (playPauseIcon) playPauseIcon.textContent = '⏸️';
          if (playPauseText) playPauseText.textContent = 'Duraklat';
        } else {
          if (playPauseIcon) playPauseIcon.textContent = '▶️';
          if (playPauseText) playPauseText.textContent = 'Oynat';
        }
      }
    } else {
      trackTitle.textContent = 'Medya Algılanmadı';
      trackArtist.textContent = 'Spotify, Apple Music, YouTube veya Tarayıcı açın';
      trackAlbum.textContent = '';
      playerSourceBadge.textContent = 'DURDURULDU';
      playerSourceBadge.className = 'badge';
      btnSkipManual.disabled = true;
      if (btnPlayPause) btnPlayPause.disabled = true;
    }

    // Update Auto-skip toggle sync
    if (autoSkipToggle && autoSkipToggle.checked !== state.auto_skip) {
      autoSkipToggle.checked = state.auto_skip;
    }

    // Render Analytics Stats
    if (state.stats) {
      if (statTotalDetections) statTotalDetections.textContent = state.stats.total_detections || 0;
      if (statAutoSkips) statAutoSkips.textContent = state.stats.total_auto_skips || 0;
      if (statManualSkips) statManualSkips.textContent = state.stats.total_manual_skips || 0;
      if (statTopPerson) {
        let topName = '-';
        let maxCount = 0;
        if (state.stats.top_persons) {
          for (const [name, count] of Object.entries(state.stats.top_persons)) {
            if (count > maxCount) {
              maxCount = count;
              topName = `${name} (${count})`;
            }
          }
        }
        statTopPerson.textContent = topName;
        statTopPerson.title = topName;
      }
    }

    // Match Alert Box
    const match = state.current_match;
    if (match && match.matched) {
      lastActiveMatch = match;
      matchAlertBox.style.display = 'block';
      alertMatchName.textContent = match.person.display_name;
      alertMatchTags.textContent = (match.person.tags || []).join(', ') || '-';
      alertMatchTopics.textContent = (match.topics || []).join(', ') || '-';
      alertMatchCategory.textContent = match.category.toUpperCase();
      playerSourceBadge.className = 'badge warn';
    } else {
      matchAlertBox.style.display = 'none';
    }

    // History Render
    renderHistory(state.history || []);
  }

  async function pollStatus() {
    try {
      const res = await fetch('/api/status');
      if (!res.ok) throw new Error('Status error');
      const state = await res.json();
      applyState(state);
    } catch (err) {
      if (!eventSource || eventSource.readyState !== EventSource.OPEN) {
        connStatus.textContent = 'Bağlantı Kesildi';
        connStatus.style.color = '#f87171';
      }
    }
  }

  async function loadPersons() {
    try {
      const url = new URL('/api/persons', window.location.origin);
      if (currentSearchQuery) url.searchParams.set('q', currentSearchQuery);
      if (selectedCategory && selectedCategory !== 'HEPSİ') url.searchParams.set('category', selectedCategory);
      if (selectedTopic && selectedTopic !== 'TÜM KONULAR') url.searchParams.set('topic', selectedTopic);

      const res = await fetch(url);
      const persons = await res.json();

      personCountBadge.textContent = `${persons.length} KİŞİ`;

      // Update autocomplete datalist if available
      const datalist = document.getElementById('allPersonsDatalist');
      if (datalist && (!currentSearchQuery && (!selectedCategory || selectedCategory === 'HEPSİ') && (!selectedTopic || selectedTopic === 'TÜM KONULAR'))) {
        datalist.innerHTML = persons.map(p => `<option value="${escapeHTML(p.display_name)}"></option>`).join('');
      }

      if (persons.length === 0) {
        personsList.innerHTML = '<div class="empty-state">Eşleşen kişi bulunamadı.</div>';
        return;
      }

      personsList.innerHTML = persons.map(p => `
        <div class="person-item">
          <div class="person-info">
            <span class="person-name">${escapeHTML(p.display_name)}</span>
            <span class="person-topics">${escapeHTML((p.topics || []).join(' • ') || 'Konu belirtilmemiş')}</span>
          </div>
          <div class="person-side">
            <div class="person-tags">
              ${(p.tags || []).map(t => `<span class="tag-pill">${escapeHTML(t)}</span>`).join('')}
            </div>
            <div class="person-actions">
              <button class="btn-item-action btn-add-alias" data-name="${escapeHTML(p.display_name)}" title="Bu kişiye alias (mahlas) tanımla">
                + Alias
              </button>
              <button class="btn-item-action btn-add-exclude" data-name="${escapeHTML(p.display_name)}" title="Bu kişiyi beyaz listeye ekle (muaf tut)">
                🛡️ Muaf
              </button>
            </div>
          </div>
        </div>
      `).join('');

    } catch (err) {
      console.error('Failed to load persons:', err);
    }
  }

  function renderHistory(items) {
    if (!items || items.length === 0) {
      historyList.innerHTML = '<div class="empty-state">Henüz bir tespit geçmişi yok.</div>';
      return;
    }

    historyList.innerHTML = items.slice(0, 15).map(item => {
      let actionLabel = 'Dinlendi';
      let actionClass = 'kept';
      if (item.action === 'skipped') {
        actionLabel = 'Geçildi (Onay)';
        actionClass = 'skipped';
      } else if (item.action === 'auto_skipped') {
        actionLabel = 'Otomatik Geçildi';
        actionClass = 'skipped';
      } else if (item.action === 'manual_skip') {
        actionLabel = 'Manuel Geçildi';
        actionClass = 'skipped';
      } else if (item.action === 'snoozed') {
        actionLabel = '1 Saat Ertelendi';
        actionClass = 'snoozed';
      } else if (item.action === 'timeout') {
        actionLabel = 'Zaman Aşımı';
        actionClass = 'kept';
      }

      return `
        <div class="history-item ${actionClass}">
          <div>
            <strong>${escapeHTML(item.person.display_name)}</strong>
            <span style="color: var(--text-secondary); margin-left: 6px;">(${escapeHTML(item.track.artist)} - ${escapeHTML(item.track.title)})</span>
          </div>
          <div style="font-family: var(--font-mono); font-size: 0.72rem; color: var(--text-muted);">
            <span style="margin-right: 8px;">${item.timestamp}</span>
            <span class="badge ${actionClass === 'skipped' ? 'warn' : ''}">${actionLabel}</span>
          </div>
        </div>
      `;
    }).join('');
  }

  function setupRulesManagement() {
    // Tab switching
    document.querySelectorAll('.rule-tab-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        document.querySelectorAll('.rule-tab-btn').forEach(b => b.classList.remove('active'));
        document.querySelectorAll('.rule-tab-pane').forEach(p => p.style.display = 'none');

        btn.classList.add('active');
        const targetId = btn.getAttribute('data-tab');
        const targetPane = document.getElementById(targetId);
        if (targetPane) targetPane.style.display = 'block';
      });
    });

    // Form 1: Add Custom Person
    const formAddPerson = document.getElementById('formAddPerson');
    if (formAddPerson) {
      formAddPerson.addEventListener('submit', async (e) => {
        e.preventDefault();
        const name = document.getElementById('customPersonName').value.trim();
        const tag = document.getElementById('customPersonTag').value;
        const topic = document.getElementById('customPersonTopic').value.trim() || 'Özel Kara Liste';
        const rawAliases = document.getElementById('customPersonAliases').value.trim();
        const aliases = rawAliases ? rawAliases.split(',').map(s => s.trim()).filter(Boolean) : [];

        try {
          const res = await fetch('/api/rules/person', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, tags: [tag], topics: [topic], aliases })
          });
          const data = await res.json();
          if (res.ok) {
            formAddPerson.reset();
            loadRules();
            loadPersons();
          } else {
            alert('Hata: ' + (data.error || 'Eklenemedi'));
          }
        } catch (err) {
          alert('Bağlantı hatası: ' + err.message);
        }
      });
    }

    // Form 2: Add Alias
    const formAddAlias = document.getElementById('formAddAlias');
    if (formAddAlias) {
      formAddAlias.addEventListener('submit', async (e) => {
        e.preventDefault();
        const alias = document.getElementById('aliasName').value.trim();
        const target = document.getElementById('aliasTarget').value.trim();

        try {
          const res = await fetch('/api/rules/alias', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ alias, target })
          });
          const data = await res.json();
          if (res.ok) {
            formAddAlias.reset();
            loadRules();
          } else {
            alert('Hata: ' + (data.error || 'Kaydedilemedi'));
          }
        } catch (err) {
          alert('Bağlantı hatası: ' + err.message);
        }
      });
    }

    // Form 3: Add Exclude
    const formAddExclude = document.getElementById('formAddExclude');
    if (formAddExclude) {
      formAddExclude.addEventListener('submit', async (e) => {
        e.preventDefault();
        const name = document.getElementById('excludeName').value.trim();

        try {
          const res = await fetch('/api/rules/exclude', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name })
          });
          const data = await res.json();
          if (res.ok) {
            formAddExclude.reset();
            loadRules();
          } else {
            alert('Hata: ' + (data.error || 'Eklenemedi'));
          }
        } catch (err) {
          alert('Bağlantı hatası: ' + err.message);
        }
      });
    }
  }

  async function loadRules() {
    try {
      const res = await fetch('/api/rules');
      if (!res.ok) return;
      const rules = await res.json();
      renderCustomRules(rules);
    } catch (err) {
      console.error('Failed to load rules:', err);
    }
  }

  function renderCustomRules(rules) {
    if (!rules) return;

    // Render Custom Persons
    const customPersonsList = document.getElementById('customPersonsList');
    if (customPersonsList) {
      const list = rules.custom_persons || [];
      if (list.length === 0) {
        customPersonsList.innerHTML = '<div class="empty-state" style="padding: 10px;">Henüz özel kişi eklenmedi.</div>';
      } else {
        customPersonsList.innerHTML = list.map(p => `
          <div class="custom-item">
            <div>
              <span class="custom-item-name">${escapeHTML(p.display_name)}</span>
              <span class="custom-item-meta">${escapeHTML((p.tags || []).join(', '))} • ${escapeHTML((p.topics || []).join(', '))}</span>
            </div>
            <button class="btn-icon-del" onclick="deleteCustomPerson('${p.id}')" title="Sil">✕</button>
          </div>
        `).join('');
      }
    }

    // Render Aliases
    const customAliasesList = document.getElementById('customAliasesList');
    if (customAliasesList) {
      const aliases = rules.aliases || {};
      const entries = Object.entries(aliases);
      if (entries.length === 0) {
        customAliasesList.innerHTML = '<div class="empty-state" style="padding: 10px;">Henüz özel alias tanımlanmadı.</div>';
      } else {
        customAliasesList.innerHTML = entries.map(([alias, target]) => `
          <div class="custom-item">
            <div>
              <span class="custom-item-name">${escapeHTML(alias)}</span>
              <span class="custom-item-meta">➔ ${escapeHTML(target)}</span>
            </div>
            <button class="btn-icon-del" onclick="deleteCustomAlias('${escapeHTML(alias)}')" title="Sil">✕</button>
          </div>
        `).join('');
      }
    }

    // Render Exclude List
    const customExcludeList = document.getElementById('customExcludeList');
    if (customExcludeList) {
      const excludes = rules.exclude_list || [];
      if (excludes.length === 0) {
        customExcludeList.innerHTML = '<div class="empty-state" style="padding: 10px;">Beyaz listede kimse yok.</div>';
      } else {
        customExcludeList.innerHTML = excludes.map(name => `
          <div class="custom-item">
            <div>
              <span class="custom-item-name">🛡️ ${escapeHTML(name)}</span>
              <span class="custom-item-meta">(Muaf)</span>
            </div>
            <button class="btn-icon-del" onclick="deleteCustomExclude('${escapeHTML(name)}')" title="Sil">✕</button>
          </div>
        `).join('');
      }
    }
  }

  // Modal Functions
  function openAliasModal(targetName) {
    if (!targetName) return;
    currentModalTarget = targetName;
    if (modalTargetName) modalTargetName.textContent = targetName;
    if (modalAliasInput) modalAliasInput.value = '';
    if (aliasModal) {
      aliasModal.style.display = 'flex';
      setTimeout(() => {
        if (modalAliasInput) modalAliasInput.focus();
      }, 50);
    }
  }

  function closeAliasModal() {
    if (aliasModal) aliasModal.style.display = 'none';
    currentModalTarget = '';
    if (modalAliasInput) modalAliasInput.value = '';
  }

  async function saveModalAlias() {
    if (!modalAliasInput) return;
    const alias = modalAliasInput.value.trim();
    if (!alias) {
      showToast('Lütfen geçerli bir takma ad / mahlas girin.', true);
      modalAliasInput.focus();
      return;
    }
    if (!currentModalTarget) return;

    try {
      const res = await fetch('/api/rules/alias', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ alias, target: currentModalTarget })
      });
      const data = await res.json();
      if (res.ok) {
        closeAliasModal();
        showToast(`✅ "${alias}" ➔ "${currentModalTarget}" aliası kaydedildi!`);
        loadRules();
        loadPersons();
      } else {
        showToast('Hata: ' + (data.error || 'Kaydedilemedi'), true);
      }
    } catch (err) {
      showToast('Bağlantı hatası: ' + err.message, true);
    }
  }

  async function quickAddExclude(name) {
    if (!name) return;
    if (!confirm(`"${name}" beyaz listeye (exclude list) eklensin mi?\n\nBu sanatçı çalsa bile uyarı verilmeyecek ve atlanmayacaktır.`)) {
      return;
    }

    try {
      const res = await fetch('/api/rules/exclude', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
      });
      const data = await res.json();
      if (res.ok) {
        showToast(`🛡️ "${name}" beyaz listeye eklendi.`);
        loadRules();
      } else {
        showToast('Hata: ' + (data.error || 'Eklenemedi'), true);
      }
    } catch (err) {
      showToast('Bağlantı hatası: ' + err.message, true);
    }
  }

  // Toast Function
  function showToast(message, isError = false) {
    const container = document.getElementById('toastContainer');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = 'toast' + (isError ? ' toast-error' : '');
    toast.textContent = message;

    container.appendChild(toast);

    setTimeout(() => {
      toast.style.animation = 'toastOut 0.25s ease-out forwards';
      setTimeout(() => toast.remove(), 250);
    }, 3200);
  }

  // Global window functions for inline delete onclick
  window.deleteCustomPerson = async (id) => {
    if (!confirm('Bu kişiyi özel listeden silmek istiyor musunuz?')) return;
    try {
      await fetch('/api/rules/person', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id })
      });
      showToast('🗑️ Özel kişi silindi.');
      loadRules();
      loadPersons();
    } catch (err) {
      console.error(err);
    }
  };

  window.deleteCustomAlias = async (alias) => {
    try {
      await fetch('/api/rules/alias', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ alias })
      });
      showToast(`🗑️ "${alias}" aliası silindi.`);
      loadRules();
    } catch (err) {
      console.error(err);
    }
  };

  window.deleteCustomExclude = async (name) => {
    try {
      await fetch('/api/rules/exclude', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
      });
      showToast(`🗑️ "${name}" beyaz listeden çıkarıldı.`);
      loadRules();
    } catch (err) {
      console.error(err);
    }
  };

  function escapeHTML(str) {
    if (!str) return '';
    return str.replace(/[&<>'"]/g, tag => ({
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      "'": '&#39;',
      '"': '&quot;'
    }[tag] || tag));
  }
});
