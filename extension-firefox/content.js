// OE Watcher Chrome Extension - Content Script
(function () {
  'use strict';

  let config = {
    enabled: true,
    autoSkip: true,
    persons: [],
    aliases: {},
    excludeList: [],
    keywords: []
  };

  let compiledMatchers = [];
  let shieldedCount = 0;
  let scanTimer = null;
  let mediaWatcherInitialized = false;

  // Targeted block and card selectors for exact in-card shielding
  const BLOCK_SELECTORS = [
    // Twitter / X (UserCell on People search, tweet on feeds)
    '[data-testid="UserCell"]',
    'article[data-testid="tweet"]',
    '[data-testid="cellInnerDiv"]',
    '[data-testid="trend"]',
    // YouTube (Video, Channel, Mix, Shorts, Shelf, Lockup)
    'ytd-channel-renderer',
    'ytd-video-renderer',
    'ytd-rich-item-renderer',
    'ytd-compact-video-renderer',
    'ytd-grid-video-renderer',
    'ytd-radio-renderer',
    'ytd-shelf-renderer',
    'ytd-reel-item-renderer',
    'ytd-playlist-renderer',
    'ytd-movie-renderer',
    'ytd-show-renderer',
    'yt-lockup-view-model',
    // YouTube Music (Queue items, tracks, cards)
    'ytmusic-player-queue-item',
    'ytmusic-responsive-list-item-renderer',
    'ytmusic-two-row-item-renderer',
    'ytmusic-video-renderer',
    // TikTok
    '[data-e2e="recommend-list-item-container"]',
    '[data-e2e="search-card-container"]',
    // Instagram
    'article[role="presentation"]',
    'article._ab6-',
    // Reddit
    'shreddit-post',
    '[data-testid="post-container"]',
    // News / Forums / Common layouts
    'article',
    '.entry-item',
    '.news-item',
    '.media-item',
    '.feed-item',
    '.card',
    '.post',
    'li[data-id]'
  ];

  // Helper to attach rock-solid click/tap handling that stops host page event stealing
  function attachSafeAction(btn, actionFn) {
    if (!btn) return;
    ['pointerdown', 'mousedown', 'click'].forEach(evtName => {
      btn.addEventListener(evtName, (e) => {
        e.stopPropagation();
        e.stopImmediatePropagation();
        if (evtName === 'click') {
          e.preventDefault();
          actionFn(e);
        }
      }, true);
    });
  }

  // Turkish character folding for accurate case-insensitive matching
  function foldTurkish(str) {
    if (!str) return '';
    return str
      .replace(/İ/g, 'i')
      .replace(/I/g, 'ı')
      .replace(/Ç/g, 'ç')
      .replace(/Ğ/g, 'ğ')
      .replace(/Ö/g, 'ö')
      .replace(/Ş/g, 'ş')
      .replace(/Ü/g, 'ü')
      .toLowerCase();
  }

  // Extract base names and aliases from compound names like "Ezhel (Ömer Sercan İpekçioğlu)"
  function extractCandidateNames(rawName) {
    const list = new Set();
    const clean = (rawName || '').trim();
    if (!clean) return [];

    const parenMatch = clean.match(/^(.*?)\s*\((.*?)\)\s*$/);
    if (parenMatch) {
      const primary = parenMatch[1].trim();
      const secondary = parenMatch[2].trim();
      if (primary.length >= 3) list.add(primary);
      if (secondary.length >= 3) list.add(secondary);
    }
    if (clean.length >= 3) {
      list.add(clean);
    }
    return Array.from(list);
  }

  // URL normalization and whitelist checking
  function normalizeUrlForMatching(u) {
    if (!u) return '';
    let str = u.trim().toLowerCase();
    str = str.replace(/^https?:\/\//i, '');
    str = str.replace(/^www\./i, '');
    str = str.replace(/\/+$/, '');
    return str;
  }

  function isUrlExcluded(targetUrl, excludeEntry) {
    if (!targetUrl || !excludeEntry) return false;
    const normTarget = normalizeUrlForMatching(targetUrl);
    const normEx = normalizeUrlForMatching(excludeEntry);
    if (!normEx) return false;

    // Check if exclude entry is a URL or domain pattern
    const isUrlPattern = excludeEntry.includes('/') || excludeEntry.includes('.') || /^https?:\/\//i.test(excludeEntry);
    if (!isUrlPattern) return false;

    // Exact match
    if (normTarget === normEx) return true;

    // Subpath match (e.g. repo URL whitelists all issues, trees, PRs under it)
    if (normTarget.startsWith(normEx + '/') || normTarget.startsWith(normEx + '?') || normTarget.startsWith(normEx + '#')) {
      return true;
    }

    // Domain / subdomain match (e.g. 'github.com' whitelists all pages on github.com)
    try {
      const targetHost = (normTarget.split('/')[0]).split('?')[0].split('#')[0];
      const exHost = (normEx.split('/')[0]).split('?')[0].split('#')[0];
      if (!normEx.includes('/') && (targetHost === exHost || targetHost.endsWith('.' + exHost))) {
        return true;
      }
    } catch (_) {}

    return false;
  }

  function isCurrentPageExcluded() {
    if (!config.excludeList || config.excludeList.length === 0) return false;
    const currentUrl = window.location.href;
    for (const entry of config.excludeList) {
      if (isUrlExcluded(currentUrl, entry)) {
        return true;
      }
    }
    return false;
  }

  function cleanAllShields() {
    document.querySelectorAll('.oe-shield-banner, .oe-revert-tag').forEach(el => el.remove());
    document.querySelectorAll('.oe-shielded-container, [data-oe-shielded]').forEach(el => {
      el.classList.remove('oe-shielded-container');
      el.removeAttribute('data-oe-shielded');
      el.removeAttribute('data-oe-revealed');
      el.style.minHeight = '';
    });
    shieldedCount = 0;
  }

  // Load config & data from background service worker
  function init() {
    chrome.runtime.sendMessage({ type: 'GET_DATA' }, (response) => {
      if (!response) return;
      config.enabled = response.settings ? response.settings.enabled !== false : true;
      config.autoSkip = response.settings && response.settings.autoSkip !== undefined ? response.settings.autoSkip !== false : true;
      config.persons = response.persons || [];
      config.aliases = response.aliases || {};
      config.excludeList = response.excludeList || [];
      config.keywords = response.keywords || [];

      if (!config.enabled) {
        cleanAllShields();
        chrome.runtime.sendMessage({ type: 'UPDATE_BADGE', count: 0 });
        return;
      }

      if (isCurrentPageExcluded()) {
        cleanAllShields();
        chrome.runtime.sendMessage({ type: 'UPDATE_BADGE', count: 0, isExcluded: true });
        startObserver();
        return;
      }

      if (config.persons.length === 0 && (!config.keywords || config.keywords.length === 0)) {
        return;
      }

      compileMatchers();
      scanDocument();
      startObserver();

      if (!mediaWatcherInitialized) {
        mediaWatcherInitialized = true;
        initMediaWatcher();
      }
    });
  }

  // Compile regex matchers with word boundary guards
  function compileMatchers() {
    const excludeSet = new Set((config.excludeList || []).map(e => foldTurkish(e)));
    compiledMatchers = [];

    // 1. Direct persons
    for (const p of config.persons) {
      if (!p || !p.name) continue;
      const candidates = extractCandidateNames(p.name);

      for (const cand of candidates) {
        const folded = foldTurkish(cand);
        if (folded.length < 3 || excludeSet.has(folded)) continue;

        const escaped = folded.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const regex = new RegExp(`(?:^|[^a-zçğıöşü0-9])${escaped}(?:$|[^a-zçğıöşü0-9])`, 'i');

        compiledMatchers.push({
          displayName: p.name,
          matchedKeyword: cand,
          category: (p.tags && p.tags[0]) || 'Kara Liste',
          topics: p.topics || [],
          regex: regex
        });
      }
    }

    // 2. Custom Aliases (Alias -> Target)
    if (config.aliases) {
      for (const [alias, target] of Object.entries(config.aliases)) {
        const folded = foldTurkish(alias);
        if (folded.length < 3 || excludeSet.has(folded)) continue;
        const escaped = folded.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const regex = new RegExp(`(?:^|[^a-zçğıöşü0-9])${escaped}(?:$|[^a-zçğıöşü0-9])`, 'i');

        compiledMatchers.push({
          displayName: `${target} (Alias: ${alias})`,
          matchedKeyword: alias,
          category: 'Özel Tanımlı',
          topics: ['Alias Eşleşmesi'],
          regex: regex
        });
      }
    }

    // 3. Custom Keywords / Hashtags
    if (config.keywords) {
      for (const kw of config.keywords) {
        const cleanKw = (kw || '').trim();
        const folded = foldTurkish(cleanKw);
        if (folded.length < 2 || excludeSet.has(folded)) continue;
        const escaped = folded.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const regex = new RegExp(`(?:^|[^a-zçğıöşü0-9])${escaped}(?:$|[^a-zçğıöşü0-9])`, 'i');

        compiledMatchers.push({
          displayName: cleanKw,
          matchedKeyword: cleanKw,
          category: 'Özel Kelime',
          topics: ['Engellenen Kelime / Tag'],
          regex: regex
        });
      }
    }
  }

  // Fast text matching against compiled blacklist matchers
  function checkTextMatch(text) {
    if (!text || typeof text !== 'string') return null;
    const clean = text.trim();
    if (clean.length < 2) return null;
    const folded = foldTurkish(clean);
    for (const m of compiledMatchers) {
      if (m.regex && m.regex.test(folded)) {
        return m;
      }
    }
    return null;
  }

  // Toast notification for auto-skipped tracks
  function showSkipToast(personName, songTitle) {
    try {
      let existing = document.querySelector('.oe-skip-toast');
      if (existing) existing.remove();

      const toast = document.createElement('div');
      toast.className = 'oe-skip-toast';
      toast.innerHTML = `
        <div class="oe-skip-toast-icon">⏩</div>
        <div class="oe-skip-toast-content">
          <div class="oe-skip-toast-badge">🛡️ OE WATCHER // OTOMATİK GEÇİLDİ</div>
          <div class="oe-skip-toast-title">Kara listedeki <strong>${escapeHTML(personName)}</strong> atlandı</div>
          <div class="oe-skip-toast-desc">${escapeHTML(songTitle)}</div>
        </div>
      `;
      document.body.appendChild(toast);

      setTimeout(() => {
        toast.classList.add('oe-fade-out');
        setTimeout(() => toast.remove(), 400);
      }, 3500);
    } catch (_) {}
  }

  // Check if an element belongs to the active media player bar / playback controls
  function isMediaControllerElement(elem) {
    if (!elem) return false;
    return !!(
      elem.closest &&
      elem.closest('ytmusic-player-bar, .ytp-chrome-bottom, #movie_player, .html5-video-player')
    );
  }

  // Clean all shield artifacts specifically from live media player bars
  function cleanPlayerBarShields() {
    const playerBars = document.querySelectorAll('ytmusic-player-bar, #movie_player, .ytp-chrome-bottom');
    playerBars.forEach(pb => {
      pb.querySelectorAll('.oe-shield-banner, .oe-revert-tag, .oe-player-shield').forEach(el => el.remove());
      pb.querySelectorAll('.oe-shielded-container, [data-oe-shielded], [data-oe-revealed], [data-oe-dismissed]').forEach(el => {
        el.classList.remove('oe-shielded-container', 'oe-revealed-container', 'oe-dismissed-container');
        el.removeAttribute('data-oe-shielded');
        el.removeAttribute('data-oe-revealed');
        el.removeAttribute('data-oe-dismissed');
        el.removeAttribute('data-oe-matched-name');
        el.removeAttribute('data-oe-orig-pos');
        el.removeAttribute('data-oe-orig-minheight');
        el.style.minHeight = '';
        el.style.color = '';
        el.style.textShadow = '';
      });
    });
  }

  // Warning toast when autoSkip is disabled
  function showWarningToast(personName, songTitle) {
    try {
      let existing = document.querySelector('.oe-warn-toast');
      if (existing) {
        if (existing.getAttribute('data-track') === songTitle) return;
        existing.remove();
      }

      const toast = document.createElement('div');
      toast.className = 'oe-skip-toast oe-warn-toast';
      toast.setAttribute('data-track', songTitle);
      toast.innerHTML = `
        <div class="oe-skip-toast-icon">⚠️</div>
        <div class="oe-skip-toast-content">
          <div class="oe-skip-toast-badge" style="color: #fbbf24;">🛡️ OE WATCHER // MEDYA UYARISI</div>
          <div class="oe-skip-toast-title">Kara listedeki <strong>${escapeHTML(personName)}</strong> çalıyor</div>
          <div class="oe-skip-toast-desc">${escapeHTML(songTitle)}</div>
        </div>
        <div style="display: flex; gap: 6px; margin-left: 8px;">
          <button type="button" class="oe-btn-skip-now" style="background: #e11d48; color: #fff; border: none; border-radius: 6px; padding: 4px 10px; font-size: 0.75rem; font-weight: 600; cursor: pointer;">
            ⏩ Atla
          </button>
          <button type="button" class="oe-btn-dismiss-warn" style="background: rgba(255,255,255,0.15); color: #fff; border: none; border-radius: 6px; padding: 4px 8px; font-size: 0.75rem; cursor: pointer;">
            ✕
          </button>
        </div>
      `;

      const btnSkip = toast.querySelector('.oe-btn-skip-now');
      attachSafeAction(btnSkip, () => {
        const nextBtn = document.querySelector('ytmusic-player-bar .next-button, #right-controls .next-button, .next-button, .ytp-next-button, [data-testid="control-button-skip-forward"], .playControls__next');
        if (nextBtn) nextBtn.click();
        const video = document.querySelector('video');
        if (video) { try { video.currentTime = video.duration || 99999; } catch(_) {} }
        toast.remove();
      });

      const btnDismiss = toast.querySelector('.oe-btn-dismiss-warn');
      attachSafeAction(btnDismiss, () => {
        toast.remove();
      });

      document.body.appendChild(toast);
    } catch (_) {}
  }

  // Real-time YouTube Music & YouTube Player Monitoring & Auto-Skip
  function initMediaWatcher() {
    const hostname = window.location.hostname;
    const isYTMusic = hostname === 'music.youtube.com';
    const isYouTube = hostname.includes('youtube.com') && !isYTMusic;
    const isSpotify = hostname.includes('spotify.com');
    const isSoundCloud = hostname.includes('soundcloud.com');

    if (!isYTMusic && !isYouTube && !isSpotify && !isSoundCloud) return;

    let lastSkippedTrackKey = '';
    let isSkipping = false;

    function checkCurrentPlaying() {
      if (!config.enabled || isCurrentPageExcluded()) {
        cleanPlayerBarShields();
        return;
      }

      let title = '';
      let artist = '';

      if (isYTMusic) {
        const titleEl = document.querySelector('ytmusic-player-bar .title, .title.ytmusic-player-bar, yt-formatted-string.title');
        const bylineEl = document.querySelector('ytmusic-player-bar .byline, .byline.ytmusic-player-bar, yt-formatted-string.byline');
        title = (titleEl ? (titleEl.getAttribute('title') || titleEl.textContent) : '').trim();
        artist = (bylineEl ? (bylineEl.getAttribute('title') || bylineEl.textContent) : '').trim();
      } else if (isYouTube) {
        const titleEl = document.querySelector('h1.ytd-watch-metadata yt-formatted-string, #title h1, .ytp-title-link');
        const channelEl = document.querySelector('#channel-name yt-formatted-string, #owner ytd-channel-name');
        title = (titleEl ? (titleEl.getAttribute('title') || titleEl.textContent) : '').trim();
        artist = (channelEl ? (channelEl.getAttribute('title') || channelEl.textContent) : '').trim();
      } else if (isSpotify) {
        const titleEl = document.querySelector('[data-testid="context-item-info-title"] a, [data-testid="context-item-info-title"]');
        const artistEl = document.querySelector('[data-testid="context-item-info-artist"] a, [data-testid="context-item-info-artist"], [data-testid="context-item-info-subtitles"] a');
        title = (titleEl ? titleEl.textContent : '').trim();
        artist = (artistEl ? artistEl.textContent : '').trim();
      } else if (isSoundCloud) {
        const titleEl = document.querySelector('.playbackSoundBadge__titleLink, .playbackSoundBadge__title');
        const artistEl = document.querySelector('.playbackSoundBadge__lightLink');
        title = (titleEl ? (titleEl.getAttribute('title') || titleEl.textContent) : '').trim();
        artist = (artistEl ? (artistEl.getAttribute('title') || artistEl.textContent) : '').trim();
      }

      // MediaSession API fallback
      if ((!artist || !title) && navigator.mediaSession && navigator.mediaSession.metadata) {
        if (!title) title = navigator.mediaSession.metadata.title || '';
        if (!artist) artist = navigator.mediaSession.metadata.artist || '';
      }

      if (!title && !artist) return;

      const trackKey = `${artist}:::${title}`;
      if (trackKey === lastSkippedTrackKey && !isSkipping) return;

      const match = checkTextMatch(artist) || checkTextMatch(title) || checkTextMatch(`${artist} - ${title}`);
      if (match) {
        // Always ensure live player bar is not cluttered by card shields
        cleanPlayerBarShields();

        if (config.autoSkip) {
          if (isSkipping) return;
          isSkipping = true;
          lastSkippedTrackKey = trackKey;

          console.warn(`[OE Watcher] Kara liste tespit edildi: "${match.displayName}" (${artist} - ${title}). Şarkı atlanıyor...`);
          showSkipToast(match.displayName, title || artist);

          // 1. Click Next track button
          const nextBtn = document.querySelector(
            'ytmusic-player-bar .next-button, #right-controls .next-button, .next-button, .ytp-next-button, [data-testid="control-button-skip-forward"], .playControls__next'
          );
          if (nextBtn) {
            nextBtn.click();
          }

          // 2. Fast forward video element to guarantee transition
          const video = document.querySelector('video');
          if (video) {
            try {
              video.currentTime = video.duration || 99999;
            } catch (_) {}
          }

          // 3. Inform background worker to register in history
          chrome.runtime.sendMessage({
            type: 'MEDIA_SKIPPED',
            track: {
              app: isYTMusic ? 'Google Chrome (YouTube Music)' : 'Google Chrome (YouTube)',
              artist: artist,
              title: title,
              state: 'playing'
            },
            person: match.displayName,
            category: match.category,
            topics: match.topics
          });

          setTimeout(() => {
            isSkipping = false;
          }, 1000);
        } else {
          // Auto-skip disabled: show lightweight warning toast without breaking player UI
          showWarningToast(match.displayName, title || artist);
        }
      } else {
        lastSkippedTrackKey = trackKey;
        isSkipping = false;
        // Clean any leftover player bar shield and warning toast immediately
        cleanPlayerBarShields();
        const existingWarn = document.querySelector('.oe-warn-toast');
        if (existingWarn) existingWarn.remove();
      }
    }

    // Interval check every 400ms for fast reaction
    setInterval(checkCurrentPlaying, 400);

    // MutationObserver on body/playerBar for instant playback change reaction
    const observer = new MutationObserver(() => checkCurrentPlaying());
    observer.observe(document.body, { childList: true, subtree: true, characterData: true });
  }

  // Find the exact block card or container for modern web layouts
  function findBlockContainer(node) {
    if (!node) return null;
    let curr = node.nodeType === Node.ELEMENT_NODE ? node : node.parentElement;

    // Never card-shield live media player controls
    if (isMediaControllerElement(curr)) {
      return null;
    }

    while (curr && curr !== document.body && curr !== document.documentElement) {
      for (const sel of BLOCK_SELECTORS) {
        if (curr.matches && curr.matches(sel)) {
          return curr;
        }
      }

      // Fallback for standard text blocks
      const tag = curr.tagName.toLowerCase();
      if (tag === 'p' || tag === 'li' || tag === 'blockquote') {
        return curr;
      }

      curr = curr.parentElement;
    }

    if (node.nodeType === Node.ELEMENT_NODE) {
      if (isMediaControllerElement(node)) return null;
      return node;
    }
    if (node.parentElement && isMediaControllerElement(node.parentElement)) {
      return null;
    }
    return node.parentElement;
  }

  // Completely unshield a container and restore its original layout & interactivity
  function unshieldElement(container) {
    if (!container) return;

    const banner = container.querySelector(':scope > .oe-shield-banner, .oe-shield-banner');
    if (banner) banner.remove();
    const revertTag = container.querySelector(':scope > .oe-revert-tag, .oe-revert-tag');
    if (revertTag) revertTag.remove();

    container.removeAttribute('data-oe-shielded');
    container.removeAttribute('data-oe-revealed');
    container.removeAttribute('data-oe-dismissed');
    container.removeAttribute('data-oe-matched-name');
    container.classList.remove('oe-shielded-container', 'oe-revealed-container', 'oe-dismissed-container');

    if (container.getAttribute('data-oe-orig-pos') === 'static') {
      container.style.position = '';
      container.removeAttribute('data-oe-orig-pos');
    }
    if (container.getAttribute('data-oe-orig-minheight') === 'true') {
      container.style.minHeight = '';
      container.removeAttribute('data-oe-orig-minheight');
    }
    container.style.color = '';
    container.style.textShadow = '';
    container.style.maxHeight = '';
    container.style.height = '';
    container.style.overflow = '';
  }

  // Update existing shield banner info if matched person changed dynamically
  function updateShieldBanner(container, matchInfo) {
    const banner = container.querySelector(':scope > .oe-shield-banner, .oe-shield-banner');
    if (!banner) return;
    const titleEl = banner.querySelector('.oe-shield-title');
    if (titleEl) {
      const subDetails = matchInfo.category ? `[${matchInfo.category}]` : '';
      titleEl.innerHTML = `Bu içerikte kara listedeki <strong>${escapeHTML(matchInfo.displayName)}</strong> ${escapeHTML(subDetails)} geçiyor.`;
    }
    const descEl = banner.querySelector('.oe-shield-desc');
    if (descEl) {
      const topicsStr = matchInfo.topics && matchInfo.topics.length > 0 ? ` • ${matchInfo.topics.slice(0, 2).join(', ')}` : '';
      descEl.innerHTML = `Bu içeriği görmek istiyor musunuz?${escapeHTML(topicsStr)}`;
    }
  }

  // Check if a container element still has text or attribute matching blacklisted patterns,
  // strictly excluding any OE Watcher UI elements (banners, revert tags).
  function checkContainerForMatch(container) {
    if (!container || compiledMatchers.length === 0) return null;

    // 1. Check relevant attributes on elements inside container
    const attrNodes = container.querySelectorAll(
      'a[title], a[aria-label], yt-formatted-string[title], yt-formatted-string[aria-label], [data-testid="UserCell"] [aria-label], article [aria-label], img[alt]'
    );
    for (let i = 0; i < attrNodes.length; i++) {
      const el = attrNodes[i];
      if (el.closest('.oe-shield-banner') || el.closest('.oe-revert-tag') || el.closest('.oe-player-shield')) {
        continue;
      }
      for (const attr of ['title', 'aria-label', 'alt']) {
        const val = el.getAttribute(attr);
        if (val) {
          const m = checkTextMatch(val);
          if (m) return m;
        }
      }
    }

    // 2. Check text nodes inside container (excluding banner/revert-tag)
    const walker = document.createTreeWalker(
      container,
      NodeFilter.SHOW_TEXT,
      {
        acceptNode: function (node) {
          const parent = node.parentElement;
          if (!parent) return NodeFilter.FILTER_REJECT;
          if (
            parent.closest('.oe-shield-banner') ||
            parent.closest('.oe-revert-tag') ||
            parent.closest('.oe-player-shield')
          ) {
            return NodeFilter.FILTER_REJECT;
          }
          const tag = parent.tagName.toLowerCase();
          if (tag === 'script' || tag === 'style') {
            return NodeFilter.FILTER_REJECT;
          }
          if (node.nodeValue.trim().length < 3) {
            return NodeFilter.FILTER_SKIP;
          }
          return NodeFilter.FILTER_ACCEPT;
        }
      }
    );

    let textNode = walker.nextNode();
    while (textNode) {
      const m = checkTextMatch(textNode.nodeValue);
      if (m) return m;
      textNode = walker.nextNode();
    }

    return null;
  }

  // Dynamically re-evaluate all currently shielded or revealed containers:
  // If their content changed (e.g. track change, virtual scroll, SPA update) and is now safe,
  // AUTOMATICALLY unshield them!
  function recheckShieldedContainers() {
    // 1. Check all shielded containers
    const shielded = document.querySelectorAll('[data-oe-shielded]');
    shielded.forEach(container => {
      if (isMediaControllerElement(container)) {
        unshieldElement(container);
        return;
      }

      const match = checkContainerForMatch(container);
      if (!match) {
        // Content changed and no longer matches blacklist! Unshield it cleanly.
        unshieldElement(container);
      } else {
        // Content still matches. If the matched person changed, update banner info.
        const prevPerson = container.getAttribute('data-oe-matched-name');
        if (prevPerson && prevPerson !== match.displayName) {
          container.setAttribute('data-oe-matched-name', match.displayName);
          updateShieldBanner(container, match);
        }
      }
    });

    // 2. Check all revealed containers
    const revealed = document.querySelectorAll('[data-oe-revealed]');
    revealed.forEach(container => {
      if (isMediaControllerElement(container)) {
        unshieldElement(container);
        return;
      }

      const match = checkContainerForMatch(container);
      if (!match) {
        // Content changed and is safe! Clean up revealed state & revert tag.
        container.removeAttribute('data-oe-revealed');
        container.removeAttribute('data-oe-matched-name');
        container.classList.remove('oe-revealed-container');
        const tag = container.querySelector(':scope > .oe-revert-tag, .oe-revert-tag');
        if (tag) tag.remove();
      } else {
        const prevPerson = container.getAttribute('data-oe-matched-name');
        if (prevPerson && prevPerson !== match.displayName) {
          // Recycled to a different blacklisted person: re-shield with new person
          container.removeAttribute('data-oe-revealed');
          container.classList.remove('oe-revealed-container');
          const tag = container.querySelector(':scope > .oe-revert-tag, .oe-revert-tag');
          if (tag) tag.remove();
          shieldElement(container, match);
        }
      }
    });
  }

  // Apply shield directly inside container as an overlay without disturbing virtual scrollers
  function shieldElement(container, matchInfo) {
    if (!container || container.hasAttribute('data-oe-shielded') || container.hasAttribute('data-oe-revealed')) {
      return;
    }
    if (isMediaControllerElement(container)) {
      return;
    }

    container.setAttribute('data-oe-shielded', 'true');
    container.setAttribute('data-oe-matched-name', matchInfo.displayName || '');

    // Ensure container has relative positioning context and minimum height for overlay
    const computed = window.getComputedStyle(container);
    if (computed.position === 'static') {
      container.style.position = 'relative';
      container.setAttribute('data-oe-orig-pos', 'static');
    }
    if (container.offsetHeight < 110) {
      container.style.minHeight = '110px';
      container.setAttribute('data-oe-orig-minheight', 'true');
    }

    container.classList.add('oe-shielded-container');

    // Create overlay banner
    const banner = document.createElement('div');
    banner.className = 'oe-shield-banner';

    const subDetails = matchInfo.category ? `[${matchInfo.category}]` : '';
    const topicsStr = matchInfo.topics && matchInfo.topics.length > 0 ? ` • ${matchInfo.topics.slice(0, 2).join(', ')}` : '';

    banner.innerHTML = `
      <div class="oe-shield-header">
        <div class="oe-shield-badge">
          <span class="oe-shield-dot"></span>
          OE WATCHER // İÇERİK KALKANI
        </div>
      </div>
      <div class="oe-shield-body">
        <div class="oe-shield-icon">🛡️</div>
        <div class="oe-shield-info">
          <div class="oe-shield-title">
            Bu içerikte kara listedeki <strong>${escapeHTML(matchInfo.displayName)}</strong> ${escapeHTML(subDetails)} geçiyor.
          </div>
          <div class="oe-shield-desc">
            Bu içeriği görmek istiyor musunuz?${escapeHTML(topicsStr)}
          </div>
        </div>
      </div>
      <div class="oe-shield-actions">
        <button type="button" class="oe-btn oe-btn-reveal" title="İçeriği bu seferlik göster">
          👁️ İçeriği Göster
        </button>
        <button type="button" class="oe-btn oe-btn-dismiss" title="Bu içeriği tamamen gizle">
          ✕ Gizli Tut
        </button>
      </div>
    `;

    // Action: "İçeriği Göster" (Unmask content completely and remove shield overlay)
    const btnReveal = banner.querySelector('.oe-btn-reveal');
    function doReveal() {
      container.setAttribute('data-oe-revealed', 'true');
      container.setAttribute('data-oe-matched-name', matchInfo.displayName || '');
      container.removeAttribute('data-oe-shielded');
      container.classList.remove('oe-shielded-container');
      container.classList.add('oe-revealed-container');

      if (container.getAttribute('data-oe-orig-minheight') === 'true') {
        container.style.minHeight = '';
      }
      container.style.color = '';
      container.style.textShadow = '';

      banner.remove();

      // Non-intrusive re-hide button in corner
      const revertTag = document.createElement('div');
      revertTag.className = 'oe-revert-tag';
      revertTag.innerHTML = `
        <span>⚠️ [${escapeHTML(matchInfo.displayName)}] Gösterildi</span>
        <button type="button" class="oe-btn-rehide">Yeniden Gizle</button>
      `;
      const rehideBtn = revertTag.querySelector('.oe-btn-rehide');
      attachSafeAction(rehideBtn, () => {
        revertTag.remove();
        container.removeAttribute('data-oe-revealed');
        container.removeAttribute('data-oe-matched-name');
        container.classList.remove('oe-revealed-container');
        shieldElement(container, matchInfo);
      });

      container.appendChild(revertTag);
    }
    attachSafeAction(btnReveal, doReveal);

    // Action: "Gizli Tut" (Cleanly collapse container into a thin bar without breaking scroller heights)
    const btnDismiss = banner.querySelector('.oe-btn-dismiss');
    function doDismiss() {
      container.setAttribute('data-oe-dismissed', 'true');
      container.classList.remove('oe-shielded-container');
      container.classList.add('oe-dismissed-container');
      container.style.minHeight = '36px';
      container.style.maxHeight = '36px';
      container.style.height = '36px';
      container.style.overflow = 'hidden';

      banner.className = 'oe-shield-banner oe-dismissed-banner';
      banner.innerHTML = `
        <div style="display: flex; align-items: center; justify-content: space-between; width: 100%; height: 100%;">
          <span style="font-size: 0.78rem; color: #94a3b8; font-weight: 500;">
            🛡️ [${escapeHTML(matchInfo.displayName)}] gizlendi.
          </span>
          <button type="button" class="oe-btn oe-btn-reveal" style="padding: 2px 8px; font-size: 0.7rem;">
            Geri Getir
          </button>
        </div>
      `;
      const unhideBtn = banner.querySelector('.oe-btn-reveal');
      attachSafeAction(unhideBtn, () => {
        container.removeAttribute('data-oe-dismissed');
        container.classList.remove('oe-dismissed-container');
        container.style.minHeight = '';
        container.style.maxHeight = '';
        container.style.height = '';
        container.style.overflow = '';
        doReveal();
      });
    }
    attachSafeAction(btnDismiss, doDismiss);

    // Append banner inside container as an absolute overlay
    container.appendChild(banner);
  }

  // Scan text nodes & key element attributes across document
  function scanDocument(rootNode = document.body) {
    if (!config.enabled || !rootNode) return;

    if (isCurrentPageExcluded()) {
      cleanAllShields();
      chrome.runtime.sendMessage({ type: 'UPDATE_BADGE', count: 0, isExcluded: true });
      return;
    }

    if (compiledMatchers.length === 0) return;

    // 1. Clear any accidental shield on active media player controls
    cleanPlayerBarShields();

    // 2. Dynamically recheck existing shields to unshield any elements that changed to safe content
    recheckShieldedContainers();

    const containersToShield = new Map();

    function checkCandidate(str, elem) {
      if (!str || str.length < 3) return;
      if (isMediaControllerElement(elem)) return; // DO NOT card-shield live player controls!

      const folded = foldTurkish(str);
      for (const m of compiledMatchers) {
        if (m.regex.test(folded)) {
          const container = findBlockContainer(elem);
          if (
            container &&
            !isMediaControllerElement(container) &&
            !container.hasAttribute('data-oe-shielded') &&
            !container.hasAttribute('data-oe-revealed') &&
            !containersToShield.has(container)
          ) {
            containersToShield.set(container, m);
          }
          break;
        }
      }
    }

    // 3. Scan attributes on targeted elements (titles, handles, media labels)
    const attrSelector = 'a[title], a[aria-label], yt-formatted-string[title], yt-formatted-string[aria-label], [data-testid="UserCell"] [aria-label], article [aria-label], img[alt]';
    const attrNodes = (rootNode === document.body ? document : rootNode).querySelectorAll(attrSelector);
    for (let i = 0; i < attrNodes.length; i++) {
      const el = attrNodes[i];
      if (
        el.closest('.oe-shield-banner') ||
        el.closest('.oe-revert-tag') ||
        el.closest('[data-oe-shielded]') ||
        el.closest('[data-oe-revealed]') ||
        isMediaControllerElement(el)
      ) {
        continue;
      }

      const title = el.getAttribute('title');
      if (title) checkCandidate(title, el);

      const aria = el.getAttribute('aria-label');
      if (aria) checkCandidate(aria, el);

      const alt = el.getAttribute('alt');
      if (alt) checkCandidate(alt, el);
    }

    // 4. Scan text nodes using TreeWalker
    const walker = document.createTreeWalker(
      rootNode,
      NodeFilter.SHOW_TEXT,
      {
        acceptNode: function (node) {
          const parent = node.parentElement;
          if (!parent) return NodeFilter.FILTER_REJECT;
          const tag = parent.tagName.toLowerCase();
          if (
            tag === 'script' ||
            tag === 'style' ||
            tag === 'textarea' ||
            tag === 'input' ||
            parent.closest('.oe-shield-banner') ||
            parent.closest('.oe-revert-tag') ||
            parent.closest('[data-oe-shielded]') ||
            parent.closest('[data-oe-revealed]') ||
            isMediaControllerElement(parent)
          ) {
            return NodeFilter.FILTER_REJECT;
          }
          if (node.nodeValue.trim().length < 3) {
            return NodeFilter.FILTER_SKIP;
          }
          return NodeFilter.FILTER_ACCEPT;
        }
      }
    );

    let textNode = walker.nextNode();
    while (textNode) {
      checkCandidate(textNode.nodeValue, textNode);
      textNode = walker.nextNode();
    }

    // Execute shielding on newly detected containers
    for (const [container, matchInfo] of containersToShield.entries()) {
      shieldElement(container, matchInfo);
    }

    // Synchronize live badge count
    shieldedCount = document.querySelectorAll('[data-oe-shielded]').length;
    chrome.runtime.sendMessage({ type: 'UPDATE_BADGE', count: shieldedCount });
  }

  // Trigger debounced scan for dynamic SPAs
  function triggerScan() {
    if (scanTimer) clearTimeout(scanTimer);
    scanTimer = setTimeout(() => {
      scanDocument();
    }, 150);
  }

  // Observe DOM for infinite scrolling & dynamic SPA frameworks (X/Twitter, YouTube, Reddit)
  function startObserver() {
    const observer = new MutationObserver(() => {
      triggerScan();
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });

    const handleUrlChange = () => {
      if (isCurrentPageExcluded()) {
        cleanAllShields();
        chrome.runtime.sendMessage({ type: 'UPDATE_BADGE', count: 0, isExcluded: true });
      } else {
        triggerScan();
      }
    };

    // YouTube & SPA navigation events
    window.addEventListener('yt-navigate-finish', () => setTimeout(handleUrlChange, 200));
    window.addEventListener('yt-page-data-updated', () => setTimeout(handleUrlChange, 200));
    window.addEventListener('popstate', handleUrlChange);
    window.addEventListener('hashchange', handleUrlChange);
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

  // Listen for setting changes from popup & background
  chrome.storage.onChanged.addListener((changes, area) => {
    if (area === 'local' && changes.settings) {
      const s = changes.settings.newValue || {};
      config.enabled = s.enabled !== false;
      config.autoSkip = s.autoSkip !== false;
      if (!config.enabled) {
        cleanAllShields();
        chrome.runtime.sendMessage({ type: 'UPDATE_BADGE', count: 0 });
      } else {
        scanDocument();
      }
    }
    if (area === 'local' && (changes.persons || changes.aliases || changes.excludeList || changes.keywords)) {
      init();
    }
  });

  // Message handler for live stats & manual rescan from popup
  chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
    if (msg.type === 'GET_PAGE_STATS') {
      const excluded = isCurrentPageExcluded();
      sendResponse({ 
        shieldedCount: excluded ? 0 : shieldedCount,
        isExcluded: excluded,
        pageUrl: window.location.href
      });
      return false;
    }
    if (msg.type === 'RESCAN') {
      scanDocument();
      const excluded = isCurrentPageExcluded();
      sendResponse({ 
        ok: true, 
        shieldedCount: excluded ? 0 : shieldedCount,
        isExcluded: excluded,
        pageUrl: window.location.href
      });
      return false;
    }
  });

  // Initialize script
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
