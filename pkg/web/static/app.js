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
  const connStatus = document.getElementById('connStatus');
  const btnSync = document.getElementById('btnSync');
  const btnSyncText = document.getElementById('btnSyncText');

  // Alert Box Elements
  const matchAlertBox = document.getElementById('matchAlertBox');
  const alertMatchName = document.getElementById('alertMatchName');
  const alertMatchTags = document.getElementById('alertMatchTags');
  const alertMatchTopics = document.getElementById('alertMatchTopics');
  const alertMatchCategory = document.getElementById('alertMatchCategory');
  const btnAlertSkip = document.getElementById('btnAlertSkip');
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

  let currentModalTarget = '';
  let lastActiveMatch = null;

  // Person List & Search Elements
  const searchInput = document.getElementById('searchInput');
  const personsList = document.getElementById('personsList');
  const personCountBadge = document.getElementById('personCountBadge');
  const historyList = document.getElementById('historyList');

  // Category & Topic Grids
  const categoriesGrid = document.getElementById('categoriesGrid');
  const topicsGrid = document.getElementById('topicsGrid');

  // Initialize
  setupFilterButtons();
  setupEventListeners();
  setupRulesManagement();
  loadPersons();
  loadRules();
  pollStatus();
  setInterval(pollStatus, 1500);

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

    btnAlertKeep.addEventListener('click', () => {
      matchAlertBox.style.display = 'none';
    });

    // Auto-Skip Toggle
    autoSkipToggle.addEventListener('change', async (e) => {
      autoSkipEnabled = e.target.checked;
      try {
        await fetch('/api/settings', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ auto_skip: autoSkipEnabled })
        });
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

  async function pollStatus() {
    try {
      const res = await fetch('/api/status');
      if (!res.ok) throw new Error('Status error');
      const state = await res.json();

      connStatus.textContent = 'Canlı İzleniyor';
      connStatus.style.color = '#8c909e';

      // Update Player Card
      const media = state.current_media;
      if (media && (media.artist || media.title)) {
        trackTitle.textContent = media.title || 'İsimsiz Medya';
        trackArtist.textContent = media.artist || (media.app || 'Bilinmiyor');
        trackAlbum.textContent = media.album ? `Albüm: ${media.album}` : (media.url || '');
        
        playerSourceBadge.textContent = `${media.app || 'MEDYA'} (${media.state.toUpperCase()})`;
        playerSourceBadge.className = 'badge ' + (media.state === 'playing' ? 'playing' : '');
        btnSkipManual.disabled = false;
      } else {
        trackTitle.textContent = 'Medya Algılanmadı';
        trackArtist.textContent = 'Spotify, Apple Music veya YouTube açın';
        trackAlbum.textContent = '';
        playerSourceBadge.textContent = 'DURDURULDU';
        playerSourceBadge.className = 'badge';
        btnSkipManual.disabled = true;
      }

      // Update Auto-skip toggle sync
      if (autoSkipToggle.checked !== state.auto_skip) {
        autoSkipToggle.checked = state.auto_skip;
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

    } catch (err) {
      connStatus.textContent = 'Bağlantı Kesildi';
      connStatus.style.color = '#f87171';
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
