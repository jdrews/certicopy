<script lang="ts">
  import logo from './assets/images/logo-universal.png'
  // import {Greet} from '../wailsjs/go/main/App.js' // Greet is gone

  let statusText = "Ready to copy files."
</script>

<main>
  <img alt="Wails logo" src={logo} id="logo" />
  <div class="result" id="result">{statusText}</div>
  <div class="input-box" id="input">
      <h2>CertiCopy Transfer Service</h2>
      <p>Backend initialized successfully.</p>
  </div>
</main>

<style>
  #logo {
    display: block;
    width: 50%;
    height: 50%;
    margin: auto;
    padding: 10% 0 0;
    background-position: center;
    background-repeat: no-repeat;
    background-size: 100% 100%;
    background-origin: content-box;
  }

  .result {
    height: 20px;
    line-height: 20px;
    margin: 1.5rem auto;
  }

  .input-box {
      display: flex;
      flex-direction: column;
      align-items: center;
  }
</style>
