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
  loadPersons();
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
          <div class="person-tags">
            ${(p.tags || []).map(t => `<span class="tag-pill">${escapeHTML(t)}</span>`).join('')}
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
