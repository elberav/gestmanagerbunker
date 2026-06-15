<script>
  import { Call_AddAccount, Call_UpdateAccount, Call_Decrypt } from '../wailsjs/go/main/App.js';
  import { createEventDispatcher, onMount } from 'svelte';
  import { t } from './i18n.js';

  const dispatch = createEventDispatcher();
  
  /** @type {number | null} */
  export let parentID = null; 
  /** @type {{ id: number, name: string, email: string, login_method: string, tag?: string, username?: string, url?: string, password_enc?: number[], notes_enc?: number[] } | null} */
  export let parentNode = null;
  /** @type {{ id: number, name: string, email: string, login_method: string, tag?: string, username?: string, url?: string, password_enc?: number[], notes_enc?: number[] } | null} */
  export let editNode = null;

  let name = '';
  let email = '';
  let loginMethod = 'password';
  let password = '';
  // Nuevos campos opcionales
  let tag = '';
  let username = '';
  let url = '';
  let notes = '';
  let loading = false;
  let showPassword = false;
  let passwordLength = 16;

  function generatePassword() {
    const charset = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%&*_-+=?';
    const maxValid = 256 - (256 % charset.length);
    let result = '';
    while (result.length < passwordLength) {
      const batch = crypto.getRandomValues(new Uint8Array(passwordLength * 2));
      for (const v of batch) {
        if (v < maxValid) {
          result += charset[v % charset.length];
          if (result.length >= passwordLength) break;
        }
      }
    }
    password = result;
    showPassword = true;
  }

  onMount(async () => {
    if (editNode) {
      name = editNode.name || '';
      email = editNode.email || '';
      tag = editNode.tag || '';
      username = editNode.username || '';
      url = editNode.url || '';
      loginMethod = editNode.login_method || 'password';
      if (editNode.notes_enc) {
        try { notes = await Call_Decrypt(editNode.notes_enc); } catch (e) { console.error('Error decrypting notes', e); }
      }
      // Desencriptar la contraseña actual para que el usuario la vea y edite
      if (editNode.password_enc) {
        try {
          password = await Call_Decrypt(editNode.password_enc);
        } catch (e) {
          console.error($t('errorDecryptEdit'), e);
        }
      }
    } else if (parentNode) {
      // Heredar método pero NO desencriptar la clave del padre automáticamente (Seguridad)
      if (parentNode.login_method) {
        loginMethod = parentNode.login_method;
      }
    }
  });

  async function handleSave() {
    if (!name || (isNaN(parentID) && parentID !== null)) return;
    
    // Si el método requiere contraseña, no permitir guardar sin una
    if (loginMethod !== 'google' && !password.trim()) return;
    
    loading = true;
    
    // Purge password if method is external-only
    if (loginMethod === 'google') {
      password = '';
    }

    try {
      if (editNode) {
        await Call_UpdateAccount(editNode.id, name, email, loginMethod, password, tag, username, url, notes);
      } else {
        let parentPtr = parentID !== null ? parseInt(String(parentID), 10) : null;
        await Call_AddAccount(name, email, loginMethod, password, parentPtr, tag, username, url, notes);
      }
      dispatch('saved'); 
    } catch (e) {
      console.error($t('errorSaving'), e);
    }
    loading = false;
  }
</script>

<div class="modal">
  <div class="modal-box">
    <h3>
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        {#if editNode}
          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
        {:else}
          <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/>
        {/if}
      </svg>
      {#if editNode}
        {$t('formEditTitle')}
      {:else if parentID}
        {$t('formAddSub')}
      {:else}
        {$t('formAddMain')}
      {/if}
    </h3>
    
    <input type="text" placeholder={$t('namePlaceholder')} bind:value={name} class="in-box" />
    <input type="email" placeholder={$t('emailPlaceholder')} bind:value={email} class="in-box" />
    
    <!-- Campos Opcionales -->
    <div class="row">
      <input type="text" placeholder={$t('tagPlaceholder')} bind:value={tag} class="in-box" />
      <input type="text" placeholder={$t('usernamePlaceholder')} bind:value={username} class="in-box" />
    </div>
    <input type="text" placeholder={$t('urlPlaceholder')} bind:value={url} class="in-box url-box" />

    <select bind:value={loginMethod} class="in-box">
      <option value="password">{$t('loginMethodPassword')}</option>
      <option value="google">{$t('loginMethodGoogle')}</option>
      <option value="both">{$t('loginMethodBoth')}</option>
    </select>

    {#if loginMethod !== 'google'}
      <div class="pw-wrapper">
        {#if showPassword}
          <input type="text" placeholder={editNode ? $t('passwordFormPlaceholder_edit') : $t('passwordFormPlaceholder_new')} bind:value={password} class="in-box mono-password" />
        {:else}
          <input type="password" placeholder={editNode ? $t('passwordFormPlaceholder_edit') : $t('passwordFormPlaceholder_new')} bind:value={password} class="in-box mono-password" />
        {/if}
        <span class="pw-toggle" role="button" tabindex="0" on:click={() => showPassword = !showPassword} on:keydown={(e) => e.key === 'Enter' && (showPassword = !showPassword)} title={showPassword ? $t('hidePassword') : $t('showPassword')}>
          {#if showPassword}
            <!-- Eye Off Icon -->
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
          {:else}
            <!-- Eye Icon -->
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
          {/if}
        </span>
      </div>
      <div class="gen-row">
        <select bind:value={passwordLength} class="gen-length">
          <option value={12}>12</option>
          <option value={16}>16</option>
          <option value={20}>20</option>
          <option value={24}>24</option>
          <option value={32}>32</option>
        </select>
        <button type="button" class="btn-generate" on:click={generatePassword}>
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="M7 15h0M2 9.5h20"/></svg>
          {$t('btnGenerate')}
        </button>
      </div>
    {/if}

    <textarea placeholder={$t('notesPlaceholder')} bind:value={notes} class="in-box notes-box" rows="3"></textarea>

    <div class="actions">
      <button on:click={() => dispatch('cancel')} class="btn-cancel">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        {$t('btnCancel')}
      </button>
      <button on:click={handleSave} class="btn-save" disabled={!name || loading || (loginMethod !== 'google' && !password.trim())}>
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
        {editNode ? $t('btnUpdate') : $t('btnSave')}
      </button>
    </div>
  </div>
</div>

<style>
  .modal {
    position: fixed; top: 0; left: 0; width: 100%; height: 100%;
    background: rgba(15, 17, 21, 0.85); display: flex; justify-content: center; align-items: center;
    z-index: 2000;
    backdrop-filter: blur(4px);
  }
  .modal-box {
    background: #1e2028; border: 1px solid rgba(255,255,255,0.08); border-radius: 14px;
    padding: 28px; width: 370px; text-align: left;
    box-shadow: 0 20px 60px rgba(0,0,0,0.4);
  }
  h3 { 
    margin-top: 0; 
    font-size: 18px; 
    color: #f8fafc; 
    border-bottom: 1px solid rgba(255,255,255,0.08); 
    padding-bottom: 12px;
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 700;
  }
  h3 svg { color: #3b82f6; }
  .in-box {
    width: 100%; padding: 11px 14px; margin-bottom: 12px; border-radius: 8px; 
    border: 1px solid rgba(255,255,255,0.1); 
    background: #252830; color: #e2e8f0; box-sizing: border-box;
    font-size: 14px;
    transition: border-color 0.2s;
  }
  .row { display: flex; gap: 8px; width: 100%; }
  .in-box:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59,130,246,0.12);
  }
  .pw-wrapper {
    position: relative;
    width: 100%;
  }
  .mono-password {
    font-family: 'Consolas', 'Courier New', monospace;
    letter-spacing: 1px;
  }
  .pw-wrapper .in-box {
    padding-right: 44px;
  }
  .pw-toggle {
    position: absolute;
    right: 12px;
    top: 50%;
    transform: translateY(-50%);
    margin-top: -6px;
    cursor: pointer;
    color: #64748b;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 4px;
    border-radius: 4px;
    transition: color 0.2s, background 0.2s;
  }
  .pw-toggle:hover {
    color: #e2e8f0;
    background: rgba(255,255,255,0.06);
  }
  .actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px;}
  button { 
    padding: 9px 16px; 
    border-radius: 8px; 
    border: none; 
    cursor: pointer; 
    font-weight: 600;
    font-size: 13px;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all 0.2s;
  }
  .btn-cancel { 
    background: #374151; 
    color: #d1d5db;
  }
  .btn-cancel:hover { background: #4b5563; }
  .btn-save { 
    background: linear-gradient(135deg, #16a34a, #15803d); 
    color: white;
    box-shadow: 0 2px 8px rgba(22,163,74,0.3);
  }
  .btn-save:hover { 
    background: linear-gradient(135deg, #16a34a, #15803d);
    transform: translateY(-1px);
  }
  .btn-save:disabled { 
    background: #1f2937; 
    color: #6b7280; 
    cursor: not-allowed;
    box-shadow: none;
    transform: none;
  }

  .gen-row {
    display: flex; gap: 8px; margin-top: -4px; margin-bottom: 12px;
  }
  .notes-box {
    resize: vertical; min-height: 60px; max-height: 160px;
    font-family: 'Consolas', 'Courier New', monospace; font-size: 13px;
    line-height: 1.5; letter-spacing: 0.3px;
  }
  .gen-length {
    background: #252830; color: #94a3b8; border: 1px solid rgba(255,255,255,0.1);
    border-radius: 8px; padding: 6px 10px; font-size: 13px; font-weight: 600;
    cursor: pointer; transition: border-color 0.2s;
  }
  .gen-length:hover { border-color: #8b5cf6; color: #e2e8f0; }
  .btn-generate {
    flex: 1; background: linear-gradient(135deg, #8b5cf6, #7c3aed); color: white;
    border: none; border-radius: 8px; padding: 7px 14px; font-size: 13px;
    font-weight: 600; cursor: pointer; display: flex; align-items: center;
    justify-content: center; gap: 6px;
    box-shadow: 0 2px 8px rgba(139,92,246,0.3); transition: all 0.2s;
  }
  .btn-generate:hover {
    background: linear-gradient(135deg, #7c3aed, #6d28d9);
    box-shadow: 0 4px 12px rgba(139,92,246,0.4); transform: translateY(-1px);
  }

  /* Ocultar el ojo nativo de contraseña de Edge/Internet Explorer */
  input[type="password"]::-ms-reveal,
  input[type="password"]::-ms-clear {
    display: none;
  }
</style>
