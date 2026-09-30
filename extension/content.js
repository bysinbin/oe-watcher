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
  let isScanning = false;
  let scanPending = false;

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

  // Load config & data from background
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

  // Compile regex matchers with word boundary guards for fast scanning
  function compileMatchers() {
    const excludeSet = new Set(config.excludeList.map(e => foldTurkish(e)));
    compiledMatchers = [];

    // 1. Direct persons
    for (const p of config.persons) {
      const cleanName = foldTurkish(p.name);
      if (cleanName.length < 3 || excludeSet.has(cleanName)) continue;

      // Escape regex special chars
      const escaped = cleanName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
      // Word boundary regex: ensures "Cem" doesn't match "acemi", "Ali" doesn't match "kalibre"
      const regex = new RegExp(`(?:^|[^a-zçğıöşüA-ZÇĞİÖŞÜ0-9])${escaped}(?:$|[^a-zçğıöşüA-ZÇĞİÖŞÜ0-9])`, 'i');

      compiledMatchers.push({
        displayName: p.name,
        category: (p.tags && p.tags[0]) || 'Kara Liste',
        topics: p.topics || [],
        regex: regex
      });
    }

    // 2. Custom Aliases (Alias -> Target)
    if (config.aliases) {
      for (const [alias, target] of Object.entries(config.aliases)) {
        const cleanAlias = foldTurkish(alias);
        if (cleanAlias.length < 3 || excludeSet.has(cleanAlias)) continue;
        const escaped = cleanAlias.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const regex = new RegExp(`(?:^|[^a-zçğıöşüA-ZÇĞİÖŞÜ0-9])${escaped}(?:$|[^a-zçğıöşüA-ZÇĞİÖŞÜ0-9])`, 'i');

        compiledMatchers.push({
          displayName: `${target} (Alias: ${alias})`,
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
      // Social Media / Video / News specific containers
      if (
        curr.matches?.('article') ||
        curr.matches?.('[data-testid="tweet"]') ||
        curr.matches?.('[data-testid="cellInnerDiv"]') ||
        curr.matches?.('ytd-video-renderer') ||
        curr.matches?.('ytd-rich-item-renderer') ||
        curr.matches?.('ytd-compact-video-renderer') ||
        curr.matches?.('ytd-grid-video-renderer') ||
        curr.matches?.('shreddit-post') ||
        curr.matches?.('[data-testid="post-container"]') ||
        curr.matches?.('.entry-item') ||
        curr.matches?.('li[data-id]') ||
        curr.matches?.('.card') ||
        curr.matches?.('.news-item') ||
        curr.matches?.('.post') ||
        curr.matches?.('blockquote')
      ) {
        return curr;
      }

      // If it's a paragraph or header
      const tag = curr.tagName.toLowerCase();
      if (tag === 'p' || tag === 'li' || tag === 'blockquote') {
        return curr;
      }

      curr = curr.parentElement;
    }

    // Fallback: direct element parent
    return node.nodeType === Node.ELEMENT_NODE ? node : node.parentElement;
  }

  // Apply shield with modern glassmorphism blur and confirmation prompt
  function shieldElement(container, matchInfo) {
    if (!container || container.hasAttribute('data-oe-shielded')) return;
    container.setAttribute('data-oe-shielded', 'true');

    // Create wrapper & banner
    container.classList.add('oe-shielded-target');

    // Shield banner element
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

    // Handle "İçeriği Göster"
    const btnReveal = banner.querySelector('.oe-btn-reveal');
    btnReveal.addEventListener('click', (e) => {
      e.stopPropagation();
      e.preventDefault();
      container.classList.remove('oe-shielded-target');
      container.classList.add('oe-revealed-target');
      banner.style.display = 'none';

      // Insert minimal revert tag
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

    // Handle "Gizli Tut"
    const btnDismiss = banner.querySelector('.oe-btn-dismiss');
    btnDismiss.addEventListener('click', (e) => {
      e.stopPropagation();
      e.preventDefault();
      container.style.display = 'none';
      banner.innerHTML = `
        <div class="oe-shield-body" style="padding: 6px 12px; margin: 0;">
          <span style="font-size: 0.8rem; color: #94a3b8;">🛡️ [${escapeHTML(matchInfo.displayName)}] içeren paylaşım gizlendi.</span>
          <button type="button" class="oe-btn oe-btn-reveal" style="padding: 2px 8px; font-size: 0.72rem; margin-left: 10px;">Geri Getir</button>
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

  // Scan text nodes across document
  function scanDocument(rootNode = document.body) {
    if (!config.enabled || compiledMatchers.length === 0 || !rootNode) return;

    const walker = document.createTreeWalker(
      rootNode,
      NodeFilter.SHOW_TEXT,
      {
        acceptNode: function (node) {
          // Skip script, style, input, textarea, and already shielded nodes
          const parent = node.parentElement;
          if (!parent) return NodeFilter.FILTER_REJECT;
          const tag = parent.tagName.toLowerCase();
          if (
            tag === 'script' ||
            tag === 'style' ||
            tag === 'textarea' ||
            tag === 'input' ||
            parent.closest('.oe-shield-banner') ||
            parent.closest('.oe-revert-tag')
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

    const matchesToShield = [];
    let currentNode = walker.nextNode();

    while (currentNode) {
      const text = foldTurkish(currentNode.nodeValue);
      for (const m of compiledMatchers) {
        if (m.regex.test(text)) {
          const container = findBlockContainer(currentNode);
          if (container && !container.hasAttribute('data-oe-shielded')) {
            matchesToShield.push({ container, matchInfo: m });
          }
          break; // One match per block is sufficient
        }
      }
      currentNode = walker.nextNode();
    }

    for (const item of matchesToShield) {
      shieldElement(item.container, item.matchInfo);
    }
  }

  // Observe DOM for infinite scrolling & dynamic SPA frameworks (React, Vue, X/Twitter, YouTube)
  function startObserver() {
    let debounceTimer = null;
    const observer = new MutationObserver((mutations) => {
      if (scanPending) return;
      scanPending = true;

      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        scanPending = false;
        if (isScanning) return;
        isScanning = true;

        if (window.requestIdleCallback) {
          requestIdleCallback(() => {
            scanDocument();
            isScanning = false;
          }, { timeout: 300 });
        } else {
          scanDocument();
          isScanning = false;
        }
      }, 200);
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });
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

  // Listen for setting changes from popup
  chrome.storage.onChanged.addListener((changes, area) => {
    if (area === 'local' && changes.settings) {
      config.enabled = changes.settings.newValue ? changes.settings.newValue.enabled !== false : true;
      if (!config.enabled) {
        // Remove all shields
        document.querySelectorAll('.oe-shield-banner, .oe-revert-tag').forEach(el => el.remove());
        document.querySelectorAll('.oe-shielded-target').forEach(el => {
          el.classList.remove('oe-shielded-target');
          el.removeAttribute('data-oe-shielded');
        });
        chrome.runtime.sendMessage({ type: 'UPDATE_BADGE', count: 0 });
      } else {
        scanDocument();
      }
    }
    if (area === 'local' && (changes.persons || changes.aliases || changes.excludeList)) {
      init();
    }
  });

  // Start initialization
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
