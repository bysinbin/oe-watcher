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

  // Find the exact block card or container for modern web layouts
  function findBlockContainer(node) {
    if (!node) return null;
    let curr = node.nodeType === Node.ELEMENT_NODE ? node : node.parentElement;

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

    return node.nodeType === Node.ELEMENT_NODE ? node : node.parentElement;
  }

  // Apply shield directly inside container as an overlay without disturbing virtual scrollers
  function shieldElement(container, matchInfo) {
    if (!container || container.hasAttribute('data-oe-shielded') || container.hasAttribute('data-oe-revealed')) {
      return;
    }
    container.setAttribute('data-oe-shielded', 'true');

    // Ensure container has relative positioning context and minimum height for overlay
    const computed = window.getComputedStyle(container);
    if (computed.position === 'static') {
      container.style.position = 'relative';
    }
    if (container.offsetHeight < 110) {
      container.style.minHeight = '110px';
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
      container.removeAttribute('data-oe-shielded');
      container.classList.remove('oe-shielded-container');
      container.classList.add('oe-revealed-container');

      container.style.minHeight = '';
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
          if (
            container &&
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

    // 1. Scan attributes on targeted elements (titles, handles, media labels)
    const attrSelector = 'a[title], a[aria-label], yt-formatted-string[title], yt-formatted-string[aria-label], [data-testid="UserCell"] [aria-label], article [aria-label], img[alt]';
    const attrNodes = (rootNode === document.body ? document : rootNode).querySelectorAll(attrSelector);
    for (let i = 0; i < attrNodes.length; i++) {
      const el = attrNodes[i];
      if (
        el.closest('.oe-shield-banner') ||
        el.closest('.oe-revert-tag') ||
        el.closest('[data-oe-shielded]') ||
        el.closest('[data-oe-revealed]')
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
            parent.closest('[data-oe-shielded]') ||
            parent.closest('[data-oe-revealed]')
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

  // Observe DOM for infinite scrolling & dynamic SPA frameworks (X/Twitter, YouTube, Reddit)
  function startObserver() {
    const observer = new MutationObserver(() => {
      triggerScan();
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });

    // YouTube & SPA navigation events
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
        document.querySelectorAll('.oe-shielded-container').forEach(el => {
          el.classList.remove('oe-shielded-container');
          el.removeAttribute('data-oe-shielded');
          el.style.minHeight = '';
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
