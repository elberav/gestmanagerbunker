<script>
  import { onMount } from 'svelte';
  import { slide } from 'svelte/transition';
  import { Call_GetAccounts, Call_SearchAccounts, Call_LockSession, Call_CopyToClipboard, Call_ChangeMasterPassword, Call_ExportAccounts, Call_CheckUpdate, Call_DownloadUpdate, Call_ApplyUpdate } from '../wailsjs/go/main/App.js';
  import AccountNode from './AccountNode.svelte';
  import LoginOverlay from './LoginOverlay.svelte';
  import AccountForm from './AccountForm.svelte';
  import VaultTab from './VaultTab.svelte';
  import DonationPanel from './DonationPanel.svelte';
  import { t, setLang, initLang, lang } from './i18n.js';
  import logoImg from './assets/images/logo-universal.png';

  let accounts = [];
  let loading = true;
  let isUnlocked = false; 
  let showAddForm = false; 
  let currentTab = "active";
  let searchQuery = ""; 
  let filterProvider = "Todos";
  let filterDomain = "Todos";
  let quickPw = "";
  let quickLen = 16;
  let quickToast = "";

  // Change Password Modal
  let showChangePassword = false;
  let cpCurrentPw = "";
  let cpNewPw = "";
  let cpConfirmPw = "";
  let cpError = "";
  let cpSuccess = false;
  let cpNewCode = "";
  let cpLoading = false;
  let cpCodeCopied = false;

  let exportToast = "";
  let updateInfo = null;
  let updateDismissed = false;
  let updateState = ""; // "", "downloading", "applying", "error"
  let updateError = "";
  let inactivityTimer;
  const INACTIVITY_TIMEOUT = 10 * 60 * 1000; // 10 minutos en milisegundos

  async function handleUpdate() {
    if (!updateInfo) return;
    updateState = "downloading";
    updateError = "";
    try {
      const dlResult = await Call_DownloadUpdate(updateInfo.download_url);
      if (dlResult !== "SUCCESS") {
        updateState = "error";
        updateError = dlResult;
        return;
      }
      updateState = "applying";
      setTimeout(() => {
        Call_ApplyUpdate();
      }, 500);
    } catch (e) {
      updateState = "error";
      updateError = e.toString();
    }
  }

  async function checkForUpdate() {
    try {
      const info = await Call_CheckUpdate();
      if (info && info.has_update) {
        updateInfo = info;
      }
    } catch (e) {
      // Silencio — sin internet no es critico
    }
  }

  async function handleExport() {
    const result = await Call_ExportAccounts();
    if (result === "SUCCESS") {
      exportToast = "✓ " + $t('exportSuccess');
    } else if (result !== "") {
      exportToast = "✗ " + result;
    }
    setTimeout(() => exportToast = '', 3000);
  }

  function quickGenerate() {
    const charset = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%&*_-+=?';
    const values = crypto.getRandomValues(new Uint8Array(quickLen));
    quickPw = Array.from(values, v => charset[v % charset.length]).join('');
    Call_CopyToClipboard(quickPw);
    quickToast = $t('passwordCopied');
    setTimeout(() => quickToast = '', 2000);
  }

  // $lang auto-suscribe reactivamente
  
  function refreshDatabase() {
    loadAccounts();
    showAddForm = false;
  }

  // Recargar al cambiar búsqueda o filtros
  $: if (searchQuery !== undefined || filterProvider || filterDomain) {
    loadAccounts();
  }
  
  onMount(() => {
    initLang();
    loading = false;
    // Iniciar el sistema de auto-bloqueo por inactividad
    startInactivityTimer();
    const events = ['mousemove', 'keydown', 'click', 'scroll', 'touchstart'];
    events.forEach(evt => window.addEventListener(evt, resetInactivityTimer));
    return () => {
      events.forEach(evt => window.removeEventListener(evt, resetInactivityTimer));
      clearTimeout(inactivityTimer);
    };
  });

  function startInactivityTimer() {
    inactivityTimer = setTimeout(async () => {
      if (isUnlocked) {
        await handleLock();
      }
    }, INACTIVITY_TIMEOUT);
  }

  function resetInactivityTimer() {
    if (!isUnlocked) return; // No reiniciar si la bóveda ya está bloqueada
    clearTimeout(inactivityTimer);
    startInactivityTimer();
  }

  function setTab(tab) {
    currentTab = tab;
    loadAccounts();
  }

  async function loadAccounts() {
    loading = true;
    try {
      if (searchQuery.trim() === "") {
        accounts = await Call_GetAccounts(currentTab, null);
      } else {
        accounts = await Call_SearchAccounts(searchQuery, currentTab);
      }
      if (!accounts) accounts = [];

      if (filterProvider !== "Todos") {
        accounts = accounts.filter(a => a.email.toLowerCase().includes(filterProvider.toLowerCase()));
      }
      if (filterDomain !== "Todos") {
        accounts = accounts.filter(a => a.email.toLowerCase().endsWith(filterDomain.toLowerCase()));
      }
    } catch (e) {
      console.error(e);
      accounts = [];
    }
    loading = false;
  }

  function changeLang(event) {
    setLang(event.target.value);
  }

  async function handleLock() {
    await Call_LockSession();
    accounts = [];
    isUnlocked = false;
    clearTimeout(inactivityTimer);
  }

  function openChangePassword() {
    cpCurrentPw = ""; cpNewPw = ""; cpConfirmPw = "";
    cpError = ""; cpSuccess = false; cpNewCode = "";
    cpLoading = false;
    showChangePassword = true;
  }

  async function handleChangePassword() {
    cpError = "";
    if (!cpCurrentPw || !cpNewPw) {
      cpError = $t('emptyPasswordError'); return;
    }
    if (cpNewPw !== cpConfirmPw) {
      cpError = $t('passwordMismatchError'); return;
    }
    cpLoading = true;
    try {
      cpNewCode = await Call_ChangeMasterPassword(cpCurrentPw, cpNewPw);
      cpSuccess = true;
    } catch (e) {
      cpError = $t('wrongCurrentPassword');
    } finally {
      cpLoading = false;
    }
  }
</script>

{#if !isUnlocked}
  <LoginOverlay on:unlocked={() => { isUnlocked = true; loadAccounts(); resetInactivityTimer(); checkForUpdate(); }} />
{/if}

{#if showAddForm}
  <AccountForm parentID={null} on:cancel={() => showAddForm = false} on:saved={refreshDatabase} />
{/if}

<main class={!isUnlocked ? 'blur-bg' : ''}>
  <div class="header">
    <div class="header-left">
      <img src={logoImg} alt="Logo" class="logo" />
      <span class="app-title">{$t('appTitle')}</span>
    </div>
    <div class="header-right">
      <button class="lock-btn gear-btn" on:click={openChangePassword} title={$t('btnChangePassword')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
      </button>
      <button class="lock-btn" on:click={handleLock} title={$t('btnLock')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
      </button>
      <select class="lang-select" value={$lang} on:change={changeLang}>
        <option value="es">🇪🇸 Español</option>
        <option value="en">🇺🇸 English</option>
      </select>
    </div>
  </div>

  <div class="tabs">
    <button class="tab-btn {currentTab === 'active' ? 'active-tab' : ''}" on:click={() => setTab('active')}>
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
      {$t('tabActive')}
    </button>
    <button class="tab-btn {currentTab === 'vault' ? 'active-tab' : ''}" on:click={() => setTab('vault')}>
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
      {$t('tabVault')}
    </button>
    <button class="tab-btn {currentTab === 'recycled' ? 'active-tab' : ''}" on:click={() => setTab('recycled')}>
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
      {$t('tabRecycled')}
    </button>
    <button class="tab-btn donation-tab {currentTab === 'donation' ? 'active-tab' : ''}" on:click={() => setTab('donation')}>
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
      {$t('tabDonation')}
    </button>
  </div>

  {#if currentTab === 'vault'}
    <VaultTab />
  {:else if currentTab === 'donation'}
    <DonationPanel />
  {:else}
    <div class="toolbar">
      <div class="search-group">
        <div class="search-wrapper">
          <svg class="search-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
          <input 
            type="text" 
            placeholder={$t('searchPlaceholder')} 
            class="search-box"
            bind:value={searchQuery}
          />
        </div>
        <div class="quick-gen">
          <select bind:value={quickLen} class="gen-len-select">
            <option value={12}>12</option>
            <option value={16}>16</option>
            <option value={20}>20</option>
            <option value={24}>24</option>
            <option value={32}>32</option>
          </select>
          <button class="btn-quick-gen" on:click={quickGenerate} title={$t('btnGenerate')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="M7 15h0M2 9.5h20"/></svg>
          </button>
          {#if quickPw}
            <span class="quick-pw-display" role="button" tabindex="0" on:click={() => { Call_CopyToClipboard(quickPw); quickToast = $t('passwordCopied'); setTimeout(() => quickToast = '', 2000); }} on:keydown={(e) => e.key === 'Enter' && Call_CopyToClipboard(quickPw)} title={$t('copyPassword')}>{quickPw}</span>
          {/if}
          {#if quickToast}
            <span class="quick-toast">✓ {quickToast}</span>
          {/if}
          {#if exportToast}
            <span class="export-toast">{exportToast}</span>
          {/if}
        </div>
        <div class="filters">
          <select bind:value={filterProvider} class="filter-select">
            <option value="Todos">{$t('filterProvider')}</option>
            <option value="gmail">{$t('providerGmail')}</option>
            <option value="outlook">{$t('providerOutlook')}</option>
            <option value="hotmail">{$t('providerHotmail')}</option>
            <option value="yahoo">{$t('providerYahoo')}</option>
          </select>
          <select bind:value={filterDomain} class="filter-select">
            <option value="Todos">{$t('filterDomain')}</option>
            <option value=".com">{$t('domainCom')}</option>
            <option value=".pe">{$t('domainPe')}</option>
            <option value=".org">{$t('domainOrg')}</option>
            <option value=".net">{$t('domainNet')}</option>
          </select>
        </div>
      </div>

      <div class="right-toolbar">
        <span class="count-badge">{accounts.length} {$t('countRecords')}</span>
        {#if currentTab === 'active'}
          <button class="btn-export" on:click={handleExport} title={$t('btnExport')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            {$t('btnExport')}
          </button>
          <button class="btn-add" on:click={() => showAddForm = true}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            {$t('btnNewAccount')}
          </button>
        {/if}
      </div>
    </div>

    {#if updateInfo && !updateDismissed && !updateState}
      <div class="update-banner">
        <span>{$t('updateAvailable').replace('{version}', updateInfo.latest_version)}</span>
        <button class="update-link" on:click={handleUpdate}>{$t('btnUpdateNow')}</button>
        <button class="update-dismiss" on:click={() => updateDismissed = true}>✕</button>
      </div>
    {/if}

    {#if updateState === "downloading"}
      <div class="update-banner downloading">
        <span>{$t('updateDownloading')}</span>
        <div class="update-spinner"></div>
      </div>
    {/if}

    {#if updateState === "applying"}
      <div class="update-banner applying">
        <span>{$t('updateApplying')}</span>
      </div>
    {/if}

    {#if updateState === "error"}
      <div class="update-banner error">
        <span>✗ {updateError}</span>
        <button class="update-dismiss" on:click={() => { updateState = ''; updateError = ''; }}>✕</button>
      </div>
    {/if}
    {#if loading}
      <div class="loader">{$t('loadingDB')}</div>
    {:else if accounts.length === 0}
      <div class="empty">
        {#if currentTab === 'recycled'}
          {#if searchQuery}
            <div class="thinking-container trash-fail">
              <svg class="thinking-bubble-glow" xmlns="http://www.w3.org/2000/svg" width="68" height="68" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" stroke="url(#orange-grad)" fill="rgba(249, 115, 22, 0.05)"/>
                <circle cx="9" cy="10" r="1.2" fill="#fb923c"/>
                <circle cx="15" cy="10" r="1.2" fill="#fb923c"/>
                <defs>
                  <linearGradient id="orange-grad" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" style="stop-color:#f97316;stop-opacity:1" />
                    <stop offset="100%" style="stop-color:#fb923c;stop-opacity:1" />
                  </linearGradient>
                </defs>
              </svg>
              <p>{$t('noSearchMatch')}</p>
            </div>
          {:else}
            <div class="emoji-container trash-success">
              <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10" stroke="url(#blue-grad)" fill="rgba(6, 182, 212, 0.05)"/>
                <path class="smile-happy" d="M8 13.5c1.5 2 4.5 2 6 0" stroke="#10b981" />
                <circle cx="8.5" cy="10" r="1" fill="#10b981"/>
                <circle cx="15.5" cy="10" r="1" fill="#10b981"/>
                <defs>
                  <linearGradient id="blue-grad" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" style="stop-color:#06b6d4;stop-opacity:1" />
                    <stop offset="100%" style="stop-color:#10b981;stop-opacity:1" />
                  </linearGradient>
                </defs>
              </svg>
              <p>{$t('emptyRecycled')}</p>
            </div>
          {/if}
        {:else}
          <div class="thinking-container">
            <svg class="thinking-bubble-glow" xmlns="http://www.w3.org/2000/svg" width="68" height="68" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" stroke="url(#cyan-grad)" fill="rgba(6, 182, 212, 0.05)"/>
              <circle cx="9" cy="10" r="1.2" fill="#22d3ee"/>
              <circle cx="15" cy="10" r="1.2" fill="#22d3ee"/>
              {#if !searchQuery}
                <path class="smile-sad" d="M10.5 13.5 Q12 12 13.5 13.5" stroke="#0ea5e9" fill="none" stroke-linecap="round" />
              {/if}
              <defs>
                <linearGradient id="cyan-grad" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" style="stop-color:#0ea5e9;stop-opacity:1" />
                  <stop offset="100%" style="stop-color:#22d3ee;stop-opacity:1" />
                </linearGradient>
              </defs>
            </svg>
            <p>{searchQuery ? $t('noSearchMatch') : $t('emptyDB')}</p>
          </div>
        {/if}
      </div>
    {:else}
      {#key currentTab}
        <div class="card-list" in:slide={{ duration: 100 }}>
          {#each accounts as acc (acc.id)}
            <AccountNode acc={acc} isRecycled={currentTab === 'recycled'} on:refresh={refreshDatabase} />
          {/each}
        </div>
      {/key}
    {/if}
  {/if}
</main>

{#if showChangePassword}
  <div class="cp-overlay">
    <div class="cp-modal">
      {#if cpSuccess}
        <div class="cp-success-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#22c55e" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
        </div>
        <h2 class="cp-title" style="color: #22c55e;">{$t('changePasswordSuccess')}</h2>
        <p class="cp-subtitle">{$t('recoveryCodeSubtitle')}</p>
        <div class="cp-code-display">
          <span class="cp-code-value">{cpNewCode}</span>
          <button class="cp-copy-btn" class:cp-copied={cpCodeCopied} on:click={() => { navigator.clipboard.writeText(cpNewCode); cpCodeCopied = true; setTimeout(() => cpCodeCopied = false, 2000); }}>
            {#if cpCodeCopied}
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6L9 17l-5-5"/></svg>
              <span class="cp-tooltip">{$t('copied')}</span>
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
            {/if}
          </button>
        </div>
        <button class="cp-btn-done" on:click={() => showChangePassword = false}>{$t('btnCodeSaved')}</button>
      {:else}
        <div class="cp-header-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="#f59e0b" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
        </div>
        <h2 class="cp-title">{$t('changePasswordTitle')}</h2>

        {#if cpError}
          <div class="cp-error">{cpError}</div>
        {/if}

        <form on:submit|preventDefault={handleChangePassword}>
          <div class="cp-input-wrap">
            <svg class="cp-input-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
            <input type="password" bind:value={cpCurrentPw} placeholder={$t('currentPasswordPlaceholder')} autocomplete="off" />
          </div>
          <div class="cp-input-wrap">
            <svg class="cp-input-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
            <input type="password" bind:value={cpNewPw} placeholder={$t('newPasswordPlaceholder')} autocomplete="off" />
          </div>
          <div class="cp-input-wrap">
            <svg class="cp-input-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
            <input type="password" bind:value={cpConfirmPw} placeholder={$t('confirmNewPasswordPlaceholder')} autocomplete="off" />
          </div>
          <div class="cp-actions">
            <button type="button" class="cp-btn-cancel" on:click={() => showChangePassword = false}>{$t('btnCancel')}</button>
            <button type="submit" class="cp-btn-submit" disabled={cpLoading}>
              {cpLoading ? '...' : $t('btnChangePassword')}
            </button>
          </div>
        </form>
      {/if}
    </div>
  </div>
{/if}

<style>
  :global(body) { 
    background-color: #1a1d23 !important; 
    color: #e2e8f0 !important; 
    font-family: 'Segoe UI', 'Inter', system-ui, sans-serif; 
    margin: 0; 
    padding: 0;
  }
  main { 
    padding: 20px 28px; 
    max-width: 940px; 
    margin: auto; 
    animation: fadeIn 0.6s ease-out;
  }
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .header { 
    display: flex; 
    justify-content: space-between; 
    align-items: center;
    margin-bottom: 22px; 
    border-bottom: 1px solid rgba(255,255,255,0.08); 
    padding-bottom: 16px;
  }
  .header-left { display: flex; align-items: center; gap: 12px; }
  .header-right { display: flex; align-items: center; gap: 10px; }
  .logo { width: 38px; height: 38px; border-radius: 10px; filter: drop-shadow(0 2px 4px rgba(0,0,0,0.3)); }
  .app-title { font-size: 22px; font-weight: 700; letter-spacing: -0.3px; color: #f8fafc; }
  .lock-btn {
    background: transparent; border: 1px solid rgba(255,255,255,0.1); border-radius: 8px;
    padding: 10px; color: #94a3b8; cursor: pointer; display: flex; align-items: center; justify-content: center;
    transition: all 0.1s;
  }
  .lock-btn:hover { background: rgba(239,68,68,0.1); color: #ef4444; border-color: rgba(239,68,68,0.3); }
  .lang-select { 
    background: #252830; 
    color: #e2e8f0; 
    border: 1px solid rgba(255,255,255,0.1); 
    border-radius: 8px; 
    padding: 6px 12px; 
    font-size: 13px; 
    cursor: pointer;
    transition: border-color 0.2s;
  }
  .lang-select:hover { border-color: #3b82f6; }
  
  .tabs { display: flex; gap: 6px; margin-bottom: 22px; border-bottom: 1px solid rgba(255,255,255,0.06); padding-bottom: 10px;}
  .tab-btn { 
    background: transparent; 
    color: #64748b; 
    border: none; 
    font-size: 14px; 
    cursor: pointer; 
    padding: 8px 14px; 
    font-weight: 600;
    display: flex; 
    align-items: center; 
    gap: 6px;
    border-radius: 8px;
    transition: all 0.2s;
  }
  .tab-btn:hover { color: #e2e8f0; background: rgba(255,255,255,0.05); }
  .tab-btn.active-tab { 
    color: #3b82f6; 
    background: rgba(59,130,246,0.1); 
  }
  .donation-tab { color: #ef4444; }
  .donation-tab:hover { color: #f87171 !important; background: rgba(239,68,68,0.1) !important; }
  .donation-tab.active-tab { color: #ef4444 !important; background: rgba(239,68,68,0.15) !important; }
  .donation-tab :global(svg) { animation: heartPulse 1.2s ease-in-out infinite; }
  @keyframes heartPulse {
    0%, 100% { transform: scale(1); }
    20% { transform: scale(1.17); }
    40% { transform: scale(1); }
    60% { transform: scale(1.17); }
    80% { transform: scale(1); }
  }
  
  .toolbar { display: flex; flex-direction: column; gap: 10px; margin-bottom: 16px;}
  .search-group { display: flex; gap: 10px; flex-wrap: wrap; }
  .search-wrapper { flex: 2; position: relative; }
  .search-icon { 
    position: absolute; 
    left: 12px; 
    top: 50%; 
    transform: translateY(-50%); 
    color: #64748b;
    pointer-events: none;
  }
  .search-box { 
    width: 100%; 
    padding: 10px 10px 10px 36px; 
    border-radius: 8px; 
    border: 1px solid rgba(255,255,255,0.1); 
    background: #252830; 
    color: #e2e8f0;
    box-sizing: border-box;
    transition: border-color 0.2s;
    font-size: 14px;
  }
  .search-box:focus { outline: none; border-color: #3b82f6; }
  .filters { display: flex; gap: 6px; flex: 1; min-width: 200px;}
  .filter-select { 
    background: #252830; 
    color: #94a3b8; 
    border: 1px solid rgba(255,255,255,0.1); 
    border-radius: 8px; 
    padding: 6px 8px; 
    font-size: 13px; 
    outline: none; 
    width: 100%;
    transition: border-color 0.2s;
  }
  .filter-select:hover { border-color: #3b82f6; color: #e2e8f0; }

  .right-toolbar { display: flex; justify-content: space-between; align-items: center; }
  .count-badge { color: #64748b; font-size: 13px; font-weight: 600; }
  .btn-add { 
    background: linear-gradient(135deg, #22c55e, #16a34a); 
    color: white; 
    border: none; 
    padding: 8px 16px; 
    border-radius: 8px; 
    cursor: pointer; 
    font-weight: 600;
    font-size: 13px;
    display: flex; 
    align-items: center; 
    gap: 6px;
    box-shadow: 0 2px 8px rgba(34,197,94,0.3);
    transition: all 0.2s;
  }
  .btn-add:hover { 
    background: linear-gradient(135deg, #16a34a, #15803d);
    box-shadow: 0 4px 12px rgba(34,197,94,0.4);
    transform: translateY(-1px);
  }

  .btn-export { 
    background: transparent; 
    color: #94a3b8; 
    border: 1px solid #334155; 
    padding: 8px 14px; 
    border-radius: 8px; 
    cursor: pointer; 
    font-weight: 600;
    font-size: 13px;
    display: flex; 
    align-items: center; 
    gap: 6px;
    transition: all 0.2s;
    white-space: nowrap;
  }
  .btn-export:hover { 
    border-color: #3b82f6; 
    color: #e2e8f0; 
    background: rgba(59,130,246,0.1);
  }

  .update-banner {
    display: flex;
    align-items: center;
    gap: 12px;
    background: linear-gradient(135deg, #1e3a5f, #1a2a4a);
    border: 1px solid #3b82f6;
    border-radius: 8px;
    padding: 10px 16px;
    margin: 8px 16px 0;
    font-size: 13px;
    color: #93c5fd;
    animation: toastIn 0.3s ease-out;
  }
  .update-banner.downloading {
    border-color: #f59e0b;
    background: linear-gradient(135deg, #422006, #2a1a00);
    color: #fbbf24;
  }
  .update-banner.applying {
    border-color: #22c55e;
    background: linear-gradient(135deg, #052e16, #0a1f0a);
    color: #4ade80;
  }
  .update-banner.error {
    border-color: #ef4444;
    background: linear-gradient(135deg, #3b0505, #2a0000);
    color: #f87171;
  }
  .update-link {
    background: #3b82f6;
    color: #fff;
    font-weight: 700;
    border: none;
    padding: 6px 16px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    transition: all 0.2s;
  }
  .update-link:hover {
    background: #2563eb;
    transform: translateY(-1px);
  }
  .update-link:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    transform: none;
  }
  .update-dismiss {
    background: transparent;
    border: none;
    color: #64748b;
    cursor: pointer;
    font-size: 16px;
    padding: 2px 6px;
    border-radius: 4px;
    margin-left: auto;
    transition: all 0.2s;
  }
  .update-dismiss:hover {
    background: rgba(255,255,255,0.1);
    color: #e2e8f0;
  }
  .update-spinner {
    width: 16px;
    height: 16px;
    border: 2px solid rgba(251,191,36,0.3);
    border-top-color: #fbbf24;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .export-toast {
    position: fixed;
    bottom: 24px;
    right: 24px;
    background: #1e293b;
    color: #e2e8f0;
    padding: 12px 20px;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    box-shadow: 0 4px 16px rgba(0,0,0,0.4);
    border: 1px solid #334155;
    z-index: 1000;
    animation: toastIn 0.25s ease-out;
  }

  @keyframes toastIn {
    from { opacity: 0; transform: translateY(12px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .card-list { display: flex; flex-direction: column; gap: 0; }

  /* Quick Password Generator (Toolbar) */
  .quick-gen {
    display: flex; align-items: center; gap: 6px; position: relative;
  }
  .gen-len-select {
    background: #252830; color: #94a3b8; border: 1px solid rgba(255,255,255,0.1);
    border-radius: 8px; padding: 6px 8px; font-size: 12px; font-weight: 600;
    cursor: pointer; transition: border-color 0.2s;
  }
  .gen-len-select:hover { border-color: #8b5cf6; color: #e2e8f0; }
  .btn-quick-gen {
    background: linear-gradient(135deg, #8b5cf6, #7c3aed); color: white;
    border: none; border-radius: 8px; padding: 8px 10px; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    box-shadow: 0 2px 6px rgba(139,92,246,0.3); transition: all 0.2s;
  }
  .btn-quick-gen:hover {
    background: linear-gradient(135deg, #7c3aed, #6d28d9);
    transform: translateY(-1px); box-shadow: 0 4px 10px rgba(139,92,246,0.4);
  }
  .quick-pw-display {
    font-family: 'Consolas', 'Courier New', monospace; font-size: 11px;
    background: rgba(139,92,246,0.1); color: #c4b5fd; padding: 4px 8px;
    border-radius: 6px; border: 1px solid rgba(139,92,246,0.25);
    cursor: pointer; max-width: 180px; overflow: hidden; text-overflow: ellipsis;
    white-space: nowrap; transition: all 0.2s;
  }
  .quick-pw-display:hover { background: rgba(139,92,246,0.2); color: #e9d5ff; }
  .quick-toast {
    position: absolute; top: -28px; right: 0;
    background: linear-gradient(135deg, #16a34a, #15803d); color: white;
    padding: 3px 10px; border-radius: 6px; font-size: 11px; font-weight: 600;
    pointer-events: none; animation: slideUpFade 1.8s forwards;
    box-shadow: 0 3px 8px rgba(34,197,94,0.3);
  }
  @keyframes slideUpFade {
    0% { opacity: 0; transform: translateY(6px); }
    15% { opacity: 1; transform: translateY(0); }
    75% { opacity: 1; transform: translateY(-3px); }
    100% { opacity: 0; transform: translateY(-8px); }
  }
  .empty, .loader { 
    color: #64748b; font-size: 15px; padding: 70px 20px; text-align: center;
    display: flex; flex-direction: column; align-items: center; gap: 16px;
  }
  .empty p { margin: 0; font-weight: 500; letter-spacing: 0.2px; color: #94a3b8; }

  /* Animaciones Emoji */
  .empty p { margin: 0; font-weight: 500; letter-spacing: 0.2px; color: #94a3b8; }

  /* Estilos Emoji Estáticos */
  .thinking-bubble-glow {
    filter: drop-shadow(0 0 12px rgba(6, 182, 212, 0.2));
  }
  .smile-sad {
    filter: drop-shadow(0 0 2px rgba(14, 165, 233, 0.3));
  }
  .thinking-container, .emoji-container {
    display: flex; flex-direction: column; align-items: center; gap: 12px;
  }
  @keyframes smileBounce {
    0% { transform: scaleY(0.8); }
    100% { transform: scaleY(1.3); }
  }
  @keyframes blinkEye {
    0%, 95%, 100% { transform: scaleY(1); }
    97.5% { transform: scaleY(0.1); }
  }

  .blur-bg { filter: blur(5px); opacity: 0.5; pointer-events: none;}

  /* Change Password Modal */
  .gear-btn:hover { background: rgba(245,158,11,0.1); color: #f59e0b; border-color: rgba(245,158,11,0.3); }
  .cp-overlay {
    position: fixed; top: 0; left: 0; width: 100vw; height: 100vh;
    background: rgba(0,0,0,0.7); backdrop-filter: blur(4px);
    display: flex; justify-content: center; align-items: center; z-index: 2000;
    animation: fadeIn 0.2s ease;
  }
  .cp-modal {
    background: #1e2028; border: 1px solid rgba(255,255,255,0.08); border-radius: 16px;
    padding: 32px; width: 420px; max-width: 90%; text-align: center;
    box-shadow: 0 20px 60px rgba(0,0,0,0.5), 0 0 40px rgba(245,158,11,0.08);
  }
  .cp-header-icon { color: #f59e0b; margin-bottom: 12px; filter: drop-shadow(0 2px 8px rgba(245,158,11,0.3)); }
  .cp-success-icon { margin-bottom: 12px; filter: drop-shadow(0 2px 8px rgba(34,197,94,0.3)); }
  .cp-title { margin: 0 0 16px 0; font-size: 20px; font-weight: 700; color: #f8fafc; }
  .cp-subtitle { font-size: 14px; color: #94a3b8; margin-bottom: 16px; line-height: 1.5; }
  .cp-error {
    color: #ef4444; font-size: 13px; margin-bottom: 14px; font-weight: 600;
    background: rgba(239,68,68,0.08); padding: 8px 12px; border-radius: 8px;
    border: 1px solid rgba(239,68,68,0.15);
  }
  .cp-input-wrap { position: relative; margin-bottom: 14px; }
  .cp-input-icon { position: absolute; left: 14px; top: 50%; transform: translateY(-50%); color: #64748b; pointer-events: none; }
  .cp-modal input {
    width: 100%; padding: 13px 14px 13px 40px; border-radius: 10px;
    border: 1px solid rgba(255,255,255,0.1); background: #252830; color: #e2e8f0;
    box-sizing: border-box; font-size: 15px; transition: border-color 0.2s, box-shadow 0.2s;
  }
  .cp-modal input:focus { outline: none; border-color: #f59e0b; box-shadow: 0 0 0 3px rgba(245,158,11,0.15); }
  .cp-actions { display: flex; gap: 10px; margin-top: 20px; }
  .cp-btn-cancel {
    flex: 1; padding: 12px; background: transparent; color: #94a3b8;
    border: 1px solid rgba(255,255,255,0.1); border-radius: 10px;
    font-weight: 600; font-size: 14px; cursor: pointer; transition: all 0.2s;
  }
  .cp-btn-cancel:hover { background: rgba(255,255,255,0.05); color: #e2e8f0; }
  .cp-btn-submit {
    flex: 1; padding: 12px; background: linear-gradient(135deg, #f59e0b, #d97706); color: #000;
    border: none; border-radius: 10px; font-weight: 600; font-size: 14px;
    cursor: pointer; transition: all 0.2s; box-shadow: 0 4px 14px rgba(245,158,11,0.3);
  }
  .cp-btn-submit:hover { background: linear-gradient(135deg, #d97706, #b45309); transform: translateY(-1px); }
  .cp-btn-submit:disabled { opacity: 0.5; cursor: not-allowed; }
  .cp-code-display {
    background: #1e293b; border: 1px solid rgba(255,255,255,0.06); border-radius: 12px;
    padding: 18px 16px; margin: 16px 0 20px 0; display: flex; align-items: center;
    justify-content: center; gap: 12px;
  }
  .cp-code-value {
    font-family: 'Consolas', monospace; font-size: 16px; font-weight: 600;
    color: #38bdf8; letter-spacing: 1.5px; user-select: all;
  }
  .cp-copy-btn {
    background: transparent; border: 1px solid rgba(255,255,255,0.1); border-radius: 8px;
    padding: 8px; cursor: pointer; color: #94a3b8; display: flex; align-items: center; transition: all 0.2s;
  }
  .cp-copy-btn:hover { background: rgba(255,255,255,0.1); color: #f8fafc; }
  .cp-copied { color: #38bdf8 !important; background: rgba(56,189,248,0.1) !important; border-color: rgba(56,189,248,0.3) !important; }
  .cp-tooltip {
    position: absolute; bottom: -32px; left: 50%; transform: translateX(-50%);
    background: #38bdf8; color: #0f1117; padding: 4px 8px; border-radius: 6px;
    font-size: 11px; font-weight: 700; white-space: nowrap; pointer-events: none;
    animation: cpFadeUp 0.2s ease-out forwards;
    box-shadow: 0 4px 12px rgba(56,189,248,0.3);
  }
  @keyframes cpFadeUp {
    0% { opacity: 0; transform: translate(-50%, 4px); }
    100% { opacity: 1; transform: translate(-50%, 0); }
  }
  .cp-btn-done {
    width: 100%; padding: 13px; background: #334155; color: #e2e8f0;
    border: 1px solid rgba(255,255,255,0.08); border-radius: 10px;
    font-weight: 600; font-size: 15px; cursor: pointer; transition: all 0.2s;
  }
  .cp-btn-done:hover { background: #475569; transform: translateY(-1px); }
</style>
