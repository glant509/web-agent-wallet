chrome.runtime.onInstalled.addListener(async () => {
  await chrome.storage.session.clear();
  await chrome.sidePanel.setPanelBehavior({ openPanelOnActionClick: false });
});

chrome.runtime.onStartup.addListener(async () => {
  await chrome.storage.session.clear();
});

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (!message || message.type !== "agent-wallet:lock") {
    return false;
  }
  chrome.storage.session.clear().then(() => sendResponse({ ok: true }));
  return true;
});
