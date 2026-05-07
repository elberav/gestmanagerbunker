<script>
  import { onMount } from 'svelte';
  import { Call_GetAllFiles, Call_UploadFile, Call_UpdateFile, Call_DownloadFile, Call_DeleteSecureFile, Call_GetAllAccountsForDropdown, Call_SaveFileToDisk } from '../wailsjs/go/main/App.js';
  import { t } from './i18n.js';

  let files = [];
  let accounts = [];
  let loading = true;
  let toastMessage = "";
  let fileSearchQuery = "";

  // Modal / Form state
  let showForm = false;
  let isEditing = false;
  let editingId = null;
  let showDeleteConfirm = false;
  let deleteTargetId = null;
  
  let formFile = null;
  let formAccountId = "null"; // string "null" for unassigned
  let formTag = "";
  let formComment = "";
  let isUploading = false;

  onMount(async () => {
    await loadData();
  });

  async function loadData() {
    loading = true;
    try {
      let fRes = await Call_GetAllFiles();
      files = fRes || [];
      
      let aRes = await Call_GetAllAccountsForDropdown();
      accounts = aRes || [];
    } catch (e) {
      toastMessage = "Error loading data"; setTimeout(() => toastMessage = '', 2000);
    } finally {
      loading = false;
    }
  }

  function openNewForm() {
    isEditing = false;
    editingId = null;
    formFile = null;
    formAccountId = "null";
    formTag = "";
    formComment = "";
    showForm = true;
  }

  function openEditForm(f) {
    isEditing = true;
    editingId = f.id;
    formFile = null; // No need to re-upload file, just editing metadata
    formAccountId = f.account_id ? f.account_id.toString() : "null";
    formTag = f.tag || "";
    formComment = f.comment || "";
    showForm = true;
  }

  function closeForm() {
    showForm = false;
  }

  async function saveForm() {
    if (!isEditing && !formFile) {
      toastMessage = $t('noFileSelected') || 'Seleccione un archivo'; 
      setTimeout(() => toastMessage = '', 2000);
      return;
    }

    let accIdParam = formAccountId === "null" ? null : parseInt(formAccountId);
    isUploading = true;

    try {
      if (!isEditing) {
        // Upload new file
        await new Promise((resolve, reject) => {
          let reader = new FileReader();
          reader.onload = async (ev) => {
            let b64 = String(ev.target.result).split('base64,')[1];
            let err = await Call_UploadFile(accIdParam, formFile.name, b64, formTag, formComment);
            if (err !== "") reject(err);
            else resolve();
          };
          reader.onerror = () => reject("Lectura fallida");
          reader.readAsDataURL(formFile);
        });
      } else {
        // Edit existing file metadata
        await Call_UpdateFile(editingId, accIdParam, formTag, formComment);
      }
      
      await loadData();
      closeForm();
    } catch (e) {
      toastMessage = e; 
      setTimeout(() => toastMessage = '', 3000);
    } finally {
      isUploading = false;
    }
  }

  function handleFileSelect(e) {
    let f = e.target.files[0];
    if (!f) return;
    if (f.size > 5 * 1024 * 1024) {
      toastMessage = "Max 5MB"; setTimeout(() => toastMessage = '', 2000);
      e.target.value = "";
      formFile = null;
    } else {
      formFile = f;
    }
  }

  async function handleDownload(id) {
    let res = await Call_SaveFileToDisk(id);
    if (res === "SUCCESS") {
      toastMessage = $t('fileDownloaded') || "Archivo guardado"; 
      setTimeout(() => toastMessage = '', 2000);
    } else if (res !== "") {
      toastMessage = res; 
      setTimeout(() => toastMessage = '', 3000);
    }
  }

  function handleDelete(id) {
    deleteTargetId = id;
    showDeleteConfirm = true;
  }

  async function confirmDelete() {
    if (!deleteTargetId) return;
    await Call_DeleteSecureFile(deleteTargetId);
    showDeleteConfirm = false;
    deleteTargetId = null;
    await loadData();
  }

  function formatSize(bytes) {
    if (bytes >= 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + " " + $t('fileSizeMB');
    return (bytes / 1024).toFixed(0) + " " + $t('fileSizeKB');
  }

  $: filteredFiles = files.filter(f => {
    if (!fileSearchQuery) return true;
    const q = fileSearchQuery.toLowerCase();
    const nameMatch = f.filename && f.filename.toLowerCase().includes(q);
    const tagMatch = f.tag && f.tag.toLowerCase().includes(q);
    const commentMatch = f.comment && f.comment.toLowerCase().includes(q);
    const accMatch = (f.account_name && f.account_name.toLowerCase().includes(q)) || (f.account_email && f.account_email.toLowerCase().includes(q));
    return nameMatch || tagMatch || commentMatch || accMatch;
  });

</script>

<div class="vault-container">
  
  <div class="vault-toolbar">
    <div class="search-wrapper">
      <svg class="search-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
      <input 
        type="text" 
        placeholder={$t('searchFiles') || "Buscar por archivo, etiqueta, comentario o cuenta..."} 
        class="search-box"
        bind:value={fileSearchQuery}
      />
    </div>
    
    <button class="btn-new-file" on:click={openNewForm}>
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
      {$t('btnUploadGlobal') || 'Nuevo Archivo'}
    </button>
  </div>

  {#if loading}
    <div class="loader-container">
      <div class="loader">{$t('vaultLoading')}</div>
    </div>
  {:else if filteredFiles.length === 0}
    <div class="empty-state">
      <div class="thinking-folder-container">
        <svg class="folder-face-glow" xmlns="http://www.w3.org/2000/svg" width="72" height="72" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <!-- Folder Outline with Gradient -->
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" stroke="url(#gold-grad)" fill="rgba(251, 191, 36, 0.05)"/>
          <circle cx="9" cy="13" r="1.2" fill="#fbbf24"/>
          <circle cx="15" cy="13" r="1.2" fill="#fbbf24"/>
          {#if !fileSearchQuery}
            <path class="smile-sad" d="M10.5 16.5 Q12 15 13.5 16.5" stroke="#f59e0b" fill="none" stroke-linecap="round" />
          {/if}
          <defs>
            <linearGradient id="gold-grad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" style="stop-color:#f59e0b;stop-opacity:1" />
              <stop offset="100%" style="stop-color:#fbbf24;stop-opacity:1" />
            </linearGradient>
          </defs>
        </svg>
        <p>{fileSearchQuery ? $t('noSearchMatch') : $t('emptyVault')}</p>
      </div>
    </div>
  {:else}
    <div class="files-list">
      {#each filteredFiles as f}
        <div class="file-card">
          <div class="card-left">
            <span class="file-icon">
              <!-- Doc Icon -->
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
            </span>
            <div class="file-info-group">
              <div class="file-header-row">
                <span class="filename" role="button" tabindex="0" on:click={() => handleDownload(f.id)} on:keydown={(e) => e.key === 'Enter' && handleDownload(f.id)}>{f.filename}</span>
                <span class="filesize">{formatSize(f.file_size)}</span>
              </div>
              
              <div class="meta-row">
                {#if f.tag}
                  <span class="tag-badge">{f.tag}</span>
                {/if}
                {#if f.account_name}
                  <span class="linked-acc">🔗 {f.account_name} ({f.account_email})</span>
                {/if}
                <span class="date">{f.uploaded_at.substring(0,10)}</span>
              </div>

              {#if f.comment}
                <div class="comment-row">
                  "{f.comment}"
                </div>
              {/if}
            </div>
          </div>

          <div class="card-right">
            <button class="icon-btn edit" on:click={() => openEditForm(f)} title={$t('editAccount')}>
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
            </button>
            <button class="icon-btn danger" on:click={() => handleDelete(f.id)} title={$t('deleteFile')}>
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if showForm}
  <div class="modal-overlay">
    <div class="modal">
      <h2>{isEditing ? $t('editAccount') : $t('btnUploadGlobal')}</h2>
      
      {#if !isEditing}
        <div class="form-group">
          <label for="vault-f-file">{$t('uploadFile')}</label>
          <input id="vault-f-file" type="file" on:change={handleFileSelect}>
        </div>
      {/if}

      <div class="form-group">
        <label for="vault-f-acc">{$t('addChild')}</label>
        <select id="vault-f-acc" bind:value={formAccountId}>
          <option value="null">-- {$t('neverViewed')} / Suelto --</option>
          {#each accounts as a}
            <option value={a.id.toString()}>{a.name} ({a.email})</option>
          {/each}
        </select>
      </div>

      <div class="form-group">
        <label for="vault-f-tag">{$t('tagPlaceholder')}</label>
        <input id="vault-f-tag" type="text" placeholder="Ej. FINANZAS / PASAPORTE / BACKUP" bind:value={formTag}>
      </div>

      <div class="form-group">
        <label for="vault-f-comment">{$t('notesPlaceholder')}</label>
        <textarea id="vault-f-comment" rows="3" placeholder="Información adicional..." bind:value={formComment}></textarea>
      </div>

      <div class="modal-actions">
        <button class="btn-cancel" on:click={closeForm}>{$t('btnCancel')}</button>
        <button class="btn-save" on:click={saveForm} disabled={isUploading}>
          {isUploading ? '...' : $t('btnSave')}
        </button>
      </div>
    </div>
  </div>
{/if}

{#if showDeleteConfirm}
  <div class="modal-overlay">
    <div class="modal small-modal">
      <div class="confirm-icon-danger">
        <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#ef4444" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
      </div>
      <h3>{$t('confirmDelFile')}</h3>
      <p style="color: #94a3b8; font-size: 13px; margin: 10px 0 20px 0; text-align: center;">Esta acción eliminará el archivo de la base de datos para siempre.</p>
      
      <div class="modal-actions">
        <button class="btn-cancel" on:click={() => showDeleteConfirm = false}>{$t('btnCancel')}</button>
        <button class="btn-danger-confirm" on:click={confirmDelete}>{$t('destroy')}</button>
      </div>
    </div>
  </div>
{/if}

{#if toastMessage}
  <div class="toast-float">{toastMessage}</div>
{/if}

<style>
  .vault-container {
    padding: 20px 0;
    animation: fadeIn 0.3s ease;
  }
  
  .vault-toolbar {
    display: flex; gap: 15px; margin-bottom: 20px;
    align-items: center; justify-content: space-between;
  }

  .search-wrapper {
    position: relative; flex-grow: 1; max-width: 500px;
  }

  .search-icon {
    position: absolute; left: 14px; top: 12px; color: #64748b;
  }

  .search-box {
    width: 100%; border: 1px solid rgba(255,255,255,0.06); border-radius: 8px;
    background-color: #1e1e24; color: #f1f5f9; padding: 10px 15px 10px 42px;
    font-size: 14px; transition: border-color 0.2s, box-shadow 0.2s;
  }

  .search-box:focus {
    outline: none; border-color: #facc15; box-shadow: 0 0 0 2px rgba(250,204,21,0.2);
  }

  .btn-new-file {
    background: #facc15; color: #000; border: none; border-radius: 8px;
    padding: 10px 16px; font-weight: 600; font-size: 14px; cursor: pointer;
    display: flex; align-items: center; gap: 8px; transition: opacity 0.2s;
  }
  .btn-new-file:hover { opacity: 0.9; }

  .empty-state {
    display: flex; flex-direction: column; align-items: center; justify-content: center;
    padding: 80px 0; gap: 15px;
  }
  .empty-state p { margin: 0; font-weight: 600; color: #94a3b8; font-size: 15px; }

  .thinking-folder-container {
    display: flex; flex-direction: column; align-items: center; gap: 12px;
  }
  .folder-face-glow {
    filter: drop-shadow(0 0 15px rgba(251, 191, 36, 0.2));
  }
  .smile-sad {
    filter: drop-shadow(0 0 2px rgba(245, 158, 11, 0.3));
  }

  .files-list {
    display: flex; flex-direction: column; gap: 8px;
  }

  .file-card {
    background-color: #252830; border: 1px solid rgba(255,255,255,0.06); border-radius: 10px;
    padding: 14px 16px; display: flex; justify-content: space-between; align-items: center;
    transition: all 0.2s ease;
  }
  .file-card:hover {
    background-color: #2d3039; border-color: rgba(250,204,21,0.3);
  }

  .card-left {
    display: flex; align-items: flex-start; gap: 14px; flex-grow: 1; overflow: hidden;
  }

  .file-icon {
    color: #facc15; background: rgba(250,204,21,0.1); padding: 10px; border-radius: 8px;
    display: flex; align-items: center; justify-content: center;
  }

  .file-info-group {
    display: flex; flex-direction: column; gap: 4px; overflow: hidden; justify-content: center; height: 100%;
  }

  .file-header-row {
    display: flex; align-items: baseline; gap: 10px;
  }

  .filename {
    color: #f1f5f9; font-weight: 600; font-size: 15px; cursor: pointer; transition: color 0.1s;
  }
  .filename:hover { color: #facc15; text-decoration: underline; }

  .filesize {
    color: #64748b; font-size: 12px;
  }

  .meta-row {
    display: flex; align-items: center; gap: 8px; font-size: 12px; flex-wrap: wrap;
  }

  .tag-badge {
    background: #3b82f6; color: white; padding: 2px 6px; border-radius: 4px; font-weight: 700;
  }

  .linked-acc {
    color: #cbd5e1; background: rgba(255,255,255,0.05); padding: 2px 6px; border-radius: 4px;
  }

  .date { color: #64748b; }

  .comment-row {
    color: #94a3b8; font-size: 13px; font-style: italic; margin-top: 4px;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 90%;
  }

  .card-right {
    display: flex; gap: 8px;
  }

  .icon-btn {
    background: none; border: none; cursor: pointer; padding: 6px; border-radius: 4px;
    display: flex; align-items: center; justify-content: center; transition: all 0.2s;
  }
  .icon-btn.edit { color: #94a3b8; }
  .icon-btn.edit:hover { background: rgba(255,255,255,0.1); color: #fff; }
  
  .icon-btn.danger { color: #ef4444; }
  .icon-btn.danger:hover { background: rgba(239, 68, 68, 0.1); }

  /* Modal Form */
  .modal-overlay {
    position: fixed; top: 0; left: 0; width: 100vw; height: 100vh;
    background: rgba(0,0,0,0.6); backdrop-filter: blur(2px);
    display: flex; justify-content: center; align-items: center; z-index: 1000;
    animation: fadeIn 0.2s ease;
  }
  .modal {
    background: #1e1e24; border: 1px solid #333; border-radius: 12px;
    padding: 24px; width: 450px; max-width: 90%;
  }
  .modal h2 { margin: 0 0 20px 0; color: #facc15; font-size: 18px; }
  .form-group { margin-bottom: 16px; display: flex; flex-direction: column; gap: 6px; }
  .form-group label { color: #cbd5e1; font-size: 13px; font-weight: 500; }
  .form-group input[type="text"], .form-group select, .form-group textarea {
    background: #121212; border: 1px solid #333; border-radius: 6px; color: #fff;
    padding: 10px; font-family: inherit; font-size: 14px; width: 100%; box-sizing: border-box;
  }
  .form-group input[type="file"] {
    background: #121212; border: 1px dashed #444; border-radius: 6px; color: #fff;
    padding: 10px; box-sizing: border-box;
  }
  .modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 24px; }
  .btn-cancel, .btn-save { padding: 10px 16px; border-radius: 6px; border: none; cursor: pointer; font-weight: 600; }
  .btn-cancel { background: transparent; color: #94a3b8; border: 1px solid #333; }
  .btn-cancel:hover { background: #333; }
  .btn-save { background: #facc15; color: #000; }
  .btn-save:hover { opacity: 0.9; }
  .btn-save:disabled { opacity: 0.5; cursor: not-allowed; }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
  .btn-danger-confirm {
    background: #ef4444; color: white; border: none; border-radius: 8px;
    padding: 10px 20px; font-weight: 600; font-size: 14px; cursor: pointer;
    transition: all 0.2s;
  }
  .btn-danger-confirm:hover { background: #dc2626; box-shadow: 0 4px 12px rgba(239, 68, 68, 0.3); }

  .small-modal { max-width: 380px !important; display: flex; flex-direction: column; align-items: center; }
  .confirm-icon-danger {
    background: rgba(239, 68, 68, 0.1); padding: 15px; border-radius: 50%; margin-bottom: 15px;
  }
</style>
