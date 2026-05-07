<script>
  import {
    Call_IsFirstRun,
    Call_SetupMaster,
    Call_Unlock,
    Call_RecoverWithCode,
    Call_GetLockoutStatus,
  } from "../wailsjs/go/main/App.js";
  import { createEventDispatcher, onMount, onDestroy } from "svelte";
  import { t } from "./i18n.js";
  import logoImg from "./assets/images/logo-universal.png";

  const dispatch = createEventDispatcher();

  let isFirstRun = false;
  let password = "";
  let confirmPassword = "";
  let errorMsg = "";
  let loading = true;

  // Rate Limiting
  let failedAttempts = 0;
  let lockoutSeconds = 0;
  let lockoutInterval;

  async function checkLockout() {
    const [attempts, seconds] = await Call_GetLockoutStatus();
    failedAttempts = attempts;
    lockoutSeconds = seconds;

    if (lockoutSeconds > 0 && !lockoutInterval) {
      lockoutInterval = setInterval(() => {
        lockoutSeconds--;
        if (lockoutSeconds <= 0) {
          clearInterval(lockoutInterval);
          lockoutInterval = null;
        }
      }, 1000);
    }
  }

  onDestroy(() => {
    if (lockoutInterval) clearInterval(lockoutInterval);
  });

  // Recovery
  let mode = "login"; // 'login' | 'recovery' | 'showCode'
  let recoveryCodeInput = "";
  let newPassword = "";
  let confirmNewPassword = "";
  let recoveryCode = ""; // El código generado para mostrar
  let codeCopied = false;

  let reminderDisplay = "";

  onMount(async () => {
    isFirstRun = await Call_IsFirstRun();
    await checkLockout();
    loading = false;

    let reminderFull = $t("loginReminder");
    let i = 0;
    const typing = setInterval(() => {
      if (i < reminderFull.length) {
        reminderDisplay += reminderFull.charAt(i);
        i++;
      } else {
        clearInterval(typing);
      }
    }, 30);
  });

  async function handleSubmit() {
    if (!password) {
      errorMsg = $t("emptyPasswordError");
      return;
    }

    if (isFirstRun) {
      if (password !== confirmPassword) {
        errorMsg = $t("passwordMismatchError");
        return;
      }

      try {
        recoveryCode = await Call_SetupMaster(password);
        mode = "showCode"; // Mostrar el código de emergencia
        errorMsg = "";
      } catch (e) {
        errorMsg = $t("setupError");
      }
    } else {
      const unlocked = await Call_Unlock(password);
      if (unlocked) {
        dispatch("unlocked");
      } else {
        await checkLockout();
        errorMsg = $t("wrongPassword");
      }
    }
  }

  async function handleRecover() {
    if (!recoveryCodeInput || !newPassword) {
      errorMsg = $t("emptyPasswordError");
      return;
    }
    if (newPassword !== confirmNewPassword) {
      errorMsg = $t("passwordMismatchError");
      return;
    }

    try {
      recoveryCode = await Call_RecoverWithCode(recoveryCodeInput, newPassword);
      mode = "showCode"; // Mostrar el NUEVO código de emergencia
      errorMsg = "";
    } catch (e) {
      errorMsg = $t("recoveryError");
    }
  }

  function handleCodeSaved() {
    dispatch("unlocked");
  }

  function copyCode() {
    navigator.clipboard.writeText(recoveryCode);
    codeCopied = true;
    setTimeout(() => {
      codeCopied = false;
    }, 2000);
  }
</script>

<div class="overlay">
  <div class="dialog">
    <img src={logoImg} alt="Logo" class="vault-logo" />

    <!-- MODO: Mostrar código de recuperación -->
    {#if mode === "showCode"}
      <div class="shield-anim">
        <!-- Shield icon -->
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="40"
          height="40"
          viewBox="0 0 24 24"
          fill="none"
          class="shield-icon"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /></svg
        >
      </div>
      <h2 class="code-title">{$t("recoveryCodeTitle")}</h2>
      <p class="subtitle">{$t("recoveryCodeSubtitle")}</p>

      <div class="recovery-code-display pulse-anim">
        <span class="code-value">{recoveryCode}</span>
        <button
          class="copy-code-btn"
          class:copied={codeCopied}
          on:click={copyCode}
        >
          {#if codeCopied}
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"><path d="M20 6L9 17l-5-5" /></svg
            >
            <span class="tooltip">{$t("copied")}</span>
          {:else}
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><rect x="9" y="9" width="13" height="13" rx="2" ry="2" /><path
                d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
              /></svg
            >
          {/if}
        </button>
      </div>

      <button class="btn-saved" on:click={handleCodeSaved}>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline
            points="22 4 12 14.01 9 11.01"
          /></svg
        >
        {$t("btnCodeSaved")}
      </button>

      <!-- MODO: Recuperación con código -->
    {:else if mode === "recovery"}
      <div class="lock-icon" style="color: #f59e0b;">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="36"
          height="36"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><circle cx="12" cy="12" r="3" /><path
            d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"
          /></svg
        >
      </div>
      <h2>{$t("recoveryTitle")}</h2>
      <p class="subtitle">{$t("recoverySubtitle")}</p>

      {#if errorMsg}
        <div class="error">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><circle cx="12" cy="12" r="10" /><line
              x1="15"
              y1="9"
              x2="9"
              y2="15"
            /><line x1="9" y1="9" x2="15" y2="15" /></svg
          >
          {errorMsg}
        </div>
      {/if}

      <form on:submit|preventDefault={handleRecover}>
        <div class="input-wrapper">
          <svg
            class="input-icon"
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><path
              d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"
            /></svg
          >
          <input
            type="text"
            bind:value={recoveryCodeInput}
            placeholder={$t("recoveryCodePlaceholder")}
            autocomplete="off"
            class="mono-input"
          />
        </div>
        <div class="input-wrapper">
          <svg
            class="input-icon"
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path
              d="M7 11V7a5 5 0 0 1 10 0v4"
            /></svg
          >
          <input
            type="password"
            bind:value={newPassword}
            placeholder={$t("newPasswordPlaceholder")}
            autocomplete="off"
          />
        </div>
        <div class="input-wrapper">
          <svg
            class="input-icon"
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline
              points="22 4 12 14.01 9 11.01"
            /></svg
          >
          <input
            type="password"
            bind:value={confirmNewPassword}
            placeholder={$t("confirmNewPasswordPlaceholder")}
            autocomplete="off"
          />
        </div>
        <button type="submit">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><path
              d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"
            /></svg
          >
          {$t("btnRecover")}
        </button>
      </form>

      <button
        class="link-btn"
        on:click={() => {
          mode = "login";
          errorMsg = "";
        }}
      >
        ← {$t("btnBackToLogin")}
      </button>

      <!-- MODO: Login normal -->
    {:else}
      <div class="lock-icon">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="36"
          height="36"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path
            d="M7 11V7a5 5 0 0 1 10 0v4"
          /></svg
        >
      </div>
      <h2>{isFirstRun ? $t("loginTitle_setup") : $t("loginTitle_locked")}</h2>
      <p class="subtitle">
        {isFirstRun ? $t("loginSubtitle_setup") : $t("loginSubtitle_locked")}
      </p>

      <div class="typewriter-text">
        {reminderDisplay}<span class="cursor">|</span>
      </div>

      {#if errorMsg && lockoutSeconds <= 0}
        <div class="error">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><circle cx="12" cy="12" r="10" /><line
              x1="15"
              y1="9"
              x2="9"
              y2="15"
            /><line x1="9" y1="9" x2="15" y2="15" /></svg
          >
          {errorMsg}
        </div>
      {/if}

      {#if lockoutSeconds > 0}
        <div class="error" style="color: #f59e0b; background: rgba(245, 158, 11, 0.1); border-color: rgba(245, 158, 11, 0.2);">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          Demasiados intentos. Espera {Math.floor(lockoutSeconds / 60)}:{(lockoutSeconds % 60).toString().padStart(2, '0')}
        </div>
      {/if}

      <form on:submit|preventDefault={handleSubmit}>
        <div class="input-wrapper">
          <svg
            class="input-icon"
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path
              d="M7 11V7a5 5 0 0 1 10 0v4"
            /></svg
          >
          <input
            type="password"
            bind:value={password}
            placeholder={$t("passwordPlaceholder")}
            autocomplete="off"
            disabled={lockoutSeconds > 0}
          />
        </div>
        {#if isFirstRun}
          <div class="input-wrapper">
            <svg
              class="input-icon"
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline
                points="22 4 12 14.01 9 11.01"
              /></svg
            >
            <input
              type="password"
              bind:value={confirmPassword}
              placeholder={$t("confirmPasswordPlaceholder")}
              autocomplete="off"
              disabled={lockoutSeconds > 0}
            />
          </div>
        {/if}
        <button type="submit" disabled={lockoutSeconds > 0}>
          {#if isFirstRun}
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /></svg
            >
            {$t("btnCreate")}
          {:else}
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path
                d="M7 11V7a5 5 0 0 1 5-5 5 5 0 0 1 5 5"
              /></svg
            >
            {lockoutSeconds > 0 ? 'Bóveda Bloqueada' : $t("btnUnlock")}
          {/if}
        </button>
      </form>

      {#if !isFirstRun}
        <button
          class="link-btn"
          on:click={() => {
            mode = "recovery";
            errorMsg = "";
          }}
        >
          {$t("forgotPassword")}
        </button>
      {/if}
    {/if}
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(15, 17, 21, 0.97);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    backdrop-filter: blur(8px);
  }
  .dialog {
    background: #1e2028;
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 16px;
    padding: 36px 32px;
    width: 400px;
    text-align: center;
    box-shadow:
      0 20px 60px rgba(0, 0, 0, 0.5),
      0 0 40px rgba(59, 130, 246, 0.08);
  }
  .vault-logo {
    width: 64px;
    height: 64px;
    border-radius: 14px;
    margin-bottom: 12px;
    filter: drop-shadow(0 4px 12px rgba(0, 0, 0, 0.3));
  }
  .lock-icon {
    color: #3b82f6;
    margin-bottom: 12px;
    filter: drop-shadow(0 2px 8px rgba(59, 130, 246, 0.3));
  }
  h2 {
    margin: 0 0 8px 0;
    font-size: 24px;
    font-weight: 700;
    color: #f8fafc;
    letter-spacing: -0.3px;
  }
  .code-title {
    color: #e2e8f0;
  }
  .subtitle {
    font-size: 16px;
    color: #94a3b8;
    margin-bottom: 12px;
    line-height: 1.5;
  }
  .typewriter-text {
    font-size: 14px;
    color: #9bbffa;
    font-family: "Consolas", monospace;
    min-height: 40px;
    margin-bottom: 20px;
    padding: 6px;
    background: rgba(59, 130, 246, 0.05);
    border-radius: 7px;
    border: 1px dashed rgba(72, 59, 246, 0.527);
  }
  .cursor {
    animation: blink 1.5s step-end infinite;
    font-weight: bold;
  }
  @keyframes blink {
    60% {
      opacity: 0;
    }
  }

  /* ---- Animations & Recovery Styling ---- */
  .shield-anim {
    color: #94a3b8;
    margin-bottom: 12px;
    display: inline-flex;
    animation: floatShield 3s ease-in-out infinite;
  }
  @keyframes floatShield {
    0% {
      transform: translateY(0);
    }
    50% {
      transform: translateY(-4px);
      filter: drop-shadow(0 4px 6px rgba(148, 163, 184, 0.4));
    }
    100% {
      transform: translateY(0);
    }
  }

  .recovery-code-display {
    background: #1e293b;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 12px;
    padding: 18px 16px;
    margin: 16px 0 20px 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.2);
  }
  .pulse-anim {
    animation: subtlePulse 4s infinite alternate;
  }
  @keyframes subtlePulse {
    0% {
      border-color: rgba(255, 255, 255, 0.06);
      box-shadow: 0 0 0 rgba(56, 189, 248, 0);
    }
    100% {
      border-color: rgba(56, 189, 248, 0.2);
      box-shadow: 0 0 15px rgba(56, 189, 248, 0.1);
    }
  }

  .code-value {
    font-family: "Consolas", monospace;
    font-size: 17px;
    font-weight: 600;
    color: #38bdf8;
    letter-spacing: 1.5px;
    user-select: all;
    text-shadow: 0 0 8px rgba(56, 189, 248, 0.2);
  }
  .copy-code-btn {
    background: transparent;
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    padding: 8px;
    cursor: pointer;
    color: #94a3b8;
    display: flex;
    align-items: center;
    transition: all 0.2s;
    position: relative;
  }
  .copy-code-btn.copied {
    color: #38bdf8;
    background: rgba(56, 189, 248, 0.1);
    border-color: rgba(56, 189, 248, 0.3);
  }
  .copy-code-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: #f8fafc;
    transform: scale(1.05);
  }
  .copy-code-btn.copied:hover {
    background: rgba(56, 189, 248, 0.15);
    color: #38bdf8;
  }
  .tooltip {
    position: absolute;
    bottom: -32px;
    left: 50%;
    transform: translateX(-50%);
    background: #38bdf8;
    color: #0f1117;
    padding: 4px 8px;
    border-radius: 6px;
    font-size: 11px;
    font-weight: 700;
    white-space: nowrap;
    animation: fadeUp 0.2s ease-out forwards;
    pointer-events: none;
    box-shadow: 0 4px 12px rgba(56, 189, 248, 0.3);
  }
  .tooltip::after {
    content: "";
    position: absolute;
    top: -4px;
    left: 50%;
    transform: translateX(-50%);
    border-width: 0 4px 4px 4px;
    border-style: solid;
    border-color: transparent transparent #38bdf8 transparent;
  }
  @keyframes fadeUp {
    0% {
      opacity: 0;
      transform: translate(-50%, 4px);
    }
    100% {
      opacity: 1;
      transform: translate(-50%, 0);
    }
  }

  .btn-saved {
    width: 100%;
    padding: 13px;
    background: #334155;
    color: #e2e8f0;
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 10px;
    font-weight: 600;
    font-size: 15px;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }
  .btn-saved:hover {
    background: #475569;
    border-color: rgba(255, 255, 255, 0.15);
    transform: translateY(-1px);
  }

  .error {
    color: #ef4444;
    font-size: 13px;
    margin-bottom: 16px;
    font-weight: 600;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    background: rgba(239, 68, 68, 0.08);
    padding: 8px 12px;
    border-radius: 8px;
    border: 1px solid rgba(239, 68, 68, 0.15);
  }

  .input-wrapper {
    position: relative;
    margin-bottom: 14px;
  }
  .input-icon {
    position: absolute;
    left: 14px;
    top: 50%;
    transform: translateY(-50%);
    color: #64748b;
    pointer-events: none;
  }
  input {
    width: 100%;
    padding: 13px 14px 13px 40px;
    border-radius: 10px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    background: #252830;
    color: #e2e8f0;
    box-sizing: border-box;
    font-size: 15px;
    transition:
      border-color 0.2s,
      box-shadow 0.2s;
  }
  .mono-input {
    font-family: "Consolas", monospace;
    letter-spacing: 1.5px;
    text-transform: uppercase;
  }
  input:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.15);
  }

  button[type="submit"] {
    width: 100%;
    padding: 13px;
    background: linear-gradient(135deg, #3b82f6, #2563eb);
    color: white;
    border: none;
    border-radius: 10px;
    font-weight: 600;
    font-size: 15px;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    box-shadow: 0 4px 14px rgba(59, 130, 246, 0.3);
    margin-top: 6px;
  }
  button[type="submit"]:hover {
    background: linear-gradient(135deg, #2563eb, #1d4ed8);
    box-shadow: 0 6px 20px rgba(59, 130, 246, 0.4);
    transform: translateY(-1px);
  }
  button[type="submit"]:active {
    transform: translateY(0);
  }

  .link-btn {
    background: none;
    border: none;
    color: #64748b;
    font-size: 13px;
    cursor: pointer;
    margin-top: 16px;
    transition: color 0.2s;
    padding: 4px 8px;
  }
  .link-btn:hover {
    color: #3b82f6;
  }
</style>
