<script>
  import { createEventDispatcher } from 'svelte';
  import { slide } from 'svelte/transition';
  import { Call_GetAllChildren, Call_Decrypt, Call_UpdateStatus, Call_DeletePermanently, Call_RestoreAccount, Call_GetAccountByID, Call_UpdateLastViewed, Call_CopyToClipboard } from '../wailsjs/go/main/App.js';
  import { BrowserOpenURL } from '../wailsjs/runtime/runtime.js';
  import AccountForm from './AccountForm.svelte';
  import { t } from './i18n.js';

  const dispatch = createEventDispatcher();

  export let acc;
  export let depth = 0;
  export let isRecycled = false;

  let expanded = false;
  let children = [];
  let loadingKids = false;
  
  let showAddChildForm = false;
  let showEditForm = false;
  let decryptedPassword = null;
  let decryptedNotes = null;
  let showNotes = false;
  let toastMessage = "";
  
  // Custom confirm dialog (sin "localhost")
  let showConfirmDialog = false;
  let confirmMessage = "";
  let pendingAction = null;

  async function toggleExpand() {
    if (!acc.has_children) return;

    expanded = !expanded;
    if (expanded && children.length === 0) {
      await reloadChildren();
    }
  }

  async function reloadChildren() {
    loadingKids = true;
    try {
      children = await Call_GetAllChildren(acc.id);
      if(!children) children = [];
    } catch (e) {
      console.error(e);
    }
    loadingKids = false;
  }

  function copyToClipboard(text, msg) {
    Call_CopyToClipboard(text).then(() => {
      toastMessage = `✓ ${msg}`;
      setTimeout(() => toastMessage = "", 1500);
    });
  }

  async function showPassword() {
    if (decryptedPassword) {
      decryptedPassword = null;
      return;
    }
    if (!acc.password_enc) return;
    try {
      decryptedPassword = await Call_Decrypt(acc.password_enc);
      Call_UpdateLastViewed(acc.id);
      acc.last_viewed = new Date().toISOString().replace('T', ' ').substring(0, 19);
    } catch (e) {
      toastMessage = $t('decryptError');
      setTimeout(() => toastMessage = "", 2000);
    }
  }

  async function copyPasswordSilent() {
    if (!acc.password_enc) return;
    try {
      let pwd = await Call_Decrypt(acc.password_enc);
      await Call_CopyToClipboard(pwd);
      Call_UpdateLastViewed(acc.id);
      acc.last_viewed = new Date().toISOString().replace('T', ' ').substring(0, 19);
      toastMessage = $t('passwordCopied');
      setTimeout(() => toastMessage = "", 2000);
    } catch (e) {
      toastMessage = $t('decryptError');
      setTimeout(() => toastMessage = "", 2000);
    }
  }

  function requestConfirm(message, action) {
    confirmMessage = message;
    pendingAction = action;
    showConfirmDialog = true;
  }

  function confirmYes() {
    showConfirmDialog = false;
    if (pendingAction) pendingAction();
    pendingAction = null;
  }

  function confirmNo() {
    showConfirmDialog = false;
    pendingAction = null;
  }

  function deleteAccount() {
    requestConfirm(
      `${$t('deleteConfirm')} ${acc.name}?`,
      async () => {
        await Call_UpdateStatus(acc.id, "recycled");
        acc = null;
      }
    );
  }

  async function restoreAccount() {
    await Call_RestoreAccount(acc.id);
    acc = null;
  }

  function destroyPermanently() {
    requestConfirm(
      `${$t('destroyConfirm')} ${acc.name}?`,
      async () => {
        await Call_DeletePermanently(acc.id);
        acc = null;
      }
    );
  }

  async function handleChildSaved() {
    showAddChildForm = false;
    acc.has_children = true;
    acc = acc; // trigger Svelte reactivity
    expanded = true;
    await reloadChildren();
  }

  function handleEditSaved() {
    showEditForm = false;
    dispatch('refresh');
  }

  function openUrlBlock(urlStr) {
    if (!urlStr) return;
    let finalUrl = urlStr.startsWith('http') ? urlStr : 'https://' + urlStr;
    BrowserOpenURL(finalUrl);
  }

  async function toggleNotes() {
    if (showNotes) { showNotes = false; decryptedNotes = null; return; }
    if (!acc.notes_enc || acc.notes_enc.length === 0) {
      toastMessage = $t('noNotes'); setTimeout(() => toastMessage = '', 2000); return;
    }
    try {
      decryptedNotes = await Call_Decrypt(acc.notes_enc);
      showNotes = true;
      Call_UpdateLastViewed(acc.id);
      acc.last_viewed = new Date().toISOString().replace('T', ' ').substring(0, 19);
    } catch (e) {
      toastMessage = $t('decryptError'); setTimeout(() => toastMessage = '', 2000);
    }
  }
</script>

<!-- Si fue borrada temporalmente en UI -->
{#if acc}
<div class="account-card">
  <div class="card-content" on:click={toggleExpand} on:keydown={(e) => e.key === 'Enter' && toggleExpand()} role="button" tabindex="0">
    <div class="main-info">

      {#if acc.has_children}
        <span class="icon" class:rotated={expanded}>
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>
        </span>
      {:else}
        <span class="icon empty-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="8" height="8" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="5"/></svg>
        </span>
      {/if}
      
      <span class="title">{acc.name}</span>
      
      {#if acc.tag}
        <span class="tag-badge">{acc.tag}</span>
      {/if}
      
      {#if acc.username}
        <span class="username" role="button" tabindex="0" on:click|stopPropagation={() => copyToClipboard(acc.username, $t('usernameCopiedToast'))} on:keydown={(e) => e.key === 'Enter' && copyToClipboard(acc.username, $t('usernameCopiedToast'))} title={$t('usernameCopiedToast')}>@{acc.username}</span>
      {/if}

      {#if acc.email}
        <span class="email">({acc.email})</span>
        <span class="control" role="button" tabindex="0" on:click|stopPropagation={() => copyToClipboard(acc.email, $t('emailCopiedToast'))} on:keydown={(e) => e.key === 'Enter' && copyToClipboard(acc.email, $t('emailCopiedToast'))} title={$t('copyEmail')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
        </span>
      {/if}

      {#if acc.url}
        <span class="control url-link" role="button" tabindex="0" on:click|stopPropagation={() => openUrlBlock(acc.url)} on:keydown={(e) => e.key === 'Enter' && openUrlBlock(acc.url)} title={$t('openUrl')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
        </span>
      {/if}
    </div>

    <div class="right-info">
      {#if acc.login_method !== 'google'}
        {#if decryptedPassword}
          <span class="unlocked-pw mono-password" role="button" tabindex="0" on:click|stopPropagation={() => copyToClipboard(decryptedPassword, $t('passwordCopied'))} on:keydown={(e) => e.key === 'Enter' && copyToClipboard(decryptedPassword, $t('passwordCopied'))}>{decryptedPassword}</span>
        {:else}
          <span class="dot-pw mono-password">●●●●●●●●</span>
        {/if}
        
        <!-- Eye icon: show/hide password -->
        <span class="control" class:active={decryptedPassword} role="button" tabindex="0" on:click|stopPropagation={showPassword} on:keydown={(e) => e.key === 'Enter' && showPassword()} title={decryptedPassword ? $t('hidePassword') : $t('showPassword')}>
          {#if decryptedPassword}
            <!-- Eye Off Icon -->
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
          {:else}
            <!-- Eye Icon -->
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
          {/if}
        </span>
        <!-- Clipboard icon: copy password -->
        <span class="control" role="button" tabindex="0" on:click|stopPropagation={copyPasswordSilent} on:keydown={(e) => e.key === 'Enter' && copyPasswordSilent()} title={$t('copyPassword')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
        </span>
      {/if}
      
      {#if isRecycled}
        <!-- Recycle/restore icon -->
        <span class="control restore-ctrl" role="button" tabindex="0" on:click|stopPropagation={restoreAccount} on:keydown={(e) => e.key === 'Enter' && restoreAccount()} title={$t('restore')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>
        </span>
        <!-- Destroy icon -->
        <span class="control delete" role="button" tabindex="0" on:click|stopPropagation={destroyPermanently} on:keydown={(e) => e.key === 'Enter' && destroyPermanently()} title={$t('destroy')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18.36 6.64A9 9 0 0 1 20.77 15M5.64 17.36A9 9 0 0 1 3.23 9"/><line x1="1" y1="1" x2="23" y2="23"/><path d="M16.24 16.24L20 20M7.76 7.76L4 4"/></svg>
        </span>
      {:else}
        <!-- Plus icon: add sub-account (limitar a profundidad < 3 para max 4 niveles) -->
        {#if depth < 3}
          <span class="control add-ctrl" role="button" tabindex="0" on:click|stopPropagation={() => showAddChildForm = true} on:keydown={(e) => e.key === 'Enter' && (showAddChildForm = true)} title={$t('addChild')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          </span>
        {/if}
        <!-- Edit icon: pencil -->
        <span class="control edit-ctrl" role="button" tabindex="0" on:click|stopPropagation={() => showEditForm = true} on:keydown={(e) => e.key === 'Enter' && (showEditForm = true)} title={$t('editAccount')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
        </span>
        <!-- Trash icon: delete -->
        <span class="control delete" role="button" tabindex="0" on:click|stopPropagation={deleteAccount} on:keydown={(e) => e.key === 'Enter' && deleteAccount()} title={$t('deleteAccount')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
        </span>
      {/if}
      {#if acc.notes_enc && acc.notes_enc.length > 0}
        <span class="control notes-ctrl" class:active={showNotes} role="button" tabindex="0" on:click|stopPropagation={toggleNotes} on:keydown={(e) => e.key === 'Enter' && toggleNotes()} title={$t('notesToggle')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
        </span>
      {/if}

      <span class="badge-method">
        {acc.login_method === 'google' ? $t('badgeExterna') : acc.login_method === 'both' ? $t('badgeBoth') : $t('badgePassword')}
      </span>
    </div>
  </div>

  {#if showNotes && decryptedNotes}
    <div class="notes-block">
      <pre class="notes-content">{decryptedNotes}</pre>
    </div>
  {/if}

  <div class="last-viewed-info">
    {$t('lastViewed')}: {acc.last_viewed ? acc.last_viewed : $t('neverViewed')}
  </div>

  {#if toastMessage}
    <div class="toast-float">{toastMessage}</div>
  {/if}
</div>

{#if expanded}
  <div transition:slide|global={{ duration: 100 }}>
    {#if loadingKids}
      <div class="loader" style="margin-left: {(depth+1) * 25}px;">{$t('loadingChildren')}</div>
    {:else if children.length > 0}
      <div class="children-container">
        {#each children as childAcc}
          <!-- Capturamos el refresh de los hijos para actualizar solo nuestra rama local, sin colapsar el árbol entero -->
          <svelte:self acc={childAcc} depth={depth + 1} isRecycled={isRecycled} on:refresh={reloadChildren} />
        {/each}
      </div>
    {/if}
  </div>
{/if}

{#if showAddChildForm}
  <AccountForm parentID={acc.id} parentNode={acc} on:cancel={() => showAddChildForm = false} on:saved={handleChildSaved} />
{/if}

{#if showEditForm}
  <AccountForm editNode={acc} on:cancel={() => showEditForm = false} on:saved={handleEditSaved} />
{/if}

{#if showConfirmDialog}
  <div class="confirm-overlay">
    <div class="confirm-box">
      <p class="confirm-msg">{confirmMessage}</p>
      <div class="confirm-actions">
        <button class="btn-confirm-no" on:click={confirmNo}>
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          {$t('btnCancel')}
        </button>
        <button class="btn-confirm-yes" on:click={confirmYes}>
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
          {$t('btnConfirmYes')}
        </button>
      </div>
    </div>
  </div>
{/if}
{/if}

<style>
  .account-card { margin-top: 6px; position:relative; }
  .card-content {
    background-color: #252830; border: 1px solid rgba(255,255,255,0.06); border-radius: 10px;
    padding: 12px 16px; display: flex; justify-content: space-between; align-items: center;
    transition: all 0.2s ease;
    box-shadow: 0 2px 8px rgba(0,0,0,0.15);
  }
  .card-content:hover { 
    background-color: #2d3039; 
    cursor: pointer;
    border-color: rgba(59,130,246,0.2);
    box-shadow: 0 4px 12px rgba(0,0,0,0.2);
  }
  
  .main-info { display: flex; align-items: center; gap: 10px; }
  .icon { 
    width: 16px; 
    height: 16px;
    color: #64748b; 
    transition: transform 0.2s; 
    display: inline-flex; 
    align-items: center;
    justify-content: center;
  }
  .icon.rotated { transform: rotate(90deg); }
  .empty-icon { color: #475569; cursor: default;}
  
  .title { font-weight: 600; font-size: 14px; color: #f1f5f9;}
  .tag-badge { background: #3b82f6; color: white; padding: 2px 5px; border-radius: 4px; font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; }
  .username { color: #38bdf8; font-size: 13px; font-weight: 500; cursor: pointer; transition: color 0.1s;}
  .username:hover { color: #7dd3fc; text-decoration: underline; }
  .url-link:hover { color: #3b82f6; }
  .email { color: #94a3b8; font-size: 13px;}
  
  .right-info { display: flex; align-items: center; gap: 10px; }
  .dot-pw { font-weight: bold; letter-spacing: 2px; color: #64748b; font-size: 12px;}
  .unlocked-pw { 
    font-family: 'Consolas', 'Courier New', monospace; 
    font-size: 13px; 
    background: rgba(34,197,94,0.12); 
    padding: 3px 8px; 
    border-radius: 6px; 
    border: 1px solid rgba(34,197,94,0.3); 
    color: #4ade80;
    cursor:pointer;
  }
  
  .control { 
    cursor: pointer; 
    opacity: 0.45; 
    transition: all 0.2s;
    display: inline-flex;
    align-items: center;
    color: #94a3b8;
  }
  .control:hover { opacity: 1; color: #e2e8f0; }
  .control.active { opacity: 1; color: #38bdf8; } /* Color vibrante cuando está viendo info */
  .control.add-ctrl:hover { color: #22c55e; }
  .control.edit-ctrl:hover { color: #f59e0b; }
  .control.restore-ctrl:hover { color: #22c55e; }
  .control.delete:hover { color: #ef4444; }
  
  .badge-method { 
    background: linear-gradient(135deg, #16a34a, #15803d); 
    padding: 3px 10px; 
    border-radius: 20px; 
    font-size: 11px; 
    font-weight: 600; 
    color: white;
    letter-spacing: 0.3px;
  }
  .control.notes-ctrl:hover { color: #a78bfa; }
  .notes-block {
    background: #1e2028; border: 1px solid rgba(139,92,246,0.2); border-radius: 8px;
    margin: 6px 0 0 30px; padding: 10px 14px;
    animation: fadeIn 0.2s ease-out;
  }
  .notes-content {
    font-family: 'Consolas', 'Courier New', monospace; font-size: 12px;
    color: #c4b5fd; margin: 0; white-space: pre-wrap; word-break: break-word;
    line-height: 1.6;
  }
  .last-viewed-info {
    font-size: 10px; color: #64748b; margin-top: 6px; padding-left: 30px;
    font-style: italic;
  }
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .children-container { 
    margin-top: 4px; 
    margin-left: 23px; /* Alineado exacto debajo del centro del icono (16px padding + 8px mitad icon minus mitad borde) */
    padding-left: 14px; /* Separación del hilo vertical hacia la sub-tarjeta */
    border-left: 2px solid rgba(255,255,255,0.06);
    display: flex;
    flex-direction: column;
    padding-bottom: 2px;
  }
  .loader { color: #3b82f6; font-size: 12px; padding: 5px; margin-left: 23px; }

  /* Tooltip Flotante de Copiado Animado */
  .toast-float {
    position: absolute; top: -30px; left: 50%;
    background: linear-gradient(135deg, #16a34a, #15803d); 
    color: white; padding: 4px 12px;
    border-radius: 6px; font-size: 11px; font-weight: 600;
    pointer-events: none;
    animation: slideUpFade 1.2s forwards;
    z-index: 100;
    box-shadow: 0 4px 12px rgba(34,197,94,0.3);
  }
  @keyframes slideUpFade {
    0% { opacity: 0; transform: translateY(10px); }
    15% { opacity: 1; transform: translateY(0); }
    80% { opacity: 1; transform: translateY(-5px); }
    100% { opacity: 0; transform: translateY(-10px); }
  }

  /* Custom Confirm Dialog */
  .confirm-overlay {
    position: fixed; top: 0; left: 0; width: 100%; height: 100%;
    background: rgba(15, 17, 21, 0.85); display: flex; justify-content: center; align-items: center;
    z-index: 3000;
    backdrop-filter: blur(4px);
  }
  .confirm-box {
    background: #1e2028; border: 1px solid rgba(255,255,255,0.1); border-radius: 14px;
    padding: 24px 28px; width: 360px; text-align: center;
    box-shadow: 0 20px 60px rgba(0,0,0,0.5);
  }
  .confirm-msg {
    color: #f1f5f9; font-size: 15px; font-weight: 600; margin: 0 0 20px 0;
    line-height: 1.5;
  }
  .confirm-actions {
    display: flex; justify-content: center; gap: 12px;
  }
  .confirm-actions button {
    padding: 9px 18px; border-radius: 8px; border: none; cursor: pointer;
    font-weight: 600; font-size: 13px; display: flex; align-items: center; gap: 6px;
    transition: all 0.2s;
  }
  .btn-confirm-no {
    background: #374151; color: #d1d5db;
  }
  .btn-confirm-no:hover { background: #4b5563; }
  .btn-confirm-yes {
    background: linear-gradient(135deg, #ef4444, #dc2626); color: white;
    box-shadow: 0 2px 8px rgba(239,68,68,0.3);
  }
  .btn-confirm-yes:hover {
    background: linear-gradient(135deg, #dc2626, #b91c1c);
    transform: translateY(-1px);
  }
</style>
