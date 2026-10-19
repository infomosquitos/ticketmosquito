import { detectCurrentIssue } from './issue-detector';

console.log('[ticketmosquito] Content script on', window.location.href);

chrome.runtime.onMessage.addListener((msg: { type: string }, _sender, sendResponse) => {
  switch (msg.type) {
    case 'get-issue-info':
      sendResponse(detectCurrentIssue());
      break;
    default:
      sendResponse({ error: 'unknown message type' });
  }
  return true;
});
