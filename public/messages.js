const { ReconnectingWebSocket } = window;

const websocketURL = `ws://${document.location.host}/ws`;

let messages = null;
let messageList = null;
let conn = null;
let startBtn = null;
let stopBtn = null;

function resetScroll() {
  messages.scrollTop = messages.scrollHeight;
}

function removeAllChildNodes(parent) {
  while (parent.firstChild) {
    parent.removeChild(parent.firstChild);
  }
}

// Handle Websocket messages.
// Two types of responses are returned:
//
// 1. Server output:
//      { "output" : "text that will appear in the messages-list" }
//
// 2. Process status changes:
//      { "status" : "Starting | Stopping | etc.." }
//
function handleMessage(event) {
  const data = JSON.parse(event.data);

  if (data.status !== undefined) {
    if (data.status === 'Starting' || data.status === 'Running') {
      startBtn.disabled = true;
      stopBtn.disabled = false;
    } else if (data.status === 'Stopping') {
      startBtn.disabled = true;
      stopBtn.disabled = true;
    } else {
      startBtn.disabled = false;
      stopBtn.disabled = true;
    }
  } else if (data.output !== undefined) {
    const li = document.createElement('li');

    li.appendChild(document.createTextNode(data.output));
    messageList.appendChild(li);

    resetScroll();
  }
}

function handleClose() {
  startBtn.disabled = false;
  stopBtn.disabled = false;
}

function clear() {
  removeAllChildNodes(messageList);
}

function init(startButton, stopButton) {
  startBtn = startButton;
  stopBtn = stopButton;

  messages = document.querySelector('.messages');
  messageList = document.querySelector('.message-list');

  conn = new ReconnectingWebSocket(websocketURL, null, {
    debug: false,
    reconnectInterval: 400,
  });

  conn.onmessage = handleMessage;
  conn.onclose = handleClose;

  resetScroll();
}

export { init, clear };
