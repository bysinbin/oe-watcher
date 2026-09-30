// OE Watcher Chrome Extension - Content Script
(function () {
  'use strict';

  let config = {
    enabled: true,
    persons: [],
    aliases: {},
    excludeList: []
  };

  let compiledMatchers = [];
  let shieldedCount = 0;
  let scanTimer = null;

  // Modern block & card selectors across YouTube, X/Twitter, Reddit, and News portals
  const BLOCK_SELECTORS = [
    // YouTube
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
    // Twitter / X
    'article[data-testid="tweet"]',
    '[data-testid="cellInnerDiv"]',
    'article',
    // Reddit
    'shreddit-post',
    '[data-testid="post-container"]',
    // News / Forums / Common layouts
    '.entry-item',
    '.news-item',
    '.media-item',
    '.feed-item',
    '.card',
    '.post',
    'li[data-id]'
  ];

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

  // Load config & data from background service worker
  function init() {
    chrome.runtime.sendMessage({ type: 'GET_DATA' }, (response) => {
      if (!response) return;
      config.enabled = response.settings ? response.settings.enabled !== false : true;
      config.persons = response.persons || [];
      config.aliases = response.aliases || {};
      config.excludeList = response.excludeList || [];

      if (!config.enabled || config.persons.length === 0) {
        return;
      }

      compileMatchers();
      scanDocument();
      startObserver();
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
        // Word boundary: matches when surrounded by non-alphanumeric or edges
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
  }

  // Find the most appropriate block card or container for modern web layouts
  function findBlockContainer(node) {
    if (!node) return null;
    let curr = node.nodeType === Node.ELEMENT_NODE ? node : node.parentElement;

    while (curr && curr !== document.body && curr !== document.documentElement) {
      for (const sel of BLOCK_SELECTORS) {
        if (curr.matches && curr.matches(sel)) {
          return curr;
        }
      }

      // Fallback for standard tags
      const tag = curr.tagName.toLowerCase();
      if (tag === 'p' || tag === 'li' || tag === 'blockquote') {
        return curr;
      }

      curr = curr.parentElement;
    }

    // Direct element parent fallback
    return node.nodeType === Node.ELEMENT_NODE ? node : node.parentElement;
  }

  // Apply shield with modern glassmorphism blur and confirmation prompt
  function shieldElement(container, matchInfo) {
    if (!container || container.hasAttribute('data-oe-shielded')) return;
    container.setAttribute('data-oe-shielded', 'true');

    // Add blurred visual shield to container
    container.classList.add('oe-shielded-target');

    // Create prompt banner
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

    // Action: "İçeriği Göster" (Temporarily unmask content)
    const btnReveal = banner.querySelector('.oe-btn-reveal');
    btnReveal.addEventListener('click', (e) => {
      e.stopPropagation();
      e.preventDefault();
      container.classList.remove('oe-shielded-target');
      container.classList.add('oe-revealed-target');
      banner.style.display = 'none';

      const revertTag = document.createElement('div');
      revertTag.className = 'oe-revert-tag';
      revertTag.innerHTML = `
        <span>⚠️ [${escapeHTML(matchInfo.displayName)}] İçeriği Gösterildi</span>
        <button type="button" class="oe-btn-rehide">Yeniden Gizle</button>
      `;
      revertTag.querySelector('.oe-btn-rehide').addEventListener('click', (ev) => {
        ev.stopPropagation();
        container.classList.add('oe-shielded-target');
        container.classList.remove('oe-revealed-target');
        banner.style.display = 'flex';
        revertTag.remove();
      });

      container.insertAdjacentElement('beforebegin', revertTag);
    });

    // Action: "Gizli Tut" (Fully collapse container)
    const btnDismiss = banner.querySelector('.oe-btn-dismiss');
    btnDismiss.addEventListener('click', (e) => {
      e.stopPropagation();
      e.preventDefault();
      container.style.display = 'none';
      banner.innerHTML = `
        <div class="oe-shield-body" style="padding: 6px 12px; margin: 0; align-items: center;">
          <span style="font-size: 0.8rem; color: #94a3b8;">🛡️ [${escapeHTML(matchInfo.displayName)}] içeren paylaşım gizlendi.</span>
          <button type="button" class="oe-btn oe-btn-reveal" style="padding: 3px 10px; font-size: 0.72rem; margin-left: auto;">Geri Getir</button>
        </div>
      `;
      banner.querySelector('.oe-btn-reveal').addEventListener('click', () => {
        container.style.display = '';
        container.classList.remove('oe-shielded-target');
        banner.remove();
      });
    });

    container.insertAdjacentElement('beforebegin', banner);
    shieldedCount++;
    chrome.runtime.sendMessage({ type: 'UPDATE_BADGE', count: shieldedCount });
  }

  // Scan text nodes & key element attributes across document
  function scanDocument(rootNode = document.body) {
    if (!config.enabled || compiledMatchers.length === 0 || !rootNode) return;

    const containersToShield = new Map();

    function checkCandidate(str, elem) {
      if (!str || str.length < 3) return;
      const folded = foldTurkish(str);
      for (const m of compiledMatchers) {
        if (m.regex.test(folded)) {
          const container = findBlockContainer(elem);
          if (container && !container.hasAttribute('data-oe-shielded') && !containersToShield.has(container)) {
            containersToShield.set(container, m);
          }
          break;
        }
      }
    }

    // 1. Scan attributes commonly containing full titles on YouTube & social media
    const attrElements = (rootNode === document.body ? document : rootNode).querySelectorAll(
      'a[title], a[aria-label], yt-formatted-string[title], yt-formatted-string[aria-label], [aria-label], img[alt]'
    );
    for (let i = 0; i < attrElements.length; i++) {
      const el = attrElements[i];
      if (el.closest('.oe-shield-banner') || el.closest('.oe-revert-tag') || el.closest('[data-oe-shielded]')) continue;

      const title = el.getAttribute('title');
      if (title) checkCandidate(title, el);

      const aria = el.getAttribute('aria-label');
      if (aria) checkCandidate(aria, el);

      const alt = el.getAttribute('alt');
      if (alt) checkCandidate(alt, el);
    }

    // 2. Scan text nodes using TreeWalker
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
            parent.closest('[data-oe-shielded]')
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

    // Execute shielding on detected containers
    for (const [container, matchInfo] of containersToShield.entries()) {
      shieldElement(container, matchInfo);
    }
  }

  // Trigger debounced scan for dynamic SPAs
  function triggerScan() {
    if (scanTimer) clearTimeout(scanTimer);
    scanTimer = setTimeout(() => {
      scanDocument();
    }, 150);
  }

  // Observe DOM for infinite scrolling & dynamic SPA frameworks (YouTube, Twitter, Reddit)
  function startObserver() {
    const observer = new MutationObserver(() => {
      triggerScan();
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });

    // YouTube SPA navigation events
    window.addEventListener('yt-navigate-finish', () => {
      setTimeout(triggerScan, 200);
    });
    window.addEventListener('yt-page-data-updated', () => {
      setTimeout(triggerScan, 200);
    });

    window.addEventListener('popstate', triggerScan);
    window.addEventListener('hashchange', triggerScan);
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
      config.enabled = changes.settings.newValue ? changes.settings.newValue.enabled !== false : true;
      if (!config.enabled) {
        document.querySelectorAll('.oe-shield-banner, .oe-revert-tag').forEach(el => el.remove());
        document.querySelectorAll('.oe-shielded-target').forEach(el => {
          el.classList.remove('oe-shielded-target');
          el.removeAttribute('data-oe-shielded');
        });
        shieldedCount = 0;
        chrome.runtime.sendMessage({ type: 'UPDATE_BADGE', count: 0 });
      } else {
        scanDocument();
      }
    }
    if (area === 'local' && (changes.persons || changes.aliases || changes.excludeList)) {
      init();
    }
  });

  // Message handler for live stats & manual rescan from popup
  chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
    if (msg.type === 'GET_PAGE_STATS') {
      sendResponse({ shieldedCount: shieldedCount });
      return false;
    }
    if (msg.type === 'RESCAN') {
      scanDocument();
      sendResponse({ ok: true, shieldedCount: shieldedCount });
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
