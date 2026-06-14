<script>
  import { onMount } from 'svelte';
  import { t } from './i18n.js';

  const API_URL = "http://localhost:8080";

  let config = null;
  let loading = true;
  let error = "";

  onMount(async () => {
    await loadConfig();
  });

  async function loadConfig() {
    loading = true;
    error = "";
    try {
      const res = await fetch(`${API_URL}/api/v1/donations/config`);
      if (!res.ok) throw new Error("HTTP " + res.status);
      config = await res.json();
    } catch (e) {
      error = $t('donationError') + " " + e.message;
    } finally {
      loading = false;
    }
  }

  function openPaypal(link) {
    window.open(link, '_blank');
  }

  function copyNumber(num) {
    navigator.clipboard.writeText(num);
  }
</script>

<div class="donation-panel">
  {#if loading}
    <div class="loading">{$t('donationLoading')}</div>
  {:else if error}
    <div class="error">{error}</div>
    <button class="retry-btn" on:click={loadConfig}>{$t('donationRetry')}</button>
  {:else if config}
    <h3>{$t('donationTitle')}</h3>

    <div class="amounts">
      <p>{$t('donationInvite')}</p>
      <div class="amount-buttons">
        {#each config.montos_sugeridos as monto}
          <span class="amount">{monto} {config.moneda}</span>
        {/each}
      </div>
    </div>

    <div class="methods">
      <div class="method">
        <h4>PayPal</h4>
        <button class="paypal-btn" on:click={() => openPaypal(config.paypal.link)}>
          {$t('donationPaypalBtn')}
        </button>
        <p class="detail">{config.paypal.email}</p>
      </div>

      <div class="method">
        <h4>Yape</h4>
        {#if config.yape.qr_base64}
          <img src="data:image/png;base64,{config.yape.qr_base64}" alt="QR Yape" class="qr" />
        {/if}
        <!-- <p class="detail" on:click={() => copyNumber(config.yape.numero)}>
          📋 {config.yape.numero}
        </p> -->
        <p class="detail">{config.yape.nombre}</p>
      </div>

      <div class="method">
        <h4>Plin</h4>
        {#if config.plin.qr_base64}
          <img src="data:image/png;base64,{config.plin.qr_base64}" alt="QR Plin" class="qr" />
        {/if}
        <!-- <p class="detail" on:click={() => copyNumber(config.plin.numero)}>
          📋 {config.plin.numero}
        </p> -->
        <p class="detail">{config.plin.nombre}</p>
      </div>
    </div>
  {/if}
</div>

<style>
  .donation-panel {
    padding: 10px;
    text-align: center;
  }
  h3 {
    margin: 0 0 12px 0;
    font-size: 1.1em;
    color: #ccc;
  }
  .amounts {
    margin-bottom: 20px;
  }
  .amounts p {
    margin: 0 0 8px 0;
    color: #999;
    font-size: 0.85em;
  }
  .amount-buttons {
    display: flex;
    justify-content: center;
    gap: 8px;
    flex-wrap: wrap;
  }
  .amount {
    background: #2a2d35;
    padding: 6px 14px;
    border-radius: 6px;
    font-size: 0.9em;
    color: #4fc3f7;
    border: 1px solid #3a3d45;
  }
  .methods {
    display: flex;
    gap: 16px;
    justify-content: center;
    flex-wrap: wrap;
  }
  .method {
    background: #22252b;
    border: 1px solid #33363e;
    border-radius: 10px;
    padding: 16px;
    width: 230px;
  }
  .method h4 {
    margin: 0 0 10px 0;
    color: #eee;
    font-size: 1em;
  }
  .qr {
    width: 230px;
    height: 230px;
    image-rendering: pixelated;
    background: white;
    border-radius: 6px;
    margin-bottom: 8px;
  }
  .paypal-btn {
    background: #0070ba;
    color: white;
    border: none;
    padding: 10px 20px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.9em;
    width: 100%;
  }
  .paypal-btn:hover {
    background: #005e9e;
  }
  .detail {
    margin: 4px 0;
    font-size: 0.8em;
    color: #aaa;
    cursor: pointer;
  }
  .detail:hover {
    color: #4fc3f7;
  }
  .loading, .error {
    color: #999;
    padding: 20px;
  }
  .error {
    color: #e57373;
  }
  .retry-btn {
    background: #444;
    color: #ccc;
    border: 1px solid #555;
    padding: 6px 16px;
    border-radius: 6px;
    cursor: pointer;
  }
  .retry-btn:hover {
    background: #555;
  }
</style>
