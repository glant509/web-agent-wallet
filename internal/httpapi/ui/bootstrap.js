// The page can run from the Go server or be opened directly from this folder.
    // Keep wallet cryptography local in both cases and version immutable assets so
    // a browser cannot reuse an older signer bundle after a wallet update.
    const walletScriptBase = window.location.protocol === "file:" || window.location.protocol === "chrome-extension:" ? "./" : "/ui/";
    const walletScriptVersion = "20260923-ai-toggle-v1";
    document.write('<script src="' + walletScriptBase + 'platform_runtime.js?v=' + walletScriptVersion + '"><\/script>');
    document.write('<script src="' + walletScriptBase + 'wallet_derivation.js?v=' + walletScriptVersion + '"><\/script>');
    document.write('<script src="' + walletScriptBase + 'evm_signer.js?v=' + walletScriptVersion + '"><\/script>');
  
